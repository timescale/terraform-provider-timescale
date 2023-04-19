package provider

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	tsClient "github.com/timescale/terraform-provider-timescale/internal/client"
	multiplyvalidator "github.com/timescale/terraform-provider-timescale/internal/utils"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ resource.Resource = &ServiceResource{}
var _ resource.ResourceWithImportState = &ServiceResource{}

const (
	ErrCreateTimeout    = "Error waiting for service creation"
	ErrUpdateService    = "Error updating service"
	ErrInvalidAttribute = "Invalid Attribute Value"

	DefaultMilliCPU = 500
	DefaultMemoryGB = 2

	DefaultEnableHAReplica = false
)

var (
	memorySizes   = []int64{2, 4, 8, 16, 32, 64, 128}
	milliCPUSizes = []int64{500, 1000, 2000, 4000, 8000, 16000, 32000}
)

func NewServiceResource() resource.Resource {
	return &ServiceResource{}
}

// ServiceResource defines the resource implementation.
type ServiceResource struct {
	client *tsClient.Client
}

// serviceResourceModel maps the resource schema data.
type serviceResourceModel struct {
	ID              types.String   `tfsdk:"id"`
	Name            types.String   `tfsdk:"name"`
	Timeouts        timeouts.Value `tfsdk:"timeouts"`
	MilliCPU        types.Int64    `tfsdk:"milli_cpu"`
	StorageGB       types.Int64    `tfsdk:"storage_gb"`
	MemoryGB        types.Int64    `tfsdk:"memory_gb"`
	Password        types.String   `tfsdk:"password"`
	Hostname        types.String   `tfsdk:"hostname"`
	Port            types.Int64    `tfsdk:"port"`
	Username        types.String   `tfsdk:"username"`
	RegionCode      types.String   `tfsdk:"region_code"`
	EnableHAReplica types.Bool     `tfsdk:"enable_ha_replica"`
	VpcId           types.Int64    `tfsdk:"vpc_id"`
}

func (r *ServiceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	tflog.Trace(ctx, "ServiceResource.Metadata")
	resp.TypeName = req.ProviderTypeName + "_service"
}

