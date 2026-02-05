package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	tsClient "github.com/timescale/terraform-provider-timescale/internal/client"
)

var _ datasource.DataSource = &privateLinkAvailableRegionsDataSource{}

func NewPrivateLinkAvailableRegionsDataSource() datasource.DataSource {
	return &privateLinkAvailableRegionsDataSource{}
}

type privateLinkAvailableRegionsDataSource struct {
	client *tsClient.Client
}

type privateLinkAvailableRegionModel struct {
	Region                  types.String `tfsdk:"region"`
	PrivateLinkServiceAlias types.String `tfsdk:"private_link_service_alias"`
}

type privateLinkAvailableRegionsDataSourceModel struct {
	Regions []privateLinkAvailableRegionModel `tfsdk:"regions"`
}

func (d *privateLinkAvailableRegionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_privatelink_available_regions"
}

func (d *privateLinkAvailableRegionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists available regions for Azure Private Link.",
		MarkdownDescription: `Lists available regions for Azure Private Link.

This data source returns all regions where Azure Private Link is available,
along with the Private Link Service alias for each region. Use the alias
when creating Azure Private Endpoints.`,
		Attributes: map[string]schema.Attribute{
			"regions": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of available regions for Private Link.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"region": schema.StringAttribute{
							Computed:    true,
							Description: "The Timescale region code (e.g., az-eastus2).",
						},
						"private_link_service_alias": schema.StringAttribute{
							Computed:    true,
							Description: "The Azure Private Link Service alias to use when creating a Private Endpoint.",
						},
					},
				},
			},
		},
	}
}

func (d *privateLinkAvailableRegionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*tsClient.Client)
}

func (d *privateLinkAvailableRegionsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Trace(ctx, "PrivateLinkAvailableRegionsDataSource.Read")

	regions, err := d.client.ListPrivateLinkAvailableRegions(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Private Link available regions", err.Error())
		return
	}

	var state privateLinkAvailableRegionsDataSourceModel
	for _, r := range regions {
		state.Regions = append(state.Regions, privateLinkAvailableRegionModel{
			Region:                  types.StringValue(r.Region),
			PrivateLinkServiceAlias: types.StringValue(r.PrivateLinkServiceAlias),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
