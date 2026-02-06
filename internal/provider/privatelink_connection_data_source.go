package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
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
	ConnectionID        types.String `tfsdk:"connection_id"`
	AzureConnectionName types.String `tfsdk:"azure_connection_name"`
	Region              types.String `tfsdk:"region"`
	SubscriptionID      types.String `tfsdk:"subscription_id"`
	LinkIdentifier      types.String `tfsdk:"link_identifier"`
	State               types.String `tfsdk:"state"`
	IPAddress           types.String `tfsdk:"ip_address"`
	Name                types.String `tfsdk:"name"`
}

func (d *privateLinkConnectionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_privatelink_connection"
}

func (d *privateLinkConnectionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up an existing Azure Private Link connection.",
		MarkdownDescription: `Looks up an existing Azure Private Link connection.

Use this data source to reference a connection that was created outside of Terraform
or in a different Terraform workspace. You can look up by ` + "`connection_id`" + ` or by
` + "`azure_connection_name`" + ` and ` + "`region`" + `.

## Example Usage

### Look up by connection ID

` + "```hcl" + `
data "timescale_privatelink_connection" "by_id" {
  connection_id = "conn-123"
}
` + "```" + `

### Look up by Azure connection name

` + "```hcl" + `
data "timescale_privatelink_connection" "by_name" {
  azure_connection_name = "my-private-endpoint"
  region                = "az-eastus2"
}
` + "```",
		Attributes: map[string]schema.Attribute{
			"connection_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The unique identifier for the connection. Either this or azure_connection_name+region must be provided.",
			},
			"azure_connection_name": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "The Azure private endpoint name to match. " +
					"Azure formats the connection name as '<pe-name>.<guid>', so this matches " +
					"connections where the name starts with this value followed by a dot.",
			},
			"region": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The Timescale region (e.g., az-eastus2). Required when using azure_connection_name.",
			},
			"subscription_id": schema.StringAttribute{
				Computed:    true,
				Description: "The Azure subscription ID.",
			},
			"link_identifier": schema.StringAttribute{
				Computed:    true,
				Description: "The Azure private link identifier.",
			},
			"state": schema.StringAttribute{
				Computed:    true,
				Description: "The state of the connection (e.g., APPROVED, PENDING).",
			},
			"ip_address": schema.StringAttribute{
				Computed:    true,
				Description: "The private IP address of the Azure Private Endpoint.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "The display name for the connection.",
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
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasConnectionID := !config.ConnectionID.IsNull()
	hasAzureName := !config.AzureConnectionName.IsNull()
	hasRegion := !config.Region.IsNull()

	if !hasConnectionID && !hasAzureName {
		resp.Diagnostics.AddAttributeError(
			path.Root("connection_id"),
			"Missing required attribute",
			"Either connection_id or azure_connection_name must be provided.",
		)
		return
	}

	if hasConnectionID && hasAzureName {
		resp.Diagnostics.AddAttributeError(
			path.Root("connection_id"),
			"Conflicting attributes",
			"Cannot specify both connection_id and azure_connection_name. Use one or the other.",
		)
		return
	}

	if hasAzureName && !hasRegion {
		resp.Diagnostics.AddAttributeError(
			path.Root("region"),
			"Missing required attribute",
			"region is required when using azure_connection_name.",
		)
		return
	}

	var found *tsClient.PrivateLinkConnection

	if hasConnectionID {
		connectionID := config.ConnectionID.ValueString()
		tflog.Debug(ctx, "Looking up Private Link connection by ID", map[string]interface{}{
			"connection_id": connectionID,
		})

		// Need to search across all regions since we don't know the region
		regions, err := d.client.ListPrivateLinkAvailableRegions(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Unable to list Private Link regions", err.Error())
			return
		}

		for _, r := range regions {
			connections, err := d.client.ListPrivateLinkConnections(ctx, r.Region)
			if err != nil {
				tflog.Warn(ctx, "Failed to list connections for region", map[string]interface{}{
					"region": r.Region,
					"error":  err.Error(),
				})
				continue
			}
			for _, conn := range connections {
				if conn.ConnectionID == connectionID {
					found = conn
					break
				}
			}
			if found != nil {
				break
			}
		}
	} else {
		region := config.Region.ValueString()
		azureName := config.AzureConnectionName.ValueString()
		tflog.Debug(ctx, "Looking up Private Link connection by Azure name", map[string]interface{}{
			"azure_connection_name": azureName,
			"region":                region,
		})

		connections, err := d.client.ListPrivateLinkConnections(ctx, region)
		if err != nil {
			resp.Diagnostics.AddError("Unable to list Private Link connections", err.Error())
			return
		}

		expectedPrefix := azureName + "."
		for _, conn := range connections {
			if strings.HasPrefix(conn.AzureConnectionName, expectedPrefix) {
				found = conn
				break
			}
		}
	}

	if found == nil {
		if hasConnectionID {
			resp.Diagnostics.AddError(
				"Connection not found",
				fmt.Sprintf("No Private Link connection found with ID '%s'.", config.ConnectionID.ValueString()),
			)
		} else {
			resp.Diagnostics.AddError(
				"Connection not found",
				fmt.Sprintf("No Private Link connection found matching azure_connection_name '%s' in region '%s'.",
					config.AzureConnectionName.ValueString(), config.Region.ValueString()),
			)
		}
		return
	}

	config.ConnectionID = types.StringValue(found.ConnectionID)
	config.AzureConnectionName = types.StringValue(found.AzureConnectionName)
	config.Region = types.StringValue(found.Region)
	config.SubscriptionID = types.StringValue(found.SubscriptionID)
	config.LinkIdentifier = types.StringValue(found.LinkIdentifier)
	config.State = types.StringValue(found.State)
	config.IPAddress = types.StringValue(found.IPAddress)
	config.Name = types.StringValue(found.Name)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