// Schema defines the schema for the service resource.
func (r *ServiceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	tflog.Trace(ctx, "ServiceResource.Schema")
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: `A Service is a TimescaleDB instance.

Please note that when updating the vpc_id attribute, it is possible to encounter a "no Endpoint for that service id exists" error. 
The change has been taken into account but must still be propagated. You can run "terraform refresh" shortly to get the updated data.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Service ID is the unique identifier for this service.",
				Description:         "service id",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Service Name is the configurable name assigned to this resource. If none is provided, a default will be generated by the provider.",
				Description:         "service name",
				Optional:            true,
				// If the name attribute is absent, the provider will generate a default.
				Computed: true,
			},
			"milli_cpu": schema.Int64Attribute{
				MarkdownDescription: "Milli CPU",
				Description:         "Milli CPU",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(DefaultMilliCPU),
				Validators: []validator.Int64{
					int64validator.OneOf(milliCPUSizes...),
					multiplyvalidator.EqualToMultipleOf(250, path.Expressions{
						path.MatchRoot("memory_gb"),
					}...),
				},
			},
			"enable_ha_replica": schema.BoolAttribute{
				MarkdownDescription: "Enable HA Replica",
				Description:         "Enable HA Replica",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(DefaultEnableHAReplica),
			},
			"storage_gb": schema.Int64Attribute{
				MarkdownDescription: "Deprecated: Storage GB",
				Description:         "Deprecated: Storage GB",
				Optional:            true,
				DeprecationMessage:  "This field is ignored. With the new usage-based storage Timescale automatically allocates the disk space needed by your service and you only pay for the disk space you use.",
			},
			"memory_gb": schema.Int64Attribute{
				MarkdownDescription: "Memory GB",
				Description:         "Memory GB",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(DefaultMemoryGB),
				Validators:          []validator.Int64{int64validator.OneOf(memorySizes...)},
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true,
			}),
			"password": schema.StringAttribute{
				Description:         "The Postgres password for this service. The password is provided once during service creation",
				MarkdownDescription: "The Postgres password for this service. The password is provided once during service creation",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"hostname": schema.StringAttribute{
				Description:         "The hostname for this service",
				MarkdownDescription: "The hostname for this service",
				Computed:            true,
			},
			"port": schema.Int64Attribute{
				Description:         "The port for this service",
				MarkdownDescription: "The port for this service",
				Computed:            true,
			},
			"username": schema.StringAttribute{
				Description:         "The Postgres user for this service",
				MarkdownDescription: "The Postgres user for this service",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"region_code": schema.StringAttribute{
				Description:         `The region for this service`,
				MarkdownDescription: "The region for this service.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vpc_id": schema.Int64Attribute{
				Description:         `The VpcID this service is tied to.`,
				MarkdownDescription: `The VpcID this service is tied to.`,
				Optional:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Configure adds the provider configured client to the service resource.
func (r *ServiceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	tflog.Trace(ctx, "ServiceResource.Configure")
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

func (r *ServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "ServiceResource.Create")
	var plan serviceResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}
	// Enabling HA replica means one replica
	var replicaCount int64
	if plan.EnableHAReplica.ValueBool() {
		replicaCount = 1
	}

	request := tsClient.CreateServiceRequest{
		Name:         plan.Name.ValueString(),
		MilliCPU:     strconv.FormatInt(plan.MilliCPU.ValueInt64(), 10),
		MemoryGB:     strconv.FormatInt(plan.MemoryGB.ValueInt64(), 10),
		RegionCode:   plan.RegionCode.ValueString(),
		ReplicaCount: strconv.FormatInt(replicaCount, 10),
	}
	if !plan.VpcId.IsNull() {
		request.VpcID = plan.VpcId.ValueInt64()
	}
	response, err := r.client.CreateService(ctx, request)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create service, got error: %s", err))
		return
	}

	plan.Password = types.StringValue(response.InitialPassword)
	service, err := r.waitForServiceReadiness(ctx, response.Service.ID, plan.Timeouts)
	if err != nil {
		resp.Diagnostics.AddError(ErrCreateTimeout, fmt.Sprintf("error occured while waiting for service deployment, got error: %s", err))
		// If we receive an error, attempt to delete the service to avoid having an orphaned instance.
		_, err = r.client.DeleteService(context.Background(), response.Service.ID)
		if err != nil {
			resp.Diagnostics.AddWarning("Error Deleting Resource", "error occurred attempting to delete the resource that timed out, please check your Timescale account to verify there is no unexpected service running from Terraform")
		}
		return
	}
	resourceModel := serviceToResource(resp.Diagnostics, service, plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, resourceModel)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, fmt.Sprintf("error updating terraform state %v", resp.Diagnostics.Errors()))
		return
	}
}

func (r *ServiceResource) waitForServiceReadiness(ctx context.Context, ID string, timeouts timeouts.Value) (*tsClient.Service, error) {
	tflog.Trace(ctx, "ServiceResource.waitForServiceReadiness")

	defaultTimeout := 45 * time.Minute
	timeout, diags := timeouts.Create(ctx, defaultTimeout)
	if diags != nil && diags.HasError() {
		tflog.Error(ctx, fmt.Sprintf("found errs %v", diags.Errors()))
		return nil, fmt.Errorf("unable to get timeout from config %v", diags.Errors())
	}

	conf := retry.StateChangeConf{
		Pending:                   []string{"QUEUED", "CONFIGURING", "UNSTABLE"},
		Target:                    []string{"READY"},
		Delay:                     10 * time.Second,
		Timeout:                   timeout,
		PollInterval:              5 * time.Second,
		ContinuousTargetOccurence: 1,
		Refresh: func() (result interface{}, state string, err error) {
			s, err := r.client.GetService(ctx, ID)
			if err != nil {
				return nil, "", err
			}
			return s, s.Status, nil
		},
	}
	result, err := conf.WaitForStateContext(ctx)
	if err != nil {
		return nil, err
	}
	s, ok := result.(*tsClient.Service)
	if !ok {
		return nil, fmt.Errorf("unexpected type found, expected Service but got %T", result)
	}
	return s, nil
}

func (r *ServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "ServiceResource.Read")
	var state serviceResourceModel
	// Read Terraform prior state plan into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Getting Service: "+state.ID.ValueString())

	service, err := r.client.GetService(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service, got error: %s", err))
		return
	}
	resourceModel := serviceToResource(resp.Diagnostics, service, state)
	// Save updated plan into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, resourceModel)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, fmt.Sprintf("error updating terraform state %v", resp.Diagnostics.Errors()))
		return
	}
}

func (r *ServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "ServiceResource.Update")
	var plan, state serviceResourceModel
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceID := state.ID.ValueString()

	if !plan.Hostname.IsUnknown() {
		resp.Diagnostics.AddError(ErrUpdateService, "Do not support hostname change")
		return
	}

	if plan.Username != state.Username {
		resp.Diagnostics.AddError(ErrUpdateService, "Do not support username change")
		return
	}

	if !plan.Port.IsUnknown() {
		resp.Diagnostics.AddError(ErrUpdateService, "Do not support port change")
		return
	}

	if plan.RegionCode != state.RegionCode {
		resp.Diagnostics.AddError(ErrUpdateService, "Do not support region code change")
		return
	}

	if plan.EnableHAReplica != state.EnableHAReplica {
		if plan.EnableHAReplica.ValueBool() {
			if err := r.client.SetReplicaCount(ctx, serviceID, 1); err != nil {
				resp.Diagnostics.AddError("Failed to add a HA replica", err.Error())
				return
			}
		} else {
			if err := r.client.SetReplicaCount(ctx, serviceID, 0); err != nil {
				resp.Diagnostics.AddError("Failed to remove a HA replica", err.Error())
				return
			}
		}

	}

	if !plan.VpcId.Equal(state.VpcId) {
		// if state.VpcId is known and different from plan.VpcId, we must detach first
		if !state.VpcId.IsNull() && !state.VpcId.IsUnknown() {
			if err := r.client.DetachServiceFromVpc(ctx, serviceID, state.VpcId.ValueInt64()); err != nil {
				resp.Diagnostics.AddError("Failed to detach service from VPC", err.Error())
				return
			}
		}
		// if plan.VpcId is known, it must be attached
		if !plan.VpcId.IsNull() && !plan.VpcId.IsUnknown() {
			if err := r.client.AttachServiceToVpc(ctx, serviceID, plan.VpcId.ValueInt64()); err != nil {
				resp.Diagnostics.AddError("Failed to attach service to VPC", err.Error())
				return
			}
		}
	}

	if !plan.Name.Equal(state.Name) {
		if err := r.client.RenameService(ctx, serviceID, plan.Name.ValueString()); err != nil {
			resp.Diagnostics.AddError("Failed to rename a service", err.Error())
			return
		}
	}

	{
		isResizeRequested := false
		const noop = "0" // Compute and storage could be resized separately. Setting value to 0 means a no-op.
		resizeConfig := tsClient.ResourceConfig{
			MilliCPU: noop,
			MemoryGB: noop,
		}

		if !plan.MilliCPU.Equal(state.MilliCPU) || !plan.MemoryGB.Equal(state.MemoryGB) {
			isResizeRequested = true
			resizeConfig.MilliCPU = strconv.FormatInt(plan.MilliCPU.ValueInt64(), 10)
			resizeConfig.MemoryGB = strconv.FormatInt(plan.MemoryGB.ValueInt64(), 10)
		}

		if isResizeRequested {
			if err := r.client.ResizeInstance(ctx, serviceID, resizeConfig); err != nil {
				resp.Diagnostics.AddError("Failed to resize an instance", err.Error())
				return
			}
		}
	}

	service, err := r.waitForServiceReadiness(ctx, serviceID, plan.Timeouts)
	if err != nil {
		resp.Diagnostics.AddError(ErrCreateTimeout, fmt.Sprintf("error occured while waiting for service reconfiguration, got error: %s", err))
		return
	}
	resources := serviceToResource(resp.Diagnostics, service, plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, resources)...)

	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, fmt.Sprintf("error updating terraform state %v", resp.Diagnostics.Errors()))
		return
	}
}

func (r *ServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "ServiceResource.Delete")
	var data serviceResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Deleting Service: "+data.ID.ValueString())

	_, err := r.client.DeleteService(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Timescale Service",
			"Could not delete order, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *ServiceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func serviceToResource(diag diag.Diagnostics, s *tsClient.Service, state serviceResourceModel) serviceResourceModel {
	model := serviceResourceModel{
		ID:              types.StringValue(s.ID),
		Password:        state.Password,
		Name:            types.StringValue(s.Name),
		MilliCPU:        types.Int64Value(s.Resources[0].Spec.MilliCPU),
		MemoryGB:        types.Int64Value(s.Resources[0].Spec.MemoryGB),
		Hostname:        types.StringValue(s.ServiceSpec.Hostname),
		Username:        types.StringValue(s.ServiceSpec.Username),
		Port:            types.Int64Value(s.ServiceSpec.Port),
		RegionCode:      types.StringValue(s.RegionCode),
		Timeouts:        state.Timeouts,
		EnableHAReplica: types.BoolValue(s.ReplicaStatus != ""),
	}
	if s.VpcEndpoint != nil {
		if vpcId, err := strconv.ParseInt(s.VpcEndpoint.VpcId, 10, 64); err != nil {
			diag.AddError("Parse Error", "could not parse vpcID")
		} else {
			model.VpcId = types.Int64Value(vpcId)
		}
		model.Hostname = types.StringValue(s.VpcEndpoint.Host)
		model.Port = types.Int64Value(s.VpcEndpoint.Port)
	}

	return model
}
