package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	tsClient "github.com/timescale/terraform-provider-timescale/internal/client"
)

var (
	_ resource.Resource                = &privateLinkAuthorizationResource{}
	_ resource.ResourceWithConfigure   = &privateLinkAuthorizationResource{}
	_ resource.ResourceWithImportState = &privateLinkAuthorizationResource{}
)

func NewPrivateLinkAuthorizationResource() resource.Resource {
	return &privateLinkAuthorizationResource{}
}

type privateLinkAuthorizationResource struct {
	client *tsClient.Client
}

type privateLinkAuthorizationResourceModel struct {
	ID             types.String `tfsdk:"id"`
	SubscriptionID types.String `tfsdk:"subscription_id"`
	Name           types.String `tfsdk:"name"`
}

func (r *privateLinkAuthorizationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_privatelink_authorization"
}

func (r *privateLinkAuthorizationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Authorizes an Azure subscription to connect via Private Link. Import using the Azure subscription ID: `terraform import timescale_privatelink_authorization.example <subscription_id>`.",
		MarkdownDescription: `Authorizes an Azure subscription to connect via Private Link.

This resource authorizes an Azure subscription to create Private Endpoint connections
to the Timescale Private Link Service. Once authorized, Private Endpoint connections
from this subscription will be auto-approved.

## Workflow

1. Create this authorization resource with your Azure subscription ID
2. Create an Azure Private Endpoint pointing to the Timescale Private Link Service alias
3. The connection will be automatically approved
4. Use ` + "`timescale_privatelink_connection`" + ` to configure the connection`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Resource identifier (same as subscription_id).",
			},
			"subscription_id": schema.StringAttribute{
				Required:    true,
				Description: "The Azure subscription ID to authorize.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "A friendly name for this authorization.",
			},
		},
	}
}

func (r *privateLinkAuthorizationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	r.client = client
}

func (r *privateLinkAuthorizationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan privateLinkAuthorizationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	subscriptionID := plan.SubscriptionID.ValueString()
	name := plan.Name.ValueString()

	auth, err := r.client.CreatePrivateLinkAuthorization(ctx, subscriptionID, name)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Private Link authorization", err.Error())
		return
	}

	plan.ID = types.StringValue(auth.SubscriptionID)
	plan.SubscriptionID = types.StringValue(auth.SubscriptionID)
	plan.Name = types.StringValue(auth.Name)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *privateLinkAuthorizationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state privateLinkAuthorizationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	subscriptionID := state.SubscriptionID.ValueString()

	authorizations, err := r.client.ListPrivateLinkAuthorizations(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Private Link authorizations", err.Error())
		return
	}

	var auth *tsClient.PrivateLinkAuthorization
	for _, a := range authorizations {
		if a.SubscriptionID == subscriptionID {
			auth = a
			break
		}
	}

	if auth == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = types.StringValue(auth.SubscriptionID)
	state.Name = types.StringValue(auth.Name)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *privateLinkAuthorizationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan privateLinkAuthorizationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	subscriptionID := plan.SubscriptionID.ValueString()
	name := plan.Name.ValueString()

	auth, err := r.client.UpdatePrivateLinkAuthorization(ctx, subscriptionID, name)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update Private Link authorization", err.Error())
		return
	}

	plan.ID = types.StringValue(auth.SubscriptionID)
	plan.Name = types.StringValue(auth.Name)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *privateLinkAuthorizationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state privateLinkAuthorizationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	subscriptionID := state.SubscriptionID.ValueString()
	tflog.Info(ctx, "Deleting Private Link authorization", map[string]interface{}{
		"subscription_id": subscriptionID,
	})

	if err := r.client.DeletePrivateLinkAuthorization(ctx, subscriptionID); err != nil {
		resp.Diagnostics.AddError("Failed to delete Private Link authorization", err.Error())
		return
	}
}

func (r *privateLinkAuthorizationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("subscription_id"), req, resp)
}
