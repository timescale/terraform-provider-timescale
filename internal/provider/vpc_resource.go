package provider

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	tsClient "github.com/timescale/terraform-provider-timescale/internal/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &vpcResource{}
	_ resource.ResourceWithConfigure = &vpcResource{}

	ErrVPCRead   = "Error reading VPC"
	ErrVPCCreate = "Error creating VPC"
	ErrVPCUpdate = "Error updating VPC"
)

// NewVpcsResource is a helper function to simplify the provider implementation.
func NewVpcsResource() resource.Resource {
	return &vpcResource{}
}

// vpcResource is the data source implementation.
type vpcResource struct {
	client *tsClient.Client
}

type vpcResourceModel struct {
	ID            types.Int64    `tfsdk:"id"`
	ProvisionedID types.String   `tfsdk:"provisioned_id"`
	ProjectID     types.String   `tfsdk:"project_id"`
	CIDR          types.String   `tfsdk:"cidr"`
	Name          types.String   `tfsdk:"name"`
	RegionCode    types.String   `tfsdk:"region_code"`
	Status        types.String   `tfsdk:"status"`
	ErrorMessage  types.String   `tfsdk:"error_message"`
	Created       types.String   `tfsdk:"created"`
	Updated       types.String   `tfsdk:"updated"`
	Timeouts      timeouts.Value `tfsdk:"timeouts"`
}

// Metadata returns the data source type name.
func (r *vpcResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpcs"
}

// Read refreshes the Terraform state with the latest data.
func (r *vpcResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "VpcResource.Read")
	var state vpcResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var vpc *tsClient.VPC
	var err error

	if !state.Name.IsNull() {
		tflog.Info(ctx, "Getting VPC by name: "+state.Name.ValueString())
		vpc, err = r.client.GetVPCByName(ctx, state.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to Read vpc, got error: %s, %s", state.Name.ValueString(), err))
			return
		}
	} else {
		resp.Diagnostics.AddError(ErrVPCRead, "error must provide Name")
		return
	}
	vpcID, err := strconv.ParseInt(vpc.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Parse Error", "could not parse vpcID")
	}
	state.ID = types.Int64Value(vpcID)
	state.Created = types.StringValue(vpc.Created)
	state.ProjectID = types.StringValue(vpc.ProjectID)
	state.ProvisionedID = types.StringValue(vpc.ProvisionedID)
	state.CIDR = types.StringValue(vpc.CIDR)
	state.RegionCode = types.StringValue(vpc.RegionCode)

	model := vpcToResource(vpc, state)
	// Save updated plan into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, fmt.Sprintf("error updating terraform state %v", resp.Diagnostics.Errors()))
		return
	}
}

func vpcToResource(s *tsClient.VPC, state vpcResourceModel) vpcResourceModel {
	model := vpcResourceModel{
		ID:            state.ID,
		ProjectID:     state.ProjectID,
		Created:       state.Created,
		RegionCode:    state.RegionCode,
		CIDR:          state.CIDR,
		Name:          types.StringValue(s.Name),
		ProvisionedID: types.StringValue(s.ProvisionedID),
		Status:        types.StringValue(s.Status),
		ErrorMessage:  types.StringValue(s.ErrorMessage),
		Updated:       types.StringValue(s.Updated),
		Timeouts:      state.Timeouts,
	}
	return model
}

