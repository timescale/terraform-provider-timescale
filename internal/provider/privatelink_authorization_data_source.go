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

var _ datasource.DataSource = &privateLinkAuthorizationDataSource{}

func NewPrivateLinkAuthorizationDataSource() datasource.DataSource {
	return &privateLinkAuthorizationDataSource{}
}

type privateLinkAuthorizationDataSource struct {
	client *tsClient.Client
}

type privateLinkAuthorizationDataSourceModel struct {
	SubscriptionID types.String `tfsdk:"subscription_id"`
	Name           types.String `tfsdk:"name"`
}

func (d *privateLinkAuthorizationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_privatelink_authorization"
}

func (d *privateLinkAuthorizationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up an existing Azure Private Link authorization by subscription ID.",
		MarkdownDescription: `Looks up an existing Azure Private Link authorization by subscription ID.

Use this data source to reference an authorization that was created outside of Terraform
or in a different Terraform workspace.

## Example Usage

` + "```hcl" + `
data "timescale_privatelink_authorization" "existing" {
  subscription_id = "00000000-0000-0000-0000-000000000000"
}

output "authorization_name" {
  value = data.timescale_privatelink_authorization.existing.name
}
` + "```",
		Attributes: map[string]schema.Attribute{
			"subscription_id": schema.StringAttribute{
				Required:    true,
				Description: "The Azure subscription ID to look up.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "The friendly name for this authorization.",
			},
		},
	}
}

func (d *privateLinkAuthorizationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *privateLinkAuthorizationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config privateLinkAuthorizationDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	subscriptionID := config.SubscriptionID.ValueString()
	tflog.Debug(ctx, "Looking up Private Link authorization", map[string]interface{}{
		"subscription_id": subscriptionID,
	})

	authorizations, err := d.client.ListPrivateLinkAuthorizations(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Private Link authorizations", err.Error())
		return
	}

	var found *tsClient.PrivateLinkAuthorization
	for _, auth := range authorizations {
		if auth.SubscriptionID == subscriptionID {
			found = auth
			break
		}
	}

	if found == nil {
		resp.Diagnostics.AddError(
			"Authorization not found",
			fmt.Sprintf("No Private Link authorization found for subscription ID '%s'.", subscriptionID),
		)
		return
	}

	config.Name = types.StringValue(found.Name)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
