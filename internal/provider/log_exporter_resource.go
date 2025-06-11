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
	"time"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &logExporterResource{}
	_ resource.ResourceWithConfigure = &logExporterResource{}
)

// NewLogExporterResource is a helper function to simplify the provider implementation.
func NewLogExporterResource() resource.Resource {
	return &logExporterResource{}
}

// logExporterResource is the data source implementation.
type logExporterResource struct {
	client *tsClient.Client
}

type logExporterResourceModel struct {
	ID      types.String `tfsdk:"id"`
	Name    types.String `tfsdk:"name"`
	Region  types.String `tfsdk:"region"`
	Created types.String `tfsdk:"created"`
	Type    types.String `tfsdk:"type"`

	Cloudwatch *cloudwatchLogConfigModel `tfsdk:"cloudwatch"`
}

type cloudwatchLogConfigModel struct {
	Region        types.String `tfsdk:"region"`
	RoleARN       types.String `tfsdk:"role_arn"`
	AccessKey     types.String `tfsdk:"access_key"`
	SecretKey     types.String `tfsdk:"secret_key"`
	LogGroupName  types.String `tfsdk:"log_group_name"`
	LogStreamName types.String `tfsdk:"log_stream_name"`
}

// Metadata returns the resource type name.
func (r *logExporterResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_log_exporter"
}

// Read refreshes the Terraform state with the latest data.
func (r *logExporterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "logExporterResource.Read")

	// Get current state
	var state logExporterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	exporters, err := r.client.GetAllGenericExporters(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error getting all Log Exporters", err.Error())
		return
	}

	var foundExporter *tsClient.GenericExporter
	for _, exporter := range exporters {
		if exporter.ID == state.ID.ValueString() {
			foundExporter = exporter
			break
		}
	}

	if foundExporter != nil {
		r.mapExporterToModel(foundExporter, &state)

		// Set the refreshed state
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	} else {
		tflog.Warn(ctx, "Log exporter not found, removing from state.", map[string]any{"id": state.ID.ValueString()})
		resp.State.RemoveResource(ctx)
	}
}

// Create creates a log exporter.
func (r *logExporterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "logExporterResource.Create")

	// Get plan
	var plan logExporterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	// Validations
	if plan.Cloudwatch == nil {
		resp.Diagnostics.AddError(
			"Missing Exporter Configuration",
			"Cloudwatch configuration block must be specified.",
		)
		return
	}

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

	if isKeyAuthAccess && len(cwConfig.AccessKey.ValueString()) < 16 || len(cwConfig.AccessKey.ValueString()) > 128 {
		resp.Diagnostics.AddAttributeError(path.Root("cloudwatch"), "Invalid AccessKey", "Length must be between 16 and 128 characters.")
	}
	if isKeyAuthSecret && len(cwConfig.SecretKey.ValueString()) < 40 || len(cwConfig.SecretKey.ValueString()) > 128 {
		resp.Diagnostics.AddAttributeError(path.Root("cloudwatch"), "Invalid SecretKey", "Length must be between 16 and 128 characters.")
	}

	// If there have been any validation errors, we don't continue.
	if resp.Diagnostics.HasError() {
		return
	}

	// Everything is good. Proceed with resource creation
	// Populate the config based on the plan
	config := tsClient.GenericExporterConfig{}
	config.Cloudwatch = &tsClient.CloudwatchGenericConfig{
		Region:        plan.Cloudwatch.Region.ValueString(),
		RoleARN:       plan.Cloudwatch.RoleARN.ValueString(),
		AccessKey:     plan.Cloudwatch.AccessKey.ValueString(),
		SecretKey:     plan.Cloudwatch.SecretKey.ValueString(),
		LogGroupName:  plan.Cloudwatch.LogGroupName.ValueString(),
		LogStreamName: plan.Cloudwatch.LogStreamName.ValueString(),
	}

	exporter, err := r.client.CreateGenericExporter(
		ctx,
		plan.Name.ValueString(),
		plan.Region.ValueString(),
		"CLOUDWATCH",
		"LOG",
		config,
	)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Create Log Exporter", err.Error())
		return
	}

	r.mapExporterToModel(exporter, &plan)

	// Set the final state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes a log exporter.
