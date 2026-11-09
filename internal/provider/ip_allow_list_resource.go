package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	tsClient "github.com/timescale/terraform-provider-timescale/internal/client"
)

var (
	_ resource.Resource                = &ipAllowListResource{}
	_ resource.ResourceWithConfigure   = &ipAllowListResource{}
	_ resource.ResourceWithImportState = &ipAllowListResource{}
)

// NewIPAllowListResource is a helper function to simplify the provider implementation.
func NewIPAllowListResource() resource.Resource {
	return &ipAllowListResource{}
}

type ipAllowListResource struct {
	client *tsClient.Client
}

type ipAllowListResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Description types.String `tfsdk:"description"`
	CIDRBlocks  types.Set    `tfsdk:"cidr_blocks"`
	ServiceIDs  types.Set    `tfsdk:"service_ids"`
}

func (r *ipAllowListResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ip_allow_list"
}

func (r *ipAllowListResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An IP allow list for the project, optionally bound to services. " +
			"Import using the numeric list ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the IP allow list.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				Description: "A description of the IP allow list.",
				Required:    true,
			},
			"cidr_blocks": schema.SetAttribute{
				Description: "IPv4 addresses or CIDR blocks allowed to connect, e.g. `203.0.113.7/32`. " +
					"At least one entry is required. To delete the list, remove the resource from your configuration.",
				ElementType: types.StringType,
				Required:    true,
				Validators:  []validator.Set{setvalidator.SizeAtLeast(1)},
			},
			"service_ids": schema.SetAttribute{
				Description: "IDs of the services this list is bound to. If omitted, bindings are not managed by Terraform.",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ipAllowListResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func setToStrings(ctx context.Context, s types.Set) ([]string, error) {
	var out []string
	if s.IsNull() || s.IsUnknown() {
		return out, nil
	}
	diags := s.ElementsAs(ctx, &out, false)
	if diags.HasError() {
		return nil, fmt.Errorf("%v", diags.Errors())
	}
	return out, nil
}

func diffStrings(have, want []string) (add, remove []string) {
	h := map[string]bool{}
	for _, s := range have {
		h[s] = true
	}
	w := map[string]bool{}
	for _, s := range want {
		w[s] = true
		if !h[s] {
			add = append(add, s)
		}
	}
	for _, s := range have {
		if !w[s] {
			remove = append(remove, s)
		}
	}
	return add, remove
}

func (r *ipAllowListResource) syncBindings(ctx context.Context, listID int64, have, want []string) error {
	add, remove := diffStrings(have, want)
	for _, id := range remove {
		if err := r.client.UnbindIPAllowList(ctx, id, listID); err != nil {
			return fmt.Errorf("unbinding service %s: %w", id, err)
		}
	}
	for _, id := range add {
		if err := r.client.BindIPAllowList(ctx, id, listID); err != nil {
			return fmt.Errorf("binding service %s: %w", id, err)
		}
	}
	return nil
}

func (r *ipAllowListResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ipAllowListResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cidrs, err := setToStrings(ctx, plan.CIDRBlocks)
	if err != nil {
		resp.Diagnostics.AddError("Invalid cidr_blocks", err.Error())
		return
	}
	services, err := setToStrings(ctx, plan.ServiceIDs)
	if err != nil {
		resp.Diagnostics.AddError("Invalid service_ids", err.Error())
		return
	}

	listID, err := r.client.ConfigureIPAllowList(ctx, nil, plan.Description.ValueString(), cidrs)
	if err != nil {
		resp.Diagnostics.AddError("Error creating IP allow list", err.Error())
		return
	}
	plan.ID = types.StringValue(strconv.FormatInt(listID, 10))
	// Persist the ID right away so a failed binding doesn't orphan the list.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), plan.ID)...)

	if err := r.syncBindings(ctx, listID, nil, services); err != nil {
		resp.Diagnostics.AddError("Error binding IP allow list", err.Error())
		return
	}
	if plan.ServiceIDs.IsUnknown() {
		plan.ServiceIDs = types.SetValueMust(types.StringType, nil)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *ipAllowListResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ipAllowListResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	listID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", err.Error())
		return
	}
	list, err := r.client.GetIPAllowList(ctx, listID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading IP allow list", err.Error())
		return
	}
	if list == nil {
		tflog.Info(ctx, "IP allow list no longer exists, removing from state")
		resp.State.RemoveResource(ctx)
		return
	}
	state.Description = types.StringValue(list.Description)
	cidrs, diags := types.SetValueFrom(ctx, types.StringType, list.CIDRBlocks)
	resp.Diagnostics.Append(diags...)
	services, diags := types.SetValueFrom(ctx, types.StringType, list.AssociatedServices)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.CIDRBlocks = cidrs
	state.ServiceIDs = services
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *ipAllowListResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ipAllowListResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	listID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", err.Error())
		return
	}
	cidrs, err := setToStrings(ctx, plan.CIDRBlocks)
	if err != nil {
		resp.Diagnostics.AddError("Invalid cidr_blocks", err.Error())
		return
	}
	if _, err := r.client.ConfigureIPAllowList(ctx, &listID, plan.Description.ValueString(), cidrs); err != nil {
		resp.Diagnostics.AddError("Error updating IP allow list", err.Error())
		return
	}
	have, _ := setToStrings(ctx, state.ServiceIDs)
	want, err := setToStrings(ctx, plan.ServiceIDs)
	if err != nil {
		resp.Diagnostics.AddError("Invalid service_ids", err.Error())
		return
	}
	if err := r.syncBindings(ctx, listID, have, want); err != nil {
		resp.Diagnostics.AddError("Error updating IP allow list bindings", err.Error())
		return
	}
	plan.ID = state.ID
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *ipAllowListResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ipAllowListResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	listID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", err.Error())
		return
	}
	have, _ := setToStrings(ctx, state.ServiceIDs)
	if err := r.syncBindings(ctx, listID, have, nil); err != nil {
		resp.Diagnostics.AddError("Error unbinding IP allow list", err.Error())
		return
	}
	if err := r.client.DeleteIPAllowList(ctx, listID); err != nil {
		resp.Diagnostics.AddError("Error deleting IP allow list", err.Error())
		return
	}
}

func (r *ipAllowListResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
