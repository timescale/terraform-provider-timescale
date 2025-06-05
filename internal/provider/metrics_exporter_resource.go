package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	tsClient "github.com/timescale/terraform-provider-timescale/internal/client"
	"strings"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &metricExporterResource{}
	_ resource.ResourceWithConfigure = &metricExporterResource{}
)

// NewMetricsExporterResource is a helper function to simplify the provider implementation.
func NewMetricsExporterResource() resource.Resource {
	return &metricExporterResource{}
}

// metricExporterResource is the data source implementation.
type metricExporterResource struct {
	client *tsClient.Client
}

type metricExporterResourceModel struct {
	ID      types.String `tfsdk:"id"`
	UUID    types.String `tfsdk:"uuid"`
	Name    types.String `tfsdk:"name"`
	Region  types.String `tfsdk:"region"`
	Created types.String `tfsdk:"created"`
	Type    types.String `tfsdk:"type"`

	Datadog    *datadogConfigModel    `tfsdk:"datadog"`
	Prometheus *prometheusConfigModel `tfsdk:"prometheus"`
	Cloudwatch *cloudwatchConfigModel `tfsdk:"cloudwatch"`
}

type datadogConfigModel struct {
	APIKey types.String `tfsdk:"api_key"`
	Site   types.String `tfsdk:"site"`
}

type prometheusConfigModel struct {
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

type cloudwatchConfigModel struct {
	Region        types.String `tfsdk:"region"`
	RoleARN       types.String `tfsdk:"role_arn"`
	AccessKey     types.String `tfsdk:"access_key"`
	SecretKey     types.String `tfsdk:"secret_key"`
	LogGroupName  types.String `tfsdk:"log_group_name"`
	LogStreamName types.String `tfsdk:"log_stream_name"`
	Namespace     types.String `tfsdk:"namespace"`
}

// Metadata returns the resource type name.
func (r *metricExporterResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_metrics_exporter"
}

// Read refreshes the Terraform state with the latest data.
func (r *metricExporterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "metricExporterResource.Read")
}

// Create creates a metrics exporter.
func (r *metricExporterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "metricExporterResource.Create")

	// Read Terraform plan data into the model
	var plan metricExporterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	// Validations
	// First check that only one of the three types of exporters is configured (datadog OR prometheus OR cloudwatch)
	authBlocksSet := 0
	if plan.Datadog != nil {
		authBlocksSet++
	}
	if plan.Prometheus != nil {
		authBlocksSet++
	}
	if plan.Cloudwatch != nil {
		authBlocksSet++
	}

	if authBlocksSet == 0 {
		resp.Diagnostics.AddError(
			"Missing Exporter Configuration",
			"One of datadog, prometheus, or cloudwatch configuration blocks must be specified.",
		)
	} else if authBlocksSet > 1 {
		resp.Diagnostics.AddError(
			"Conflicting Exporter Configuration",
			"Only one of datadog, prometheus, or cloudwatch configuration blocks can be specified.",
		)
	}

	// If the `cloudwatch` block is present, ensure that:
	// 1. Either `role_arn` is set OR both `access_key` AND `secret_key` are set.
	// 2. Not both `role_arn` and keys are set.
	// 3. If one key is set, the other must also be set.
	if plan.Cloudwatch != nil {
		cwConfig := plan.Cloudwatch
		isRoleAuth := !cwConfig.RoleARN.IsNull() && cwConfig.RoleARN.ValueString() != ""
		isKeyAuthAccess := !cwConfig.AccessKey.IsNull() && cwConfig.AccessKey.ValueString() != ""
		isKeyAuthSecret := !cwConfig.SecretKey.IsNull() && cwConfig.SecretKey.ValueString() != ""

		if isRoleAuth && (isKeyAuthAccess || isKeyAuthSecret) {
			resp.Diagnostics.AddAttributeError(path.Root("cloudwatch").AtName("role_arn"), "Conflicting Authentication", "Cannot use `role_arn` with `access_key` or `secret_key`.")
		} else if (isKeyAuthAccess && !isKeyAuthSecret) || (!isKeyAuthAccess && isKeyAuthSecret) {
			resp.Diagnostics.AddAttributeError(path.Root("cloudwatch"), "Incomplete Key Authentication", "Both `access_key` and `secret_key` must be provided.")
		} else if !isRoleAuth && !isKeyAuthAccess {
			resp.Diagnostics.AddAttributeError(path.Root("cloudwatch"), "Missing Authentication Method", "Either `role_arn` or both `access_key` and `secret_key` must be provided.")
		}
	}
	// If there have been any validation errors, we don't continue.
	if resp.Diagnostics.HasError() {
		return
	}

	// Everything is good. Proceed with resource creation
	if plan.Datadog != nil {
		exporter, err := r.client.CreateDatadogMetricsExporter(ctx, plan.Name.ValueString(), plan.Region.ValueString(), plan.Datadog.APIKey.ValueString(), plan.Datadog.Site.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Unable to Create Datadog Exporter: %v", plan),
				err.Error(),
			)
			return
		}

		// Map client model to tf schema
		plan.ID = types.StringValue(exporter.ID)
		plan.UUID = types.StringValue(exporter.ID)
		plan.Name = types.StringValue(exporter.Name)
		plan.Created = types.StringValue(exporter.Created)
		plan.Type = types.StringValue(strings.ToLower(exporter.Type))
		plan.Datadog.APIKey = types.StringValue(exporter.Config.APIKey)
		plan.Datadog.Site = types.StringValue(exporter.Config.Site)

		// Set state
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
}

