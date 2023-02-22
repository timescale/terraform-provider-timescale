package provider

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	helper "github.com/hashicorp/terraform-plugin-testing/helper/resource"

	tsClient "github.com/timescale/terraform-provider-timescale/internal/client"
	multiplyvalidator "github.com/timescale/terraform-provider-timescale/internal/utils"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ resource.Resource = &ServiceResource{}

const (
	ErrCreateTimeout    = "Error waiting for service creation"
	ErrUpdateService    = "Updating service name is currently unsupported"
	ErrInvalidAttribute = "Invalid Attribute Value"

	DefaultMilliCPU  = 500
	DefaultStorageGB = 10
	DefaultMemoryGB  = 2
)

var (
	storageSizes  = []int64{10, 25, 50, 75, 100, 125, 150, 175, 200, 225, 250, 275, 300, 325, 350, 375, 400, 425, 450, 475, 500, 600, 700, 800, 900, 1000, 1500, 2000, 2500, 3000, 4000, 5000, 6000, 7000, 800, 9000, 10000, 12000, 14000, 16000}
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
	ID                       types.String   `tfsdk:"id"`
	Name                     types.String   `tfsdk:"name"`
	EnableStorageAutoscaling types.Bool     `tfsdk:"enable_storage_autoscaling"`
	Timeouts                 timeouts.Value `tfsdk:"timeouts"`
	MilliCPU                 types.Int64    `tfsdk:"milli_cpu"`
	StorageGB                types.Int64    `tfsdk:"storage_gb"`
	MemoryGB                 types.Int64    `tfsdk:"memory_gb"`
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
		// TODO: decide if we will use generated docs. If so, write complete markdown descriptions.
		MarkdownDescription: "Service Description",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Service ID",
				Description:         "service id",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Service Name",
				Description:         "service name",
				Optional:            true,
				// If the name attribute is absent, the provider will generate a default.
				Computed: true,
			},
			"enable_storage_autoscaling": schema.BoolAttribute{
				MarkdownDescription: "Enable Storage Autoscaling",
				Description:         "Flag to enable storage autoscaling",
				Optional:            true,
				Computed:            true,
			},
			"milli_cpu": schema.Int64Attribute{
				MarkdownDescription: "Milli CPU",
				Description:         "Milli CPU",
				Optional:            true,
				Computed:            true,
				Validators: []validator.Int64{
					int64validator.OneOf(milliCPUSizes...),
					multiplyvalidator.EqualToMultipleOf(250, path.Expressions{
						path.MatchRoot("memory_gb"),
					}...),
				},
			},
			"storage_gb": schema.Int64Attribute{
				MarkdownDescription: "Storage GB",
				Description:         "Storage GB",
				Optional:            true,
				Computed:            true,
				Validators:          []validator.Int64{int64validator.OneOf(storageSizes...)},
			},
			"memory_gb": schema.Int64Attribute{
				MarkdownDescription: "Memory GB",
				Description:         "Memory GB",
				Optional:            true,
				Computed:            true,
				Validators:          []validator.Int64{int64validator.OneOf(memorySizes...)},
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true,
			}),
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
	var data serviceResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	service, err := r.client.CreateService(ctx, tsClient.CreateServiceRequest{
		Name:                     data.Name.ValueString(),
		EnableStorageAutoscaling: data.EnableStorageAutoscaling.ValueBool(),
		MilliCPU:                 useDefaultIfEmpty(data.MilliCPU, DefaultMilliCPU),
		StorageGB:                useDefaultIfEmpty(data.StorageGB, DefaultStorageGB),
		MemoryGB:                 useDefaultIfEmpty(data.MemoryGB, DefaultMemoryGB),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create service, got error: %s", err))
		return
	}

	state, err := r.waitForServiceReadiness(ctx, service.ID, data.Timeouts)
	if err != nil {
		resp.Diagnostics.AddError(ErrCreateTimeout, fmt.Sprintf("error occured while waiting for service deployment, got error: %s", err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, fmt.Sprintf("error updating terraform state %v", resp.Diagnostics.Errors()))
		return
	}
}

func useDefaultIfEmpty(value basetypes.Int64Value, defaultValue int64) string {
	if !value.IsUnknown() {
		return strconv.FormatInt(value.ValueInt64(), 10)
	}
	return strconv.FormatInt(defaultValue, 10)
}

func (r *ServiceResource) waitForServiceReadiness(ctx context.Context, ID string, timeouts timeouts.Value) (serviceResourceModel, error) {
	tflog.Trace(ctx, "ServiceResource.waitForServiceReadiness")

	defaultTimeout := 45 * time.Minute
	timeout, diags := timeouts.Create(ctx, defaultTimeout)
	if diags != nil && diags.HasError() {
		tflog.Error(ctx, fmt.Sprintf("found errs %v", diags.Errors()))
		return serviceResourceModel{}, fmt.Errorf("unable to get timeout from config %v", diags.Errors())
	}

	conf := helper.StateChangeConf{
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
	s, err := conf.WaitForStateContext(ctx)
	if err != nil {
		return serviceResourceModel{}, err
	}
	service := s.(*tsClient.Service)
	state := serviceToResource(service, timeouts)
	return state, nil
}

func (r *ServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "ServiceResource.Read")
	var plan serviceResourceModel
	// Read Terraform prior state plan into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Getting Service: "+plan.ID.ValueString())

	service, err := r.client.GetService(ctx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service, got error: %s", err))
		return
	}
	state := serviceToResource(service, plan.Timeouts)
	// Save updated plan into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
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

	if !plan.Name.Equal(state.ID) {
		resp.Diagnostics.AddError("Unsupported operation", ErrUpdateService)
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

func serviceToResource(s *tsClient.Service, timeouts timeouts.Value) serviceResourceModel {
	return serviceResourceModel{
		ID:                       types.StringValue(s.ID),
		Name:                     types.StringValue(s.Name),
		EnableStorageAutoscaling: types.BoolValue(s.AutoscaleSettings.Enabled),
		MilliCPU:                 types.Int64Value(s.Resources[0].Spec.MilliCPU),
		StorageGB:                types.Int64Value(s.Resources[0].Spec.StorageGB),
		MemoryGB:                 types.Int64Value(s.Resources[0].Spec.MemoryGB),
		Timeouts:                 timeouts,
	}
}
