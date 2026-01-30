package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	tsClient "github.com/timescale/terraform-provider-timescale/internal/client"
)

var (
	_ datasource.DataSource              = &privateLinkConnectionsDataSource{}
	_ datasource.DataSourceWithConfigure = &privateLinkConnectionsDataSource{}
)

func NewPrivateLinkConnectionsDataSource() datasource.DataSource {
	return &privateLinkConnectionsDataSource{}
}

type privateLinkConnectionsDataSource struct {
	client *tsClient.Client
}

type privateLinkConnectionsDataSourceModel struct {
	ID          types.String                       `tfsdk:"id"`
	Region      types.String                       `tfsdk:"region"`
	Connections []privateLinkConnectionDSModel     `tfsdk:"connections"`
}

type privateLinkConnectionDSModel struct {
	ConnectionID   types.String                  `tfsdk:"connection_id"`
	SubscriptionID types.String                  `tfsdk:"subscription_id"`
	LinkIdentifier types.String                  `tfsdk:"link_identifier"`
	State          types.String                  `tfsdk:"state"`
	IPAddress      types.String                  `tfsdk:"ip_address"`
	Name           types.String                  `tfsdk:"name"`
	Region         types.String                  `tfsdk:"region"`
	CreatedAt      types.String                  `tfsdk:"created_at"`
	UpdatedAt      types.String                  `tfsdk:"updated_at"`
	Bindings       []privateLinkBindingDSModel   `tfsdk:"bindings"`
}

type privateLinkBindingDSModel struct {
	ProjectID                   types.String `tfsdk:"project_id"`
	ServiceID                   types.String `tfsdk:"service_id"`
	PrivateEndpointConnectionID types.String `tfsdk:"private_endpoint_connection_id"`
	BindingType                 types.String `tfsdk:"binding_type"`
	Port                        types.Int64  `tfsdk:"port"`
	Hostname                    types.String `tfsdk:"hostname"`
	CreatedAt                   types.String `tfsdk:"created_at"`
}

func (d *privateLinkConnectionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_privatelink_connections"
}

func (d *privateLinkConnectionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Data source for listing Azure Private Link connections in a Timescale project.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Placeholder identifier for Terraform.",
			},
			"region": schema.StringAttribute{
				Optional:    true,
				Description: "Filter connections by region (e.g., az-eastus2).",
			},
			"connections": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of private link connections.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"connection_id": schema.StringAttribute{
							Computed:    true,
							Description: "The unique identifier for this private endpoint connection. Use this value for the private_endpoint_connection_id attribute on timescale_service.",
						},
						"subscription_id": schema.StringAttribute{
							Computed:    true,
							Description: "Azure subscription ID.",
						},
						"link_identifier": schema.StringAttribute{
							Computed:    true,
							Description: "Azure link identifier.",
						},
						"state": schema.StringAttribute{
							Computed:    true,
							Description: "Connection state (e.g., Approved, Pending).",
						},
						"ip_address": schema.StringAttribute{
							Computed:    true,
							Description: "IP address of the private endpoint.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Name of the private endpoint connection.",
						},
						"region": schema.StringAttribute{
							Computed:    true,
							Description: "Azure region of the connection.",
						},
						"created_at": schema.StringAttribute{
							Computed:    true,
							Description: "Timestamp when the connection was created.",
						},
						"updated_at": schema.StringAttribute{
							Computed:    true,
							Description: "Timestamp when the connection was last updated.",
						},
						"bindings": schema.ListAttribute{
							Computed:    true,
							Description: "List of service bindings for this connection.",
							ElementType: types.ObjectType{
								AttrTypes: map[string]attr.Type{
									"project_id":                     types.StringType,
									"service_id":                     types.StringType,
									"private_endpoint_connection_id": types.StringType,
									"binding_type":                   types.StringType,
									"port":                           types.Int64Type,
									"hostname":                       types.StringType,
									"created_at":                     types.StringType,
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *privateLinkConnectionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client, _ = req.ProviderData.(*tsClient.Client)
}

func (d *privateLinkConnectionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config privateLinkConnectionsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	region := ""
	if !config.Region.IsNull() {
		region = config.Region.ValueString()
	}

	connections, err := d.client.ListPrivateLinkConnections(ctx, region)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Private Link connections", err.Error())
		return
	}

	var state privateLinkConnectionsDataSourceModel
	state.Region = config.Region

	for _, conn := range connections {
		connModel := privateLinkConnectionDSModel{
			ConnectionID:   types.StringValue(conn.ConnectionID),
			SubscriptionID: types.StringValue(conn.SubscriptionID),
			LinkIdentifier: types.StringValue(conn.LinkIdentifier),
			State:          types.StringValue(conn.State),
			IPAddress:      types.StringValue(conn.IPAddress),
			Name:           types.StringValue(conn.Name),
			Region:         types.StringValue(conn.Region),
			CreatedAt:      types.StringValue(conn.CreatedAt),
			UpdatedAt:      types.StringValue(conn.UpdatedAt),
		}

		var bindings []privateLinkBindingDSModel
		for _, binding := range conn.Bindings {
			bindings = append(bindings, privateLinkBindingDSModel{
				ProjectID:                   types.StringValue(binding.ProjectID),
				ServiceID:                   types.StringValue(binding.ServiceID),
				PrivateEndpointConnectionID: types.StringValue(binding.PrivateEndpointConnectionID),
				BindingType:                 types.StringValue(string(binding.BindingType)),
				Port:                        types.Int64Value(int64(binding.Port)),
				Hostname:                    types.StringValue(binding.Hostname),
				CreatedAt:                   types.StringValue(binding.CreatedAt),
			})
		}
		connModel.Bindings = bindings

		state.Connections = append(state.Connections, connModel)
	}

	state.ID = types.StringValue(fmt.Sprintf("privatelink_connections_%d", len(connections)))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