// Create creates a VPC shell.
func (r *vpcResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "VpcResource.Create")
	var plan vpcResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if plan.RegionCode.IsNull() {
		resp.Diagnostics.AddError(ErrVPCCreate, "Region code is required")
		return
	}
	if plan.CIDR.IsNull() {
		resp.Diagnostics.AddError(ErrVPCCreate, "CIDR is required")
		return
	}
	vpc, err := r.client.CreateVPC(ctx, plan.Name.ValueString(), plan.CIDR.ValueString(), plan.RegionCode.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to Create Vpc %v", plan),
			err.Error(),
		)
		return
	}
	vpcID, err := strconv.ParseInt(vpc.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Parse Error", "could not parse vpcID")
		return
	}
	vpc, err = r.waitForVPCReadiness(ctx, vpcID, plan.Timeouts)
	if err != nil {
		resp.Diagnostics.AddError("Create VPC Error", "error waiting for VPC readiness: "+err.Error())
		return
	}
	plan.ID = types.Int64Value(vpcID)
	plan.Created = types.StringValue(vpc.Created)
	plan.ProjectID = types.StringValue(vpc.ProjectID)
	plan.ProvisionedID = types.StringValue(vpc.ProvisionedID)
	plan.CIDR = types.StringValue(vpc.CIDR)
	plan.RegionCode = types.StringValue(vpc.RegionCode)
	model := vpcToResource(vpc, plan)

	// Set state
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *vpcResource) waitForVPCReadiness(ctx context.Context, id int64, timeouts timeouts.Value) (*tsClient.VPC, error) {
	tflog.Trace(ctx, "VPCResource.waitForServiceReadiness")

	defaultTimeout := 5 * time.Minute
	timeout, diags := timeouts.Create(ctx, defaultTimeout)
	if diags != nil && diags.HasError() {
		tflog.Error(ctx, fmt.Sprintf("found errs %v", diags.Errors()))
		return nil, fmt.Errorf("unable to get timeout from config %v", diags.Errors())
	}
	conf := retry.StateChangeConf{
		Pending:                   []string{"CREATING"},
		Target:                    []string{"CREATED"},
		Delay:                     5 * time.Second,
		Timeout:                   timeout,
		PollInterval:              5 * time.Second,
		ContinuousTargetOccurence: 1,
		Refresh: func() (result interface{}, state string, err error) {
			vpc, err := r.client.GetVPCByID(ctx, id)
			if err != nil {
				return nil, "", err
			}
			return vpc, vpc.Status, nil
		},
	}
	res, err := conf.WaitForStateContext(ctx)
	if err != nil {
		return nil, err
	}
	vpc, ok := res.(*tsClient.VPC)
	if !ok {
		return nil, fmt.Errorf("unexpected type found, expected VPC but got %T", res)
	}
	return vpc, nil
}

// Delete deletes a VPC shell.
func (r *vpcResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "VpcsResource.Delete")
	var state vpcResourceModel
	// TODO: Workaround to avoid deadlocks when many resources try to delete at once
	time.Sleep(10 * time.Second)
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Deleting Vpc: %v", state.ID.ValueInt64()))

	err := r.client.DeleteVPC(ctx, state.ID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Timescale Vpc",
			"Could not delete vpc, unexpected error: "+err.Error(),
		)
		return
	}
}

// Update updates a VPC shell.
func (r *vpcResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "VpcsResource.Update")
	var plan, state vpcResourceModel
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.RegionCode != state.RegionCode {
		resp.Diagnostics.AddError(ErrVPCUpdate, "Do not support region code change")
		return
	}
	if plan.CIDR != state.CIDR {
		resp.Diagnostics.AddError(ErrVPCUpdate, "Do not support cidr change")
		return
	}

	if !plan.Name.Equal(state.Name) {
		if err := r.client.RenameVPC(ctx, state.ID.ValueInt64(), plan.Name.ValueString()); err != nil {
			resp.Diagnostics.AddError(ErrVPCUpdate, err.Error())
			return
		}
		state.Name = plan.Name
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *vpcResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

// Configure adds the provider configured client to the data source.
func (r *vpcResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	tflog.Trace(ctx, "vpcsResource.Configure")
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*tsClient.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *tsClient.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.client = client
}

// Schema defines the schema for the data source.
func (r *vpcResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Schema for a VPC. Import can be done using your VPCs name`,
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"provisioned_id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cidr": schema.StringAttribute{
				Description:         `The IPv4 CIDR block`,
				MarkdownDescription: "The IPv4 CIDR block",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "VPC Name is the configurable name assigned to this vpc. If none is provided, a default will be generated by the provider.",
				Description:         "Vpc name",
				Optional:            true,
				// If the name attribute is absent, the provider will generate a default.
				Computed: true,
			},
			"region_code": schema.StringAttribute{
				Description:         `The region for this VPC`,
				MarkdownDescription: "The region for this VPC.",
				Required:            true,
			},
			"status": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"error_message": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"timeouts": timeoutSchema(ctx, timeouts.Opts{
				Create: true,
			}),
		},
	}
}

// adding timeouts can break earlier versions of the provider since the field is not set.
// reference: https://github.com/hashicorp/terraform-plugin-framework-timeouts/issues/49#issuecomment-1511027690
func timeoutSchema(ctx context.Context, opts timeouts.Opts) schema.SingleNestedAttribute {
	timeout, _ := timeouts.Attributes(ctx, opts).(schema.SingleNestedAttribute)
	at := map[string]attr.Type{}
	if opts.Create {
		at["create"] = types.StringType
	}
	if opts.Read {
		at["read"] = types.StringType
	}
	if opts.Update {
		at["update"] = types.StringType
	}
	if opts.Delete {
		at["delete"] = types.StringType
	}
	timeout.Computed = true
	timeout.Default = objectdefault.StaticValue(
		types.ObjectNull(at),
	)
	return timeout
}
