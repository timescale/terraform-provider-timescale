package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	tsClient "github.com/timescale/terraform-provider-timescale/internal/client"
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

//type metricExporterModel struct {
//	ID types.Int64 `tfsdk:"id"`
//}

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