// Delete deletes a metrics exporter.
func (r *metricExporterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "metricExporterResource.Delete")
}

// Update updates a metrics exporter.
func (r *metricExporterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "metricExporterResource.Update")
}

func (r *metricExporterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

// Configure adds the provider configured client to the resource.
func (r *metricExporterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	tflog.Trace(ctx, "metricExporterResource.Configure")
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

// Schema defines the schema for the resource.
func (r *metricExporterResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Schema for a metrics exporter.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Metrics exporter internal ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"uuid": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Metrics exporter UUID to be used for service attachment.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Metrics exporter name.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "Region where the exporter will be deployed. Only services running in the same region can be attached.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created": schema.StringAttribute{
				MarkdownDescription: "Timestamp of when the metric exporter was created (RFC3339 format).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Type of the metric exporter. Possible values: datadog, prometheus, cloudwatch.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"datadog": schema.SingleNestedAttribute{
				MarkdownDescription: "Configuration for Datadog exporter. Cannot be used with `prometheus` or `cloudwatch`.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"api_key": schema.StringAttribute{
						MarkdownDescription: "Datadog API key.",
						Required:            true,
						Sensitive:           true,
					},
					"site": schema.StringAttribute{
						MarkdownDescription: "Datadog site (e.g., 'datadoghq.com', 'datadoghq.eu').",
						Required:            true,
					},
				},
			},
			"prometheus": schema.SingleNestedAttribute{
				MarkdownDescription: "Configuration for Prometheus exporter. Cannot be used with `datadog` or `cloudwatch`.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"username": schema.StringAttribute{
						MarkdownDescription: "Username for Prometheus basic authentication.",
						Required:            true,
					},
					"password": schema.StringAttribute{
						MarkdownDescription: "Password for Prometheus basic authentication.",
						Required:            true,
						Sensitive:           true,
					},
				},
			},
			"cloudwatch": schema.SingleNestedAttribute{
				MarkdownDescription: "Configuration for AWS CloudWatch exporter. Configure authentication using either `role_arn` or `access_key` with `secret_key`. Cannot be used with `datadog` or `prometheus`.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"log_group_name": schema.StringAttribute{
						MarkdownDescription: "Name of the CloudWatch Log Group.",
						Required:            true,
					},
					"log_stream_name": schema.StringAttribute{
						MarkdownDescription: "Name of the CloudWatch Log Stream.",
						Required:            true,
					},
					"namespace": schema.StringAttribute{
						MarkdownDescription: "CloudWatch Metrics Namespace.",
						Required:            true,
					},
					"region": schema.StringAttribute{
						MarkdownDescription: "AWS region for CloudWatch.",
						Required:            true,
					},
					// Role authentication method
					"role_arn": schema.StringAttribute{
						MarkdownDescription: "ARN of the IAM role to assume for CloudWatch access. If provided, `access_key` and `secret_key` must not be set.",
						Optional:            true,
					},
					// CloudWatch credentials authentication method
					"access_key": schema.StringAttribute{
						MarkdownDescription: "AWS access key ID. If provided, `secret_key` must also be set, and `role_arn` must not be set.",
						Optional:            true,
					},
					"secret_key": schema.StringAttribute{
						MarkdownDescription: "AWS secret access key. If provided, `access_key` must also be set, and `role_arn` must not be set.",
						Optional:            true,
						Sensitive:           true,
					},
				},
			}},
	}
}
