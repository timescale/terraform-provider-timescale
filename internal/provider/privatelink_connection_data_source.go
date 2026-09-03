package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	tsClient "github.com/timescale/terraform-provider-timescale/internal/client"
)

var _ datasource.DataSource = &privateLinkConnectionDataSource{}

func NewPrivateLinkConnectionDataSource() datasource.DataSource {
	return &privateLinkConnectionDataSource{}
}

type privateLinkConnectionDataSource struct {
	client *tsClient.Client
}

type privateLinkConnectionDataSourceModel struct {
	ConnectionID         types.String `tfsdk:"connection_id"`
	CloudProvider        types.String `tfsdk:"cloud_provider"`
	Region               types.String `tfsdk:"region"`
	State                types.String `tfsdk:"state"`
	Name                 types.String `tfsdk:"name"`
	LinkIdentifier       types.String `tfsdk:"link_identifier"`
	ProviderConnectionID types.String `tfsdk:"provider_connection_id"`
	PrincipalID          types.String `tfsdk:"principal_id"`
}

func (d *privateLinkConnectionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_privatelink_connection"
}

func (d *privateLinkConnectionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up a Private Link connection this project has claimed.",
		MarkdownDescription: `Looks up a Private Link connection this project has claimed.

Use this to reference a connection claimed outside of Terraform or in another
workspace. Only claimed connections are visible: an unclaimed connection belongs
to no project and cannot be looked up — claim it with
` + "`timescale_privatelink_connection`" + ` instead.

## Example Usage

` + "```hcl" + `
data "timescale_privatelink_connection" "existing" {
  connection_id = "549b2b12-c82c-4005-a7f7-175f607f086c"
}

resource "timescale_service" "example" {
  name                            = "example"
  private_endpoint_connection_ids = [data.timescale_privatelink_connection.existing.connection_id]
}
` + "```",
		Attributes: map[string]schema.Attribute{
			"connection_id": schema.StringAttribute{
				Required:    true,
				Description: "The unique identifier of the connection to look up.",
			},
			"cloud_provider": schema.StringAttribute{
				Computed:    true,
				Description: "The cloud provider the connection originates from.",
			},
			"region": schema.StringAttribute{
				Computed:    true,
				Description: "The Tiger Cloud region of the endpoint service this connection reaches.",
			},
			"state": schema.StringAttribute{
				Computed:    true,
				Description: "The state of the connection (approved, pending, rejected, removed).",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "The display name for the connection.",
			},
			"link_identifier": schema.StringAttribute{
				Computed:    true,
				Description: "The cloud provider's own identifier for the connection.",
			},
			"provider_connection_id": schema.StringAttribute{
				Computed: true,
				Description: "Provider-specific connection identifier: the VPC endpoint ID on AWS, " +
					"\"<endpoint name>.<resourceGuid>\" on Azure.",
			},
			"principal_id": schema.StringAttribute{
				Computed:    true,
				Description: "The cloud account the connection originates from.",
			},
		},
	}
}

func (d *privateLinkConnectionDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*tsClient.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *tsClient.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	d.client = client
}

func (d *privateLinkConnectionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config privateLinkConnectionDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	connectionID := config.ConnectionID.ValueString()
	tflog.Debug(ctx, "Looking up Private Link connection", map[string]interface{}{
		"connection_id": connectionID,
	})

	connections, err := d.client.ListPrivateLinkConnections(ctx, "")
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Private Link connections", err.Error())
		return
	}

	for _, conn := range connections {
		if conn.ConnectionID != connectionID {
			continue
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, &privateLinkConnectionDataSourceModel{
			ConnectionID:         types.StringValue(conn.ConnectionID),
			CloudProvider:        types.StringValue(conn.CloudProvider),
			Region:               types.StringValue(conn.Region),
			State:                types.StringValue(conn.State),
			Name:                 types.StringValue(conn.Name),
			LinkIdentifier:       types.StringValue(conn.LinkIdentifier),
			ProviderConnectionID: types.StringValue(conn.ProviderConnectionID),
			PrincipalID:          types.StringValue(conn.PrincipalID),
		})...)
		return
	}

	resp.Diagnostics.AddError(
		"Connection not found",
		fmt.Sprintf("No Private Link connection with ID %q is claimed by this project. "+
			"An unclaimed connection is not visible until it is claimed.", connectionID),
	)
}
