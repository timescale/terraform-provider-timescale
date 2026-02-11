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

var _ datasource.DataSource = &privateLinkAvailableRegionsDataSource{}

func NewPrivateLinkAvailableRegionsDataSource() datasource.DataSource {
	return &privateLinkAvailableRegionsDataSource{}
}

type privateLinkAvailableRegionsDataSource struct {
	client *tsClient.Client
}

type privateLinkAvailableRegionModel struct {
	CloudProvider types.String `tfsdk:"cloud_provider"`
	ServiceName   types.String `tfsdk:"service_name"`
}

type privateLinkAvailableRegionsDataSourceModel struct {
	Regions map[string]privateLinkAvailableRegionModel `tfsdk:"regions"`
}

func (d *privateLinkAvailableRegionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_privatelink_available_regions"
}

func (d *privateLinkAvailableRegionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists available regions for Private Link.",
		MarkdownDescription: `Lists available regions for Private Link.

This data source returns all regions where Private Link is available,
along with the service name and cloud provider for each region.

## Example Usage

` + "```hcl" + `
data "timescale_privatelink_available_regions" "all" {}

# Access the service name for an Azure region
locals {
  azure_service = data.timescale_privatelink_available_regions.all.regions["az-eastus"].service_name
}

# Access the service name for an AWS region
locals {
  aws_service = data.timescale_privatelink_available_regions.all.regions["us-east-1"].service_name
}
` + "```",
		Attributes: map[string]schema.Attribute{
			"regions": schema.MapNestedAttribute{
				Computed:    true,
				Description: "Map of available regions for Private Link, keyed by region code.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"cloud_provider": schema.StringAttribute{
							Computed:    true,
							Description: "The cloud provider for this region (AZURE or AWS).",
						},
						"service_name": schema.StringAttribute{
							Computed:    true,
							Description: "The service name to use when creating a Private Endpoint (Azure alias or AWS VPC Endpoint Service name).",
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

func (d *privateLinkAvailableRegionsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Trace(ctx, "PrivateLinkAvailableRegionsDataSource.Read")

	regions, err := d.client.ListPrivateLinkAvailableRegions(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Private Link available regions", err.Error())
		return
	}

	state := privateLinkAvailableRegionsDataSourceModel{
		Regions: make(map[string]privateLinkAvailableRegionModel),
	}
	for _, r := range regions {
		state.Regions[r.Region] = privateLinkAvailableRegionModel{
			CloudProvider: types.StringValue(r.CloudProvider),
			ServiceName:   types.StringValue(r.ServiceName),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
