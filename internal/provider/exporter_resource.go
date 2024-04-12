package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	tsClient "github.com/timescale/terraform-provider-timescale/internal/client"
)

var _ resource.ResourceWithImportState = &ExporterResource{}

func NewExporterResource() resource.Resource {
	return &ExporterResource{}
}

type ExporterResource struct {
	client *tsClient.Client
}

type exporterResourceModel struct {
	ID         types.String         `tfsdk:"id"`
	Provider   types.String         `tfsdk:"export_to"`
	Type       types.String         `tfsdk:"type"`
	Name       types.String         `tfsdk:"name"`
	RegionCode types.String         `tfsdk:"region_code"`
	Config     jsontypes.Normalized `tfsdk:"config"`
}

func (e *ExporterResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	tflog.Trace(ctx, "ExporterResource.Metadata")
	resp.TypeName = req.ProviderTypeName + "_exporter"
}

// Configure adds the provider configured client to the service resource.
func (e *ExporterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	tflog.Trace(ctx, "ExporterResource.Configure")
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
	e.client = client
}

func (e *ExporterResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	tflog.Trace(ctx, "ExporterResource.Schema")
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Computed: true,
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"export_to": schema.StringAttribute{
				Required: true,
			},
			"type": schema.StringAttribute{
				Required: true,
			},
			"region_code": schema.StringAttribute{
				Required: true,
			},
			"config": schema.StringAttribute{
				CustomType: jsontypes.NormalizedType{},
				Required:   true,
			},
		},
	}
}

func (e *ExporterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "ExporterResource.Create")
	var plan exporterResourceModel
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "config", map[string]interface{}{"config": plan.Config.ValueString()})

	request := &tsClient.CreateExporterRequest{
		Provider:   plan.Provider.ValueString(),
		Type:       plan.Type.ValueString(),
		Name:       plan.Name.ValueString(),
		RegionCode: plan.RegionCode.ValueString(),
		Config:     json.RawMessage(plan.Config.ValueString()),
	}
	exporter, err := e.client.CreateExporter(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create exporter, got error: %s", err))
		return
	}

	exporterModel := exporterToResource(exporter, plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, exporterModel)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, fmt.Sprintf("error updating terraform state %v", resp.Diagnostics.Errors()))
		return
	}
}

func (e *ExporterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "ExporterResource.Read")
	var state exporterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Getting Exporter: "+state.ID.ValueString())

	exporter, err := e.client.GetExporterByID(ctx, &tsClient.GetExporterByIDRequest{
		ID:       state.ID.ValueString(),
		Provider: state.Provider.ValueString(),
		Type:     state.Type.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", "unable to get exporter, got error "+err.Error())
		return
	}
	exporterModel := exporterToResource(exporter, state)
	resp.Diagnostics.Append(resp.State.Set(ctx, exporterModel)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, fmt.Sprintf("error updating terraform state %v", resp.Diagnostics.Errors()))
		return
	}
}

func (e *ExporterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

}

func (e *ExporterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

}

func (e *ExporterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ",")
	if len(parts) != 3 {
		resp.Diagnostics.AddError("unexpected import format", "expected format is `exporter_name,exporter_provider,exporter_type")
		return
	}
	name, provider, dataType := parts[0], parts[1], parts[2]
	exporter, err := e.client.GetExporterByName(ctx, &tsClient.GetExporterByNameRequest{
		Name:     name,
		Provider: provider,
		Type:     dataType,
	})
	if err != nil {
		resp.Diagnostics.AddError("unable to import exporter", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), exporter.ID)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("export_to"), provider)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("type"), dataType)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func exporterToResource(e *tsClient.Exporter, state exporterResourceModel) exporterResourceModel {
	model := exporterResourceModel{
		ID:         types.StringValue(e.ID),
		Provider:   state.Provider,
		Type:       state.Type,
		Name:       types.StringValue(e.Name),
		RegionCode: types.StringValue(e.RegionCode),
		Config:     jsontypes.NewNormalizedValue(string(e.Config)),
	}
	return model
}
