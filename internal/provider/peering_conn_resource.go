package provider

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vektah/gqlparser/v2/gqlerror"

	tsClient "github.com/timescale/terraform-provider-timescale/internal/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &peeringConnectionResource{}
	_ resource.ResourceWithConfigure = &peeringConnectionResource{}

	ErrPeeringConnRead         = "Error reading Peering Connection"
	ErrPeeringConnCreate       = "Error creating Peering Connection"
	ErrPeeringConnectionUpdate = "Error updating Peering Connection"
)

// NewPeeringConnectionResource is a helper function to simplify the provider implementation.
func NewPeeringConnectionResource() resource.Resource {
	return &peeringConnectionResource{}
}

// peeringConnectionResource is the data source implementation.
type peeringConnectionResource struct {
	client *tsClient.Client
}

type peeringConnectionResourceModel struct {
	ID             types.Int64  `tfsdk:"id"`
	VpcID          types.String `tfsdk:"vpc_id"`
	ProvisionedID  types.String `tfsdk:"provisioned_id"`
	Status         types.String `tfsdk:"status"`
	ErrorMessage   types.String `tfsdk:"error_message"`
	PeerVPCID      types.String `tfsdk:"peer_vpc_id"`
	PeerCIDRBlocks types.List   `tfsdk:"peer_cidr_blocks"`
	PeerCIDR       types.String `tfsdk:"peer_cidr"`
	PeerAccountID  types.String `tfsdk:"peer_account_id"`
	PeerRegionCode types.String `tfsdk:"peer_region_code"`
	TimescaleVPCID types.Int64  `tfsdk:"timescale_vpc_id"`
}

// Metadata returns the data source type name.
func (r *peeringConnectionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_peering_connection"
}

// Read refreshes the Terraform state with the latest data.
func (r *peeringConnectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state peeringConnectionResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	vpc, err := r.client.GetVPCByID(ctx, state.TimescaleVPCID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(ErrPeeringConnRead, err.Error())
		return
	}

	var pcm peeringConnectionResourceModel
	for _, pc := range vpc.PeeringConnections {
		peeringConnID, err := strconv.ParseInt(pc.ID, 10, 64)
		if err != nil {
			resp.Diagnostics.AddError("Parse Error", "could not parse peering connection ID")
		}

		if state.ID.ValueInt64() == peeringConnID {
			pcm.ID = types.Int64Value(peeringConnID)
			pcm.VpcID = types.StringValue(pc.VPCID)
			pcm.ProvisionedID = types.StringValue(pc.ProvisionedID)
			pcm.Status = types.StringValue(pc.Status)
			pcm.PeerAccountID = state.PeerAccountID
			pcm.PeerRegionCode = state.PeerRegionCode
			pcm.PeerVPCID = state.PeerVPCID
			pcm.TimescaleVPCID = state.TimescaleVPCID
			pcm.ErrorMessage = types.StringValue(pc.ErrorMessage)
			pcm.PeerCIDR = types.StringValue("deprecated")

			peerCIDRBlocks, cidrDiags := types.ListValueFrom(ctx, types.StringType, pc.PeerVPC.CIDRBlocks)
			resp.Diagnostics.Append(cidrDiags...)
			if resp.Diagnostics.HasError() {
				return
			}
			pcm.PeerCIDRBlocks = peerCIDRBlocks
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, pcm)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, fmt.Sprintf("error updating terraform state %v", resp.Diagnostics.Errors()))
		return
	}
}

