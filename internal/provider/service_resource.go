package provider

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
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

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &serviceResource{}
var _ resource.ResourceWithImportState = &serviceResource{}

const (
	ErrCreateTimeout            = "Error waiting for service creation"
	ErrUpdateService            = "Error updating service"
	ErrInvalidAttribute         = "Invalid Attribute Value"
	errReplicaFromFork          = "cannot create a read replica from a read replica or fork"
	errReplicaWithHA            = "cannot create a read replica with HA enabled"
	errUpdateReplicaSource      = "cannot update read replica source"
	errAttachExporter           = "error attaching exporter to service"
	errDetachExporter           = "error detaching exporter form service"
	errHAFieldConflict          = "cannot set enable_ha_replica as false together with ha_replicas > 0"
	errHAFieldConflict2         = "cannot set enable_ha_replica as true together with ha_replicas = 0"
	errSyncReplicaInvalidConfig = "sync_replicas can only be 1 when ha_replicas = 2"
	DefaultMilliCPU             = 500
	DefaultMemoryGB             = 2
)

var (
	memorySizes   = []int64{2, 4, 8, 16, 32, 64, 128}
	milliCPUSizes = []int64{500, 1000, 2000, 4000, 8000, 16000, 32000}
)

func NewServiceResource() resource.Resource {
	return &serviceResource{}
}

// serviceResource defines the resource implementation.
type serviceResource struct {
	client *tsClient.Client
}

// serviceResourceModel maps the resource schema data.
type serviceResourceModel struct {
	ID                          types.String   `tfsdk:"id"`
	Name                        types.String   `tfsdk:"name"`
	Timeouts                    timeouts.Value `tfsdk:"timeouts"`
	MilliCPU                    types.Int64    `tfsdk:"milli_cpu"`
	StorageGB                   types.Int64    `tfsdk:"storage_gb"`
	MemoryGB                    types.Int64    `tfsdk:"memory_gb"`
	Password                    types.String   `tfsdk:"password"`
	Hostname                    types.String   `tfsdk:"hostname"`
	Port                        types.Int64    `tfsdk:"port"`
	ReplicaHostname             types.String   `tfsdk:"replica_hostname"`
	ReplicaPort                 types.Int64    `tfsdk:"replica_port"`
	PoolerHostname              types.String   `tfsdk:"pooler_hostname"`
	PoolerPort                  types.Int64    `tfsdk:"pooler_port"`
	Username                    types.String   `tfsdk:"username"`
	RegionCode                  types.String   `tfsdk:"region_code"`
	EnableHAReplica             types.Bool     `tfsdk:"enable_ha_replica"`
	HAReplicas                  types.Int64    `tfsdk:"ha_replicas"`
	SyncReplicas                types.Int64    `tfsdk:"sync_replicas"`
	Paused                      types.Bool     `tfsdk:"paused"`
	ReadReplicaSource           types.String   `tfsdk:"read_replica_source"`
	VpcID                       types.Int64    `tfsdk:"vpc_id"`
	ConnectionPoolerEnabled     types.Bool     `tfsdk:"connection_pooler_enabled"`
	EnvironmentTag              types.String   `tfsdk:"environment_tag"`
	MetricExporterID            types.String   `tfsdk:"metric_exporter_id"`
	LogExporterID               types.String   `tfsdk:"log_exporter_id"`
	PrivateEndpointConnectionID types.String   `tfsdk:"private_endpoint_connection_id"`
}

func (r *serviceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	tflog.Trace(ctx, "ServiceResource.Metadata")
	resp.TypeName = req.ProviderTypeName + "_service"
}

