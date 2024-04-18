package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	tsClient "github.com/timescale/terraform-provider-timescale/internal/client"
)

var _ datasource.DataSource

func NewExporterDataSource() datasource.DataSource {
	return &ExporterDataSource{}
}

type ExporterDataSource struct {
	client *tsClient.Client
}

type ExporterDataSourceModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func (e *ExporterDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_exporter"
}

func (e *ExporterDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	tflog.Trace(ctx, "ExporterDataSource.Configure")

	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*tsClient.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Client Type",
			fmt.Sprintf("Expected *tsClient.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	e.client = client
}

func (e *ExporterDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Exporter data source",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "exporter id is the unique identifier for an exporter",
				Description:         "exporter id",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of this exporter. Exporter names must be unique in order to manage them using Terraform.",
				Description:         "The name of this exporter. Exporter names must be unique in order to manage them using Terraform.",
				Required:            true,
			},
		},
	}
}

func (e *ExporterDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Trace(ctx, "ExporterDataSource.Read")
	var name string
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("name"), &name)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, fmt.Sprintf("error reading terraform plan %v", resp.Diagnostics.Errors()))
		return
	}
	tflog.Info(ctx, "getting exporter: "+name)
	exporter, err := e.client.GetExporterByName(ctx, &tsClient.GetExporterByNameRequest{
		Name: name,
	})
	if err != nil {
		resp.Diagnostics.AddError("client error, unable to get exporter", err.Error())
		return
	}
	state := exporterToDataModel(exporter)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "error updating terraform state")
		return
	}
}

func exporterToDataModel(e *tsClient.Exporter) ExporterDataSourceModel {
	exporterModel := ExporterDataSourceModel{
		ID:   types.StringValue(e.ID),
		Name: types.StringValue(e.Name),
	}
	return exporterModel
}
