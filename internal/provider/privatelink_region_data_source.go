package provider

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	tsClient "github.com/timescale/terraform-provider-timescale/internal/client"
)

var _ datasource.DataSource = &privateLinkRegionDataSource{}

func NewPrivateLinkRegionDataSource() datasource.DataSource {
	return &privateLinkRegionDataSource{}
}

type privateLinkRegionDataSource struct {
	client *tsClient.Client
}

type privateLinkRegionDataSourceModel struct {
	Region        types.String `tfsdk:"region"`
	CloudProvider types.String `tfsdk:"cloud_provider"`
	ServiceName   types.String `tfsdk:"service_name"`
}

func (d *privateLinkRegionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_privatelink_region"
}

func (d *privateLinkRegionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up Private Link availability for a single Timescale region.",
		MarkdownDescription: `Looks up Private Link availability for a single Timescale region.

Returns the cloud provider and service name (AWS VPC Endpoint Service name or
Azure alias) for the given region. If the region is not available for Private
Link, the data source fails with an error listing the available regions.

## Example Usage

` + "```hcl" + `
data "timescale_privatelink_region" "selected" {
  region = "us-east-1"
}

resource "aws_vpc_endpoint" "timescale" {
  service_name      = data.timescale_privatelink_region.selected.service_name
  vpc_endpoint_type = "Interface"
  # ...
}
` + "```",
		Attributes: map[string]schema.Attribute{
			"region": schema.StringAttribute{
				Required:    true,
				Description: "The Timescale region code (e.g. us-east-1, az-eastus).",
			},
			"cloud_provider": schema.StringAttribute{
				Computed:    true,
				Description: "The cloud provider for this region (azure or aws).",
			},
			"service_name": schema.StringAttribute{
				Computed:    true,
				Description: "The service name to use when creating a Private Endpoint (Azure alias or AWS VPC Endpoint Service name).",
			},
		},
	}
}

func (d *privateLinkRegionDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *privateLinkRegionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Trace(ctx, "PrivateLinkRegionDataSource.Read")

	var cfg privateLinkRegionDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	regions, err := d.client.ListPrivateLinkAvailableRegions(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Private Link available regions", err.Error())
		return
	}

	wanted := cfg.Region.ValueString()
	for _, r := range regions {
		if r.Region == wanted {
			resp.Diagnostics.Append(resp.State.Set(ctx, &privateLinkRegionDataSourceModel{
				Region:        types.StringValue(r.Region),
				CloudProvider: types.StringValue(r.CloudProvider),
				ServiceName:   types.StringValue(r.ServiceName),
			})...)
			return
		}
	}

	resp.Diagnostics.AddAttributeError(
		path.Root("region"),
		"Region not available for Private Link",
		formatUnavailableRegionMessage(wanted, regions),
	)
}

func formatUnavailableRegionMessage(requested string, regions []*tsClient.PrivateLinkAvailableRegion) string {
	byProvider := make(map[string][]string)
	for _, r := range regions {
		byProvider[r.CloudProvider] = append(byProvider[r.CloudProvider], r.Region)
	}

	providers := make([]string, 0, len(byProvider))
	for p := range byProvider {
		providers = append(providers, p)
	}
	sort.Strings(providers)

	var b strings.Builder
	fmt.Fprintf(&b, "Region %q is not available for Private Link.", requested)
	if len(regions) == 0 {
		b.WriteString(" No regions are currently available.")
		return b.String()
	}
	b.WriteString(" Available regions:")
	for _, p := range providers {
		codes := byProvider[p]
		sort.Strings(codes)
		fmt.Fprintf(&b, "\n  - %s: %s", p, strings.Join(codes, ", "))
	}
	return b.String()
}
