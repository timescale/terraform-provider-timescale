package provider

import (
	"context"
	"fmt"
	"strings"

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
	ID            types.String `tfsdk:"id"`
	PrincipalID   types.String `tfsdk:"principal_id"`
	CloudProvider types.String `tfsdk:"cloud_provider"`
	Name          types.String `tfsdk:"name"`
}

func (r *privateLinkAuthorizationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_privatelink_authorization"
}

func (r *privateLinkAuthorizationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Authorizes a cloud account to connect via Private Link. Import using `cloud_provider,principal_id` format: `terraform import timescale_privatelink_authorization.example AZURE,<principal_id>`.",
		MarkdownDescription: `Authorizes a cloud account to connect via Private Link.

This resource authorizes an Azure subscription or AWS account to create Private Endpoint
or VPC Endpoint connections to the Timescale Private Link Service. Once authorized,
connections from this account will be auto-approved.

## Workflow

1. Create this authorization resource with your principal ID and cloud provider
2. Create an Azure Private Endpoint or AWS VPC Endpoint pointing to the Timescale service
3. The connection will be automatically approved
4. Use ` + "`timescale_privatelink_connection`" + ` to configure the connection`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Resource identifier (same as principal_id).",
			},
			"principal_id": schema.StringAttribute{
				Required:    true,
				Description: "The Azure subscription ID or AWS account ID to authorize.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cloud_provider": schema.StringAttribute{
				Required:    true,
				Description: "The cloud provider: AZURE or AWS.",
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

	principalID := plan.PrincipalID.ValueString()
	cloudProvider := plan.CloudProvider.ValueString()
	name := plan.Name.ValueString()

	auth, err := r.client.CreatePrivateLinkAuthorization(ctx, principalID, cloudProvider, name)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Private Link authorization", err.Error())
		return
	}

	plan.ID = types.StringValue(principalID)
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

	principalID := state.PrincipalID.ValueString()
	cloudProvider := state.CloudProvider.ValueString()

	authorizations, err := r.client.ListPrivateLinkAuthorizations(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Private Link authorizations", err.Error())
		return
	}

	var auth *tsClient.PrivateLinkAuthorization
	for _, a := range authorizations {
		if a.PrincipalID == principalID && a.CloudProvider == cloudProvider {
			auth = a
			break
		}
	}

	if auth == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = types.StringValue(principalID)
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

	principalID := plan.PrincipalID.ValueString()
	cloudProvider := plan.CloudProvider.ValueString()
	name := plan.Name.ValueString()

	auth, err := r.client.UpdatePrivateLinkAuthorization(ctx, principalID, cloudProvider, name)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update Private Link authorization", err.Error())
		return
	}

	plan.ID = types.StringValue(principalID)
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

	principalID := state.PrincipalID.ValueString()
	cloudProvider := state.CloudProvider.ValueString()
	tflog.Info(ctx, "Deleting Private Link authorization", map[string]interface{}{
		"principal_id":   principalID,
		"cloud_provider": cloudProvider,
	})

	if err := r.client.DeletePrivateLinkAuthorization(ctx, principalID, cloudProvider); err != nil {
		resp.Diagnostics.AddError("Failed to delete Private Link authorization", err.Error())
		return
	}
}

func (r *privateLinkAuthorizationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, ",")

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: cloud_provider,principal_id (e.g. AZURE,<principal_id> or AWS,<account_id>). Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cloud_provider"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("principal_id"), idParts[1])...)
}