func (r *peeringConnectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "VpcResource.Create")
	var plan peeringConnectionResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if plan.PeerRegionCode.IsNull() {
		resp.Diagnostics.AddError(ErrPeeringConnCreate, "Peer region code is required")
		return
	}
	if plan.PeerVPCID.IsNull() {
		resp.Diagnostics.AddError(ErrPeeringConnCreate, "Peer VPC ID is required")
		return
	}
	if plan.PeerAccountID.IsNull() {
		resp.Diagnostics.AddError(ErrPeeringConnCreate, "Peer Account ID is required")
		return
	}

	var peerCIDRBlocks []string
	if !plan.PeerCIDRBlocks.IsNull() && !plan.PeerCIDRBlocks.IsUnknown() {
		resp.Diagnostics.Append(plan.PeerCIDRBlocks.ElementsAs(ctx, &peerCIDRBlocks, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	pcIDStr, err := r.client.OpenPeerRequest(ctx, plan.TimescaleVPCID.ValueInt64(), plan.PeerVPCID.ValueString(), plan.PeerAccountID.ValueString(), plan.PeerRegionCode.ValueString(), peerCIDRBlocks)
	if err != nil {
		resp.Diagnostics.AddError(ErrPeeringConnCreate, err.Error())
		return
	}
	pcID, err := strconv.ParseInt(pcIDStr, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Parse Error", "could not parse peering connection ID")
		return
	}

	pc, err := r.waitForPCReadiness(ctx, plan.TimescaleVPCID.ValueInt64(), pcID)
	if err != nil {
		resp.Diagnostics.AddError("Create PC Error", "error waiting for PC readiness: "+err.Error())
		return
	}

	plan.ID = types.Int64Value(pcID)
	plan.VpcID = types.StringValue(pc.VPCID)
	plan.Status = types.StringValue(pc.Status)
	plan.ErrorMessage = types.StringValue(pc.ErrorMessage)
	plan.ProvisionedID = types.StringValue(pc.ProvisionedID)
	plan.PeerCIDR = types.StringValue("deprecated")

	// If the API doesn't return CIDR blocks, means they will be populated asynchronously (once the peering is approved).
	// The terraform state will be updated in future operations.
	if len(pc.PeerVPC.CIDRBlocks) == 0 {
		plan.PeerCIDRBlocks = types.ListNull(types.StringType)
	}

	// Set state
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *peeringConnectionResource) waitForPCReadiness(ctx context.Context, vpcID int64, pcID int64) (*tsClient.PeeringConnection, error) {
	tflog.Trace(ctx, "VpcResource.waitForPCReadiness")

	conf := retry.StateChangeConf{
		Target:                    []string{"PENDING"},
		Delay:                     5 * time.Second,
		Timeout:                   5 * time.Minute,
		PollInterval:              5 * time.Second,
		ContinuousTargetOccurence: 1,
		Refresh: func() (result interface{}, state string, err error) {
			vpc, err := r.client.GetVPCByID(ctx, vpcID)
			if err != nil {
				return nil, "", err
			}

			for _, pc := range vpc.PeeringConnections {
				if pc.ID == strconv.FormatInt(pcID, 10) {
					if pc.ProvisionedID == "" {
						// We also wait for a valid provisioned ID, so this resource can be used for the peering acceptance
						return nil, "", err
					}
					return pc, pc.Status, nil
				}
			}
			return nil, "", errors.New("peering connection not found in API response")
		},
	}
	res, err := conf.WaitForStateContext(ctx)
	if err != nil {
		return nil, err
	}
	pc, ok := res.(*tsClient.PeeringConnection)
	if !ok {
		return nil, fmt.Errorf("unexpected type found, expected PeeringConnection but got %T", res)
	}
	return pc, nil
}

func (r *peeringConnectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "PeeringConnectionResource.Delete")
	// TODO: Workaround to avoid deadlocks when many resources try to delete at once
	time.Sleep(10 * time.Second)
	var state peeringConnectionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeletePeeringConnection(ctx, state.TimescaleVPCID.ValueInt64(), state.ID.ValueInt64())
	if err != nil {
		var gqlErr *gqlerror.Error
		if errors.As(err, &gqlErr) {
			if gqlErr.Extensions["status"] != 404 {
				resp.Diagnostics.AddError("Error Deleting Timescale peering connection", err.Error())
				return
			}
		} else {
			resp.Diagnostics.AddError("Error Deleting Timescale peering connection", err.Error())
			return
		}
	}
}

func (r *peeringConnectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "PeeringConnectionResource.Update")

	// Retrieve values from plan
	var plan peeringConnectionResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract CIDR blocks from the plan
	var peerCIDRBlocks []string
	if !plan.PeerCIDRBlocks.IsNull() && !plan.PeerCIDRBlocks.IsUnknown() {
		resp.Diagnostics.Append(plan.PeerCIDRBlocks.ElementsAs(ctx, &peerCIDRBlocks, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if len(peerCIDRBlocks) == 0 {
		resp.Diagnostics.AddError(ErrPeeringConnectionUpdate, "peer_cidr_blocks can not be empty")
		return
	}

	if err := r.client.UpdatePeeringConnectionCIDRs(ctx, plan.TimescaleVPCID.ValueInt64(), plan.ID.ValueInt64(), peerCIDRBlocks); err != nil {
		resp.Diagnostics.AddError(ErrPeeringConnectionUpdate, err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Configure adds the provider configured client to the data source.
func (r *peeringConnectionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	tflog.Trace(ctx, "PeeringConnectionResource.Configure")
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

func (r *peeringConnectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *peeringConnectionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Schema for a peering connection. Import can be done with timescale_vpc_id,peer_account_id,peer_region_code,peer_vpc_id format`,
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "Timescale internal ID for a peering connection",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"vpc_id": schema.StringAttribute{
				Description: "AWS VPC ID of the timescale instance VPC",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"provisioned_id": schema.StringAttribute{
				Description: "AWS ID of the peering connection (starts with pcx-...)",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Description: "Peering connection status",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"error_message": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"peer_account_id": schema.StringAttribute{
				Description: "AWS account ID where the VPC to be paired",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"peer_region_code": schema.StringAttribute{
				Description: "Region code for the VPC to be paired",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"peer_cidr_blocks": schema.ListAttribute{
				Description: "List of CIDR blocks for the VPC to be paired",
				Optional:    true,
				Computed:    true, // If CIDRs are not provided, we will ask the AWS SKD for them (default peer VPC CIDRs).
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"peer_cidr": schema.StringAttribute{
				Description:        "CIDR for the VPC to be paired",
				DeprecationMessage: "Use cidr_blocks instead. This field will be removed in a future version.",
				Computed:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"peer_vpc_id": schema.StringAttribute{
				Description: "AWS ID for the VPC to be paired",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"timescale_vpc_id": schema.Int64Attribute{
				Description: "Timescale internal ID for a vpc",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
					int64planmodifier.RequiresReplace(),
				},
			},
		},
	}
}