// Schema defines the schema for the service resource.
func (r *serviceResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	tflog.Trace(ctx, "ServiceResource.Schema")
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: `A Service is a TimescaleDB instance.
Supported providers: AWS and Azure, though there's no VPC support yet on Azure.
AWS and Azure region naming is official, with the caveat that Azure regions
have the prefix "az-" before, such as az-eastus2 or az-westeurope.

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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
				MarkdownDescription: "Enable HA Replica (deprecated - use ha_replicas and sync_replicas instead)",
				Description:         "Enable HA Replica (deprecated - use ha_replicas and sync_replicas instead)",
				Optional:            true,
				DeprecationMessage:  "This field is deprecated. Use ha_replicas and sync_replicas fields instead for more granular HA configuration.",
			},
			"ha_replicas": schema.Int64Attribute{
				MarkdownDescription: "Number of HA replicas (0, 1 or 2). Modes: 1 for 'High availability'; 2 'Highest availability'. Async replicas (i.e. 'High performance' mode) will be created by default if sync_replicas is not set.",
				Description:         "Number of HA replicas (0, 1 or 2). Modes: 1 for 'High availability'; 2 'Highest availability'. Async replicas (i.e. 'High performance' mode) will be created by default if sync_replicas is not set.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.Int64{
					int64validator.Between(0, 2),
				},
			},
			"sync_replicas": schema.Int64Attribute{
				MarkdownDescription: "Number of synchronous replicas (0 or 1). Set to 1 to enable 'High data integrity mode' (1 Sync and 1 Async replicas). To set sync_replicas to 1, you must also set ha_replicas to 2.",
				Description:         "Number of synchronous replicas (0 or 1). Set to 1 to enable 'High data integrity mode' (1 Sync and 1 Async replicas). To set sync_replicas to 1, you must also set ha_replicas to 2.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.Int64{
					int64validator.Between(0, 1),
				},
			},
			"read_replica_source": schema.StringAttribute{
				MarkdownDescription: "If set, this database will be a read replica of the provided source database. The region must be the same as the source, or if omitted will be handled by the provider",
				Description:         "If set, this database will be a read replica of the provided source database. The region must be the same as the source, or if omitted will be handled by the provider",
				Optional:            true,
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
				Description:         "The Postgres password for this service. For read replicas, the password is synchronized with the parent service. If not explicitly set for a read replica, it will be null in the state. To maintain the password in state, set this attribute to match the parent service's password.",
				MarkdownDescription: "The Postgres password for this service. **Note for read replicas:** Read replicas automatically synchronize their password with the parent service. If not explicitly set for a read replica, the password will be `null` in the Terraform state. To maintain the password in state, set this attribute to match the parent service's password.",
				Optional:            true,
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
			"replica_hostname": schema.StringAttribute{
				MarkdownDescription: "Hostname of the HA-Replica of this service.",
				Description:         "Hostname of the HA-Replica of this service.",
				Computed:            true,
			},
			"replica_port": schema.Int64Attribute{
				MarkdownDescription: "Port of the HA-Replica of this service.",
				Description:         "Port of the HA-Replica of this service.",
				Computed:            true,
			},
			"pooler_hostname": schema.StringAttribute{
				MarkdownDescription: "Hostname of the pooler of this service.",
				Description:         "Hostname of the pooler of this service.",
				Computed:            true,
			},
			"pooler_port": schema.Int64Attribute{
				MarkdownDescription: "Port of the pooler of this service.",
				Description:         "Port of the pooler of this service.",
				Computed:            true,
			},
			"connection_pooler_enabled": schema.BoolAttribute{
				MarkdownDescription: "Set connection pooler status for this service.",
				Description:         "Set connection pooler status for this service.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"environment_tag": schema.StringAttribute{
				MarkdownDescription: "Set environment tag for this service.",
				Description:         "Set environment tag for this service.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{stringvalidator.OneOf("DEV", "PROD")},
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
				Description:         `The VpcID this service is tied to, only supported in AWS for now.`,
				MarkdownDescription: `The VpcID this service is tied to, only supported in AWS for now.`,
				Optional:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"paused": schema.BoolAttribute{
				Description:         `Paused status of the service.`,
				MarkdownDescription: `Paused status of the service.`,
				Default:             booldefault.StaticBool(false),
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"metric_exporter_id": schema.StringAttribute{
				Description:         "The Exporter ID attached to this service, only supported in AWS for now",
				MarkdownDescription: "The Exporter ID attached to this service, only supported in AWS for now",
				Optional:            true,
			},
			"log_exporter_id": schema.StringAttribute{
				Description: `The Log Exporter ID attached to this service, only supported in AWS for now.
				WARNING: To complete the logs exporter attachment, a service restart is required.`,
				MarkdownDescription: `The Log Exporter ID attached to this service, only supported in AWS for now.
				WARNING: To complete the logs exporter attachment, a service restart is required.`,
				Optional: true,
			},
			"private_endpoint_connection_id": schema.StringAttribute{
				Description:         "The Private Endpoint Connection ID to attach this service to (Azure Private Link).",
				MarkdownDescription: "The Private Endpoint Connection ID to attach this service to (Azure Private Link).",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Configure adds the provider configured client to the service resource.
func (r *serviceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func validateHAConfiguration(plan serviceResourceModel) error {
	haReplicas := plan.HAReplicas.ValueInt64()
	syncReplicas := plan.SyncReplicas.ValueInt64()

	if !plan.EnableHAReplica.IsNull() && !plan.EnableHAReplica.ValueBool() && haReplicas > 0 {
		return errors.New(errHAFieldConflict)
	}

	if !plan.EnableHAReplica.IsNull() && plan.EnableHAReplica.ValueBool() && haReplicas == 0 {
		return errors.New(errHAFieldConflict2)
	}

	if syncReplicas == 1 && haReplicas != 2 {
		return errors.New(errSyncReplicaInvalidConfig)
	}

	return nil
}

func (r *serviceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "ServiceResource.Create")
	var plan serviceResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if err := validateHAConfiguration(plan); err != nil {
		resp.Diagnostics.AddError(ErrInvalidAttribute, err.Error())
		return
	}

	var replicaCount, syncReplicaCount int64
	if !plan.HAReplicas.IsNull() {
		// Use the new ha_replicas field
		replicaCount = plan.HAReplicas.ValueInt64()
		syncReplicaCount = plan.SyncReplicas.ValueInt64()
	} else if !plan.EnableHAReplica.IsNull() && plan.EnableHAReplica.ValueBool() {
		// Fall back to the deprecated enable_ha_replica field (default to 1 replica sync mode)
		replicaCount = 1
		syncReplicaCount = 0
	}

	request := tsClient.CreateServiceRequest{
		Name:                   plan.Name.ValueString(),
		MilliCPU:               strconv.FormatInt(plan.MilliCPU.ValueInt64(), 10),
		MemoryGB:               strconv.FormatInt(plan.MemoryGB.ValueInt64(), 10),
		RegionCode:             plan.RegionCode.ValueString(),
		ReplicaCount:           strconv.FormatInt(replicaCount, 10),
		SyncReplicaCount:       strconv.FormatInt(syncReplicaCount, 10),
		EnableConnectionPooler: plan.ConnectionPoolerEnabled.ValueBool(),
		EnvironmentTag:         plan.EnvironmentTag.ValueString(),
	}

	readReplicaSource := plan.ReadReplicaSource.ValueString()
	if readReplicaSource != "" {
		primary, err := r.client.GetService(ctx, readReplicaSource)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get primary service %s, got error: %s", readReplicaSource, err))
			return
		}
		err = r.validateCreateReadReplicaRequest(ctx, primary, plan)
		if err != nil {
			resp.Diagnostics.AddError("read replica validation error", err.Error())
			return
		}
		if request.Name == "" {
			request.Name = "replica-" + primary.Name
		}
		if request.RegionCode == "" {
			request.RegionCode = primary.RegionCode
		}
		request.ForkConfig = &tsClient.ForkConfig{
			ProjectID: primary.ProjectID,
			ServiceID: primary.ID,
			IsStandby: true,
		}
		if len(primary.Resources) > 0 {
			request.StorageGB = strconv.FormatInt(primary.Resources[0].Spec.StorageGB, 10)
		}
	}

	if !plan.VpcID.IsNull() {
		request.VpcID = plan.VpcID.ValueInt64()
	}

	var response *tsClient.CreateServiceResponse
	var err error

	// If creating a read replica, retry on backup availability errors
	if readReplicaSource != "" {
		response, err = r.createReadReplicaWithRetry(ctx, request)
	} else {
		response, err = r.client.CreateService(ctx, request)
	}

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create service, got error: %s", err))
		return
	}

	// Set the password to the initial password if not provided by the user
	if plan.Password.IsNull() || plan.Password.IsUnknown() {
		if readReplicaSource != "" {
			// Read replicas synchronize their password with the parent service
			// The API returns a fake password that doesn't match reality
			// Store null since we cannot know the actual password
			// Users should explicitly set the password attribute to match the parent
			plan.Password = types.StringNull()
		} else {
			plan.Password = types.StringValue(response.InitialPassword)
		}
	}
	service, err := r.waitForServiceReadiness(ctx, response.Service.ID, plan.Timeouts)
	if err != nil {
		resp.Diagnostics.AddError(ErrCreateTimeout, fmt.Sprintf("error occurred while waiting for service deployment, got error: %s", err))
		// If we receive an error, attempt to delete the service to avoid having an orphaned instance.
		_, err = r.client.DeleteService(context.Background(), response.Service.ID)
		if err != nil {
			resp.Diagnostics.AddWarning("Error Deleting Resource", "error occurred attempting to delete the resource that timed out, please check your Timescale account to verify there is no unexpected service running from Terraform")
		}
		return
	}

	// Check if a user-specified password was provided and update it if so, but only if the service is not a read replica
	if !plan.Password.IsNull() && plan.Password.ValueString() != response.InitialPassword && readReplicaSource == "" {
		err = r.client.ResetServicePassword(ctx, service.ID, plan.Password.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Setting the password failed", fmt.Sprintf("Unable to set user configured password, got error: %s", err))

			// Attempt to delete the service to avoid leaving an instance in an inconsistent state
			_, deleteErr := r.client.DeleteService(context.Background(), service.ID)
			if deleteErr != nil {
				resp.Diagnostics.AddWarning("Error Deleting Resource", fmt.Sprintf("Failed to delete service after password setting error; Remove orphaned resources from your account manually. Error: %s", deleteErr))
			}
			return
		}
	}

	// Exporters
	if !plan.MetricExporterID.IsNull() {
		err := r.client.AttachMetricExporter(ctx, service.ID, plan.MetricExporterID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(errAttachExporter, err.Error())
			return
		}
		service, err = r.client.GetService(ctx, service.ID)
		if err != nil {
			resp.Diagnostics.AddError(errAttachExporter, "unable to refresh service after attaching exporter")
			return
		}
	}

	if !plan.LogExporterID.IsNull() {
		err := r.client.AttachGenericExporter(ctx, service.ID, plan.LogExporterID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(errAttachExporter, err.Error())
			return
		}
		service, err = r.client.GetService(ctx, service.ID)
		if err != nil {
			resp.Diagnostics.AddError(errAttachExporter, "unable to refresh service after attaching exporter")
			return
		}
	}

	// Private Link
	if !plan.PrivateEndpointConnectionID.IsNull() {
		err := r.client.AttachServiceToPrivateLink(ctx, service.ID, plan.PrivateEndpointConnectionID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("error attaching service to private link", err.Error())
			return
		}
		service, err = r.client.GetService(ctx, service.ID)
		if err != nil {
			resp.Diagnostics.AddError("error refreshing service after attaching to private link", err.Error())
			return
		}
	}

	resourceModel := serviceToResource(resp.Diagnostics, service, plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, resourceModel)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, fmt.Sprintf("error updating terraform state %v", resp.Diagnostics.Errors()))
		return
	}
}

func (r *serviceResource) validateCreateReadReplicaRequest(ctx context.Context, primary *tsClient.Service, plan serviceResourceModel) error {
	tflog.Trace(ctx, "validateCreateReadReplicaRequest")

	if primary.ForkSpec != nil {
		return errors.New(errReplicaFromFork)
	}
	if (!plan.EnableHAReplica.IsNull() && plan.EnableHAReplica.ValueBool()) || (!plan.HAReplicas.IsNull() && plan.HAReplicas.ValueInt64() > 0) {
		return errors.New(errReplicaWithHA)
	}
	return nil
}

// createReadReplicaWithRetry attempts to create a read replica with retry logic for backup availability errors.
func (r *serviceResource) createReadReplicaWithRetry(ctx context.Context, request tsClient.CreateServiceRequest) (*tsClient.CreateServiceResponse, error) {
	tflog.Trace(ctx, "ServiceResource.createReadReplicaWithRetry")

	var response *tsClient.CreateServiceResponse

	err := retry.RetryContext(ctx, 10*time.Minute, func() *retry.RetryError {
		resp, err := r.client.CreateService(ctx, request)
		if err != nil {
			errMsg := err.Error()
			if strings.Contains(errMsg, "doesn't yet have any backups or snapshots available") {
				tflog.Info(ctx, "Parent service doesn't have backups yet, retrying...")
				// Retry. The parent service needs more time to create backups
				return retry.RetryableError(err)
			}
			return retry.NonRetryableError(err)
		}

		// Success
		response = resp
		return nil
	})

	if err != nil {
		return nil, err
	}

	return response, nil
}

func (r *serviceResource) waitForServiceReadiness(ctx context.Context, id string, timeouts timeouts.Value) (*tsClient.Service, error) {
	tflog.Trace(ctx, "ServiceResource.waitForServiceReadiness")

	defaultTimeout := 45 * time.Minute
	timeout, diags := timeouts.Create(ctx, defaultTimeout)
	if diags != nil && diags.HasError() {
		tflog.Error(ctx, fmt.Sprintf("found errs %v", diags.Errors()))
		return nil, fmt.Errorf("unable to get timeout from config %v", diags.Errors())
	}

	conf := retry.StateChangeConf{
		Pending:                   []string{"QUEUED", "CONFIGURING", "UNSTABLE", "PAUSING", "RESUMING"},
		Target:                    []string{"READY", "PAUSED"},
		Delay:                     30 * time.Second,
		Timeout:                   timeout,
		PollInterval:              15 * time.Second,
		NotFoundChecks:            40,
		ContinuousTargetOccurence: 1,
		Refresh: func() (result interface{}, state string, err error) {
			s, err := r.client.GetService(ctx, id)
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

func (r *serviceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
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

func (r *serviceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "ServiceResource.Update")
	var plan, state serviceResourceModel
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	serviceID := state.ID.ValueString()

	// Validate HA configuration
	if err := validateHAConfiguration(plan); err != nil {
		resp.Diagnostics.AddError(ErrInvalidAttribute, err.Error())
		return
	}

	readReplicaSource := plan.ReadReplicaSource.ValueString()
	if readReplicaSource != state.ReadReplicaSource.ValueString() {
		resp.Diagnostics.AddError(ErrUpdateService, errUpdateReplicaSource)
		return
	}

	if readReplicaSource != "" && ((!plan.EnableHAReplica.IsNull() && plan.EnableHAReplica.ValueBool()) || (!plan.HAReplicas.IsNull() && plan.HAReplicas.ValueInt64() > 0)) {
		resp.Diagnostics.AddError(ErrUpdateService, errReplicaWithHA)
		return
	}

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

	if plan.Paused != state.Paused {
		status := "ACTIVE"
		if plan.Paused.ValueBool() {
			status = "INACTIVE"
		}
		if _, err := r.client.ToggleService(ctx, serviceID, status); err != nil {
			resp.Diagnostics.AddError("Failed to toggle service", err.Error())
			return
		}
	}

	// Connection pooler ////////////////////////////////////////
	if plan.ConnectionPoolerEnabled != state.ConnectionPoolerEnabled {
		if err := r.client.ToggleConnectionPooler(ctx, serviceID, plan.ConnectionPoolerEnabled.ValueBool()); err != nil {
			resp.Diagnostics.AddError("Failed to toggle connection pooler", err.Error())
			return
		}
	}
	if plan.EnvironmentTag != state.EnvironmentTag {
		if err := r.client.SetEnvironmentTag(ctx, serviceID, plan.EnvironmentTag.ValueString()); err != nil {
			resp.Diagnostics.AddError("Failed to set environment tag", err.Error())
			return
		}
	}
	if !plan.PoolerHostname.IsUnknown() {
		resp.Diagnostics.AddError(ErrUpdateService, "Do not support pooler hostname change")
		return
	}
	if !plan.PoolerPort.IsUnknown() {
		resp.Diagnostics.AddError(ErrUpdateService, "Do not support pooler port change")
		return
	}

	// HA Replica ////////////////////////////////////////
	// Check if either the deprecated field or new fields have changed
	var planReplicaCount, stateReplicaCount int64
	var planSyncReplicaCount, stateSyncReplicaCount int64

	// Calculate current state replica count and sync replica count
	if !state.HAReplicas.IsNull() {
		stateReplicaCount = state.HAReplicas.ValueInt64()
		stateSyncReplicaCount = state.SyncReplicas.ValueInt64()
	} else if !state.EnableHAReplica.IsNull() && state.EnableHAReplica.ValueBool() {
		stateReplicaCount = 1
		stateSyncReplicaCount = 0
	}

	// Calculate planned replica count and sync replica count
	if !plan.HAReplicas.IsNull() {
		planReplicaCount = plan.HAReplicas.ValueInt64()
		planSyncReplicaCount = plan.SyncReplicas.ValueInt64()
	} else if !plan.EnableHAReplica.IsNull() && plan.EnableHAReplica.ValueBool() {
		planReplicaCount = 1
		planSyncReplicaCount = 0
	}

	// Update the replica count if it has changed
	if planReplicaCount != stateReplicaCount || planSyncReplicaCount != stateSyncReplicaCount {
		if err := r.client.SetReplicaCount(ctx, serviceID, int(planReplicaCount), int(planSyncReplicaCount)); err != nil {
			resp.Diagnostics.AddError("Failed to update HA replicas", err.Error())
			return
		}
	}

	// VPC ////////////////////////////////////////
	if !plan.VpcID.Equal(state.VpcID) {
		// if state.VpcId is known and different from plan.VpcId, we must detach first
		if !state.VpcID.IsNull() && !state.VpcID.IsUnknown() {
			if err := r.client.DetachServiceFromVPC(ctx, serviceID, state.VpcID.ValueInt64()); err != nil {
				resp.Diagnostics.AddError("Failed to detach service from VPC", err.Error())
				return
			}
		}
		// if plan.VpcId is known, it must be attached
		if !plan.VpcID.IsNull() && !plan.VpcID.IsUnknown() {
			if err := r.client.AttachServiceToVPC(ctx, serviceID, plan.VpcID.ValueInt64()); err != nil {
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

	// Exporters
	if !plan.MetricExporterID.Equal(state.MetricExporterID) {
		// Detach old and attach new
		if !state.MetricExporterID.IsNull() && !state.MetricExporterID.IsUnknown() {
			err := r.client.DetachMetricExporter(ctx, serviceID, state.MetricExporterID.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(errDetachExporter, err.Error())
				return
			}
		}
		if !plan.MetricExporterID.IsNull() && !plan.MetricExporterID.IsUnknown() {
			err := r.client.AttachMetricExporter(ctx, serviceID, plan.MetricExporterID.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(errAttachExporter, err.Error())
				return
			}
		}
	}

	if !plan.LogExporterID.Equal(state.LogExporterID) {
		// Detach old and attach new
		if !state.LogExporterID.IsNull() && !state.LogExporterID.IsUnknown() {
			err := r.client.DetachGenericExporter(ctx, serviceID, state.LogExporterID.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(errDetachExporter, err.Error())
				return
			}
		}
		if !plan.LogExporterID.IsNull() && !plan.LogExporterID.IsUnknown() {
			err := r.client.AttachGenericExporter(ctx, serviceID, plan.LogExporterID.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(errAttachExporter, err.Error())
				return
			}
		}
	}

	// Private Link ////////////////////////////////////////
	if !plan.PrivateEndpointConnectionID.Equal(state.PrivateEndpointConnectionID) {
		// Detach old if exists
		if !state.PrivateEndpointConnectionID.IsNull() && !state.PrivateEndpointConnectionID.IsUnknown() {
			if err := r.client.DetachServiceFromPrivateLink(ctx, serviceID, state.PrivateEndpointConnectionID.ValueString()); err != nil {
				resp.Diagnostics.AddError("Failed to detach service from private link", err.Error())
				return
			}
		}
		// Attach new if specified (with retry for async detach completion)
		if !plan.PrivateEndpointConnectionID.IsNull() && !plan.PrivateEndpointConnectionID.IsUnknown() {
			err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
				attachErr := r.client.AttachServiceToPrivateLink(ctx, serviceID, plan.PrivateEndpointConnectionID.ValueString())
				if attachErr == nil {
					return nil
				}
				// Retry if service is still detaching from previous connection
				if strings.Contains(attachErr.Error(), "already attached") {
					tflog.Info(ctx, "Service still detaching from previous private link, retrying attach...", map[string]interface{}{
						"service_id": serviceID,
					})
					return retry.RetryableError(attachErr)
				}
				return retry.NonRetryableError(attachErr)
			})
			if err != nil {
				resp.Diagnostics.AddError("Failed to attach service to private link", err.Error())
				return
			}
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
		resp.Diagnostics.AddError(ErrCreateTimeout, fmt.Sprintf("error occurred while waiting for service reconfiguration, got error: %s", err))
		return
	}

	// Update Password if it has changed and if it's not a read replica
	if !plan.Password.Equal(state.Password) && !plan.Password.IsNull() && readReplicaSource == "" {
		err := r.client.ResetServicePassword(ctx, serviceID, plan.Password.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to update password", fmt.Sprintf("Unable to update password, got error: %s", err))
			return
		}
	}

	resources := serviceToResource(resp.Diagnostics, service, plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, resources)...)

	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, fmt.Sprintf("error updating terraform state %v", resp.Diagnostics.Errors()))
		return
	}
}

func (r *serviceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
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

func (r *serviceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func serviceToResource(diag diag.Diagnostics, s *tsClient.Service, state serviceResourceModel) serviceResourceModel {
	hasPooler := s.ServiceSpec.PoolerEnabled
	replicaCount := s.Resources[0].Spec.ReplicaCount
	syncReplicaCount := s.Resources[0].Spec.SyncReplicaCount

	model := serviceResourceModel{
		ID:                      types.StringValue(s.ID),
		Password:                state.Password,
		Name:                    types.StringValue(s.Name),
		MilliCPU:                types.Int64Value(s.Resources[0].Spec.MilliCPU),
		MemoryGB:                types.Int64Value(s.Resources[0].Spec.MemoryGB),
		Username:                types.StringValue(s.ServiceSpec.Username),
		RegionCode:              types.StringValue(s.RegionCode),
		Timeouts:                state.Timeouts,
		HAReplicas:              types.Int64Value(replicaCount),
		SyncReplicas:            types.Int64Value(syncReplicaCount),
		Paused:                  types.BoolValue(s.Status == "PAUSED" || s.Status == "PAUSING"),
		ReadReplicaSource:       state.ReadReplicaSource,
		ConnectionPoolerEnabled: types.BoolValue(hasPooler),
		EnableHAReplica:         types.BoolNull(),
		Hostname:                types.StringNull(),
		Port:                    types.Int64Null(),
		ReplicaHostname:         types.StringNull(),
		ReplicaPort:             types.Int64Null(),
		PoolerHostname:          types.StringNull(),
		PoolerPort:              types.Int64Null(),
	}

	// If the user was using the deprecated has_ha_replica field, populate it from the API for backwards compatibility
	if !state.EnableHAReplica.IsNull() {
		if replicaCount > 0 {
			model.EnableHAReplica = types.BoolValue(true)
		} else {
			model.EnableHAReplica = types.BoolValue(false)
		}
	}

	if s.VPCEndpoint != nil {
		if vpcID, err := strconv.ParseInt(s.VPCEndpoint.VPCId, 10, 64); err != nil {
			diag.AddError("Parse Error", "could not parse vpcID")
		} else {
			model.VpcID = types.Int64Value(vpcID)
		}
	}
	if s.Metadata != nil {
		model.EnvironmentTag = types.StringValue(s.Metadata.Environment)
	}
	if s.ForkSpec != nil && s.ForkSpec.IsStandby {
		model.ReadReplicaSource = types.StringValue(s.ForkSpec.ServiceID)
	}

	if s.Endpoints != nil {
		if s.Endpoints.Primary != nil && s.Endpoints.Primary.Host != "" {
			model.Hostname = types.StringValue(s.Endpoints.Primary.Host)
			model.Port = types.Int64Value(int64(s.Endpoints.Primary.Port))
		}

		if replicaCount > 0 && s.Endpoints.Replica != nil && s.Endpoints.Replica.Host != "" {
			model.ReplicaHostname = types.StringValue(s.Endpoints.Replica.Host)
			model.ReplicaPort = types.Int64Value(int64(s.Endpoints.Replica.Port))
		}

		if hasPooler && s.Endpoints.Pooler != nil && s.Endpoints.Pooler.Host != "" {
			model.PoolerHostname = types.StringValue(s.Endpoints.Pooler.Host)
			model.PoolerPort = types.Int64Value(int64(s.Endpoints.Pooler.Port))
		}
	}

	if s.ServiceSpec.MetricExporterUUID != nil {
		model.MetricExporterID = types.StringValue(*s.ServiceSpec.MetricExporterUUID)
	}
	if s.ServiceSpec.GenericExporterID != nil {
		model.LogExporterID = types.StringValue(*s.ServiceSpec.GenericExporterID)
	}

	if s.PrivateLinkEndpointConnectionID != nil {
		model.PrivateEndpointConnectionID = types.StringValue(*s.PrivateLinkEndpointConnectionID)
	} else {
		model.PrivateEndpointConnectionID = types.StringNull()
	}

	return model
}