func (r *logExporterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "logExporterResource.Delete")

	// Get current state
	var state logExporterResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteGenericExporter(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting Log Exporter", err.Error())
	}
}

func (r *logExporterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "logExporterResource.Update")

	// Get plan
	var plan logExporterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get state
	var state logExporterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := state.ID.ValueString()

	// Populate the config based on the plan
	config := tsClient.GenericExporterConfig{}
	config.Cloudwatch = &tsClient.CloudwatchGenericConfig{
		Region:        plan.Cloudwatch.Region.ValueString(),
		RoleARN:       plan.Cloudwatch.RoleARN.ValueString(),
		AccessKey:     plan.Cloudwatch.AccessKey.ValueString(),
		SecretKey:     plan.Cloudwatch.SecretKey.ValueString(),
		LogGroupName:  plan.Cloudwatch.LogGroupName.ValueString(),
		LogStreamName: plan.Cloudwatch.LogStreamName.ValueString(),
	}

	err := r.client.UpdateGenericExporter(
		ctx,
		id,
		plan.Name.ValueString(),
		config,
	)

	if err != nil {
		resp.Diagnostics.AddError("Error updating Log Exporter", err.Error())
		return
	}

	// Update state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *logExporterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Configure adds the provider configured client to the resource.
func (r *logExporterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	tflog.Trace(ctx, "logExporterResource.Configure")
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
func (r *logExporterResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Schema for a log exporter.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Log exporter ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Log exporter name.",
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
				MarkdownDescription: "Timestamp of when the log exporter was created (RFC3339 format).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Type of the log exporter. Possible values: cloudwatch.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			// Note: this is a separate block to support other providers in the future.
			"cloudwatch": schema.SingleNestedAttribute{
				MarkdownDescription: "Configuration for AWS CloudWatch exporter. Configure authentication using either `role_arn` or `access_key` with `secret_key`.",
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

// mapExporterToModel maps the unified API model to the terraform resource model.
func (r *logExporterResource) mapExporterToModel(exporter *tsClient.GenericExporter, model *logExporterResourceModel) {
	model.ID = types.StringValue(exporter.ID)
	model.Name = types.StringValue(exporter.Name)
	model.Created = types.StringValue(exporter.Created)
	model.Type = types.StringValue(strings.ToLower(exporter.Type))

	if exporter.RegionCode != "" {
		model.Region = types.StringValue(exporter.RegionCode)
	}

	createdTime, err := time.Parse(time.RFC3339Nano, exporter.Created)
	if err == nil {
		// Format to a consistent, lower precision that matches the Read API response.
		model.Created = types.StringValue(createdTime.Format("2006-01-02T15:04:05.999999Z07:00"))
	} else {
		model.Created = types.StringValue(exporter.Created)
	}

	switch strings.ToUpper(exporter.Type) {
	case "CLOUDWATCH":
		if exporter.Cloudwatch != nil {
			if model.Cloudwatch == nil {
				model.Cloudwatch = &cloudwatchLogConfigModel{}
			}
			model.Cloudwatch.Region = types.StringValue(exporter.Cloudwatch.Region)
			model.Cloudwatch.LogGroupName = types.StringValue(exporter.Cloudwatch.LogGroupName)
			model.Cloudwatch.LogStreamName = types.StringValue(exporter.Cloudwatch.LogStreamName)

			// Sensitive values are not always returned from APIs
			if exporter.Cloudwatch.RoleARN != "" {
				model.Cloudwatch.RoleARN = types.StringValue(exporter.Cloudwatch.RoleARN)
			}
			if exporter.Cloudwatch.AccessKey != "" {
				model.Cloudwatch.AccessKey = types.StringValue(exporter.Cloudwatch.AccessKey)
			}
			if exporter.Cloudwatch.SecretKey != "" {
				model.Cloudwatch.SecretKey = types.StringValue(exporter.Cloudwatch.SecretKey)
			}
		}
	}
}
