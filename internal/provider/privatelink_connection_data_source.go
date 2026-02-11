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
	ConnectionID         types.String `tfsdk:"connection_id"`
	ProviderConnectionID types.String `tfsdk:"provider_connection_id"`
	CloudProvider        types.String `tfsdk:"cloud_provider"`
	Region               types.String `tfsdk:"region"`
	LinkIdentifier       types.String `tfsdk:"link_identifier"`
	State                types.String `tfsdk:"state"`
	IPAddress            types.String `tfsdk:"ip_address"`
	Name                 types.String `tfsdk:"name"`
}

func (d *privateLinkConnectionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_privatelink_connection"
}

func (d *privateLinkConnectionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up an existing Private Link connection.",
		MarkdownDescription: `Looks up an existing Private Link connection.

Use this data source to reference a connection that was created outside of Terraform
or in a different Terraform workspace. You can look up by ` + "`connection_id`" + ` or by
` + "`provider_connection_id`" + `, ` + "`cloud_provider`" + `, and ` + "`region`" + `.

## Example Usage

### Look up by connection ID

` + "```hcl" + `
data "timescale_privatelink_connection" "by_id" {
  connection_id = "conn-123"
}
` + "```" + `

### Look up by provider connection ID

` + "```hcl" + `
data "timescale_privatelink_connection" "by_provider_id" {
  provider_connection_id = "my-private-endpoint"
  cloud_provider         = "AZURE"
  region                 = "az-eastus2"
}
` + "```",
		Attributes: map[string]schema.Attribute{
			"connection_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The unique identifier for the connection. Either this or provider_connection_id+cloud_provider+region must be provided.",
			},
			"provider_connection_id": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "The cloud provider connection identifier to match. " +
					"For Azure: the private endpoint name. For AWS: the VPC Endpoint ID.",
			},
			"cloud_provider": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The cloud provider: AZURE or AWS. Required when using provider_connection_id.",
			},
			"region": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The Timescale region. Required when using provider_connection_id.",
			},
			"link_identifier": schema.StringAttribute{
				Computed:    true,
				Description: "The private link identifier.",
			},
			"state": schema.StringAttribute{
				Computed:    true,
				Description: "The state of the connection (e.g., APPROVED, PENDING).",
			},
			"ip_address": schema.StringAttribute{
				Computed:    true,
				Description: "The private IP address of the endpoint.",
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
	hasProviderConnID := !config.ProviderConnectionID.IsNull()
	hasRegion := !config.Region.IsNull()
	hasCloudProvider := !config.CloudProvider.IsNull()

	if !hasConnectionID && !hasProviderConnID {
		resp.Diagnostics.AddAttributeError(
			path.Root("connection_id"),
			"Missing required attribute",
			"Either connection_id or provider_connection_id must be provided.",
		)
		return
	}

	if hasConnectionID && hasProviderConnID {
		resp.Diagnostics.AddAttributeError(
			path.Root("connection_id"),
			"Conflicting attributes",
			"Cannot specify both connection_id and provider_connection_id. Use one or the other.",
		)
		return
	}

	if hasProviderConnID && !hasRegion {
		resp.Diagnostics.AddAttributeError(
			path.Root("region"),
			"Missing required attribute",
			"region is required when using provider_connection_id.",
		)
		return
	}

	if hasProviderConnID && !hasCloudProvider {
		resp.Diagnostics.AddAttributeError(
			path.Root("cloud_provider"),
			"Missing required attribute",
			"cloud_provider is required when using provider_connection_id.",
		)
		return
	}

	var found *tsClient.PrivateLinkConnection

	if hasConnectionID {
		connectionID := config.ConnectionID.ValueString()
		tflog.Debug(ctx, "Looking up Private Link connection by ID", map[string]interface{}{
			"connection_id": connectionID,
		})

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
		cloudProvider := config.CloudProvider.ValueString()
		providerConnID := config.ProviderConnectionID.ValueString()
		tflog.Debug(ctx, "Looking up Private Link connection by provider connection ID", map[string]interface{}{
			"provider_connection_id": providerConnID,
			"cloud_provider":         cloudProvider,
			"region":                 region,
		})

		connections, err := d.client.ListPrivateLinkConnections(ctx, region)
		if err != nil {
			resp.Diagnostics.AddError("Unable to list Private Link connections", err.Error())
			return
		}

		switch cloudProvider {
		case "AZURE":
			expectedPrefix := providerConnID + "."
			for _, conn := range connections {
				if strings.HasPrefix(conn.ProviderConnectionID, expectedPrefix) {
					found = conn
					break
				}
			}
		case "AWS":
			for _, conn := range connections {
				if conn.ProviderConnectionID == providerConnID {
					found = conn
					break
				}
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
				fmt.Sprintf("No Private Link connection found matching provider_connection_id '%s' (cloud_provider=%s) in region '%s'.",
					config.ProviderConnectionID.ValueString(), config.CloudProvider.ValueString(), config.Region.ValueString()),
			)
		}
		return
	}

	config.ConnectionID = types.StringValue(found.ConnectionID)
	config.ProviderConnectionID = types.StringValue(found.ProviderConnectionID)
	config.CloudProvider = types.StringValue(found.CloudProvider)
	config.Region = types.StringValue(found.Region)
	config.LinkIdentifier = types.StringValue(found.LinkIdentifier)
	config.State = types.StringValue(found.State)
	config.IPAddress = types.StringValue(found.IPAddress)
	config.Name = types.StringValue(found.Name)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
