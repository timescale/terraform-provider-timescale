package provider

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"strconv"
	"strings"
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
	PeerTGWID      types.String `tfsdk:"peer_tgw_id"`
	PeerCIDRBlocks types.List   `tfsdk:"peer_cidr_blocks"`
	PeerCIDR       types.String `tfsdk:"peer_cidr"`
	PeerAccountID  types.String `tfsdk:"peer_account_id"`
	PeerRegionCode types.String `tfsdk:"peer_region_code"`
	TimescaleVPCID types.Int64  `tfsdk:"timescale_vpc_id"`
	PeeringType    types.String `tfsdk:"peering_type"`
}

// Metadata returns the data source type name.
func (r *peeringConnectionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_peering_connection"
}

// Read refreshes the Terraform state with the latest data.
func (r *peeringConnectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "PeeringConnectionResource.Read")
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

	vpcID, err := strconv.ParseInt(vpc.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Unable to convert vpc ID", err.Error())
		return
	}

	var pcm peeringConnectionResourceModel
	for _, pc := range vpc.PeeringConnections {
		pcID, err := strconv.ParseInt(pc.ID, 10, 64)
		if err != nil {
			resp.Diagnostics.AddError("Parse Error", "could not parse peering connection ID")
			return
		}

		if state.ID.ValueInt64() == pcID {
			if pc.ErrorMessage != "" {
				pcm.ErrorMessage = types.StringValue(pc.ErrorMessage)
			}

			pcm.ID = types.Int64Value(pcID)
			pcm.VpcID = types.StringValue(vpc.ProvisionedID)
			pcm.Status = types.StringValue(pc.Status)
			pcm.ProvisionedID = types.StringValue(pc.ProvisionedID)
			if pc.PeerVPC.ID != "" {
				if strings.HasPrefix(pc.PeerVPC.ID, "vpc-") {
					pcm.PeerVPCID = types.StringValue(pc.PeerVPC.ID)
					pcm.PeeringType = types.StringValue("vpc")
				} else if strings.HasPrefix(pc.PeerVPC.ID, "tgw-") {
					pcm.PeerTGWID = types.StringValue(pc.PeerVPC.ID)
					pcm.PeeringType = types.StringValue("tgw")
				} else {
					resp.Diagnostics.AddError("Peering type error", "Received an invalid peering provisioned ID: "+pc.PeerVPC.ID)
					return
				}
			}
			pcm.PeerAccountID = types.StringValue(pc.PeerVPC.AccountID)

			peerCIDRBlocks, cidrDiags := types.ListValueFrom(ctx, types.StringType, pc.PeerVPC.CIDRBlocks)
			resp.Diagnostics.Append(cidrDiags...)
			if resp.Diagnostics.HasError() {
				return
			}
			pcm.PeerCIDRBlocks = peerCIDRBlocks

			pcm.PeerRegionCode = types.StringValue(pc.PeerVPC.RegionCode)
			pcm.TimescaleVPCID = types.Int64Value(vpcID)
			pcm.PeerCIDR = types.StringValue("deprecated")

		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, pcm)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, fmt.Sprintf("error updating terraform state %v", resp.Diagnostics.Errors()))
		return
	}
}

func (r *peeringConnectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "PeeringConnectionResource.Create")
	var plan peeringConnectionResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if plan.PeerRegionCode.IsNull() {
		resp.Diagnostics.AddError(ErrPeeringConnCreate, "Peer region code is required")
		return
	}
	if plan.PeerAccountID.IsNull() {
		resp.Diagnostics.AddError(ErrPeeringConnCreate, "Peer Account ID is required")
		return
	}

	// Validate that either PeerVPCID or PeerTGWID is provided, but not both
	if plan.PeerVPCID.IsNull() && plan.PeerTGWID.IsNull() {
		resp.Diagnostics.AddError(ErrPeeringConnCreate, "Only one of peer_vpc_id or peer_tgw_id can be provided")
		return
	}
	if !plan.PeerVPCID.IsNull() && !plan.PeerTGWID.IsNull() {
		resp.Diagnostics.AddError(ErrPeeringConnCreate, "Only one of peer_vpc_id or peer_tgw_id can be provided")
		return
	}

	var peerCIDRBlocks []string
	if !plan.PeerCIDRBlocks.IsNull() && !plan.PeerCIDRBlocks.IsUnknown() {
		resp.Diagnostics.Append(plan.PeerCIDRBlocks.ElementsAs(ctx, &peerCIDRBlocks, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if !plan.PeerTGWID.IsNull() && (plan.PeerCIDRBlocks.IsNull() || len(peerCIDRBlocks) == 0) {
		resp.Diagnostics.AddError(ErrPeeringConnCreate, "peer_cidr_blocks is required for Transit Gateway peering")
		return
	}

	var pcIDStr string
	var err error

	if !plan.PeerVPCID.IsNull() {
		plan.PeeringType = types.StringValue("vpc")
		pcIDStr, err = r.client.OpenPeerRequest(ctx, plan.TimescaleVPCID.ValueInt64(), plan.PeerVPCID.ValueString(), plan.PeerAccountID.ValueString(), plan.PeerRegionCode.ValueString(), peerCIDRBlocks)
	} else {
		plan.PeeringType = types.StringValue("tgw")
		pcIDStr, err = r.client.OpenPeerRequest(ctx, plan.TimescaleVPCID.ValueInt64(), plan.PeerTGWID.ValueString(), plan.PeerAccountID.ValueString(), plan.PeerRegionCode.ValueString(), peerCIDRBlocks)
	}

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
	tflog.Trace(ctx, "PeeringConnectionResource.waitForPCReadiness", map[string]interface{}{
		"vpcID": vpcID,
		"pcID":  pcID,
	})

	conf := retry.StateChangeConf{
		Target:                    []string{"PENDING", "ACTIVE", "APPROVED"},
		Delay:                     30 * time.Second,
		Timeout:                   10 * time.Minute,
		PollInterval:              15 * time.Second,
		NotFoundChecks:            40,
		ContinuousTargetOccurence: 1,
		Refresh: func() (result interface{}, state string, err error) {
			tflog.Debug(ctx, "Checking peering connection status", map[string]interface{}{
				"vpcID": vpcID,
				"pcID":  pcID,
			})

			vpc, err := r.client.GetVPCByID(ctx, vpcID)
			if err != nil {
				tflog.Error(ctx, "Error getting VPC", map[string]interface{}{
					"error": err.Error(),
				})
				return nil, "", err
			}

			for _, pc := range vpc.PeeringConnections {
				tflog.Debug(ctx, "Found peering connection", map[string]interface{}{
					"pcID":          pc.ID,
					"provisionedID": pc.ProvisionedID,
					"status":        pc.Status,
				})

				if pc.ID == strconv.FormatInt(pcID, 10) {
					if pc.ProvisionedID == "" {
						tflog.Debug(ctx, "Provisioned ID not yet available, continuing to wait")
						return nil, "", nil
					}

					if pc.Status == "INVALID" || pc.Status == "TIMEOUT" || pc.Status == "DISABLED" {
						return nil, pc.Status, fmt.Errorf("peering connection failed: %s", pc.ErrorMessage)
					}

					tflog.Info(ctx, "Peering connection ready", map[string]interface{}{
						"status":        pc.Status,
						"provisionedID": pc.ProvisionedID,
					})
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

	// TODO: Workaround to ensure the peering is deleted (async operation).
	time.Sleep(30 * time.Second)
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
	idParts := strings.Split(req.ID, ",")

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: peering_connection_id,timescale_vpc_id. Got: %q", req.ID),
		)
		return
	}

	pcID, err := strconv.ParseInt(idParts[0], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Convert pcID", err.Error())
		return
	}

	vpcID, err := strconv.ParseInt(idParts[1], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Convert vpcID", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), pcID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("timescale_vpc_id"), vpcID)...)
}

func (r *peeringConnectionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Schema for a peering connection (VPC or Transit Gateway). Import can be done with `peering_connection_id,timescale_vpc_id` format. Both internal IDs can be retrieved using the timescale_vpcs datasource.",
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
				Description: "AWS ID of the peering connection (starts with pcx-... for VPC peering or tgw-... for TGW.)",
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
				Description: "AWS account ID where the VPC or Transit Gateway to be paired is located",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"peer_region_code": schema.StringAttribute{
				Description: "Region code for the VPC or Transit Gateway to be paired",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"peer_cidr_blocks": schema.ListAttribute{
				Description: "List of CIDR blocks for the peering connection. Required for Transit Gateway peering, optional for VPC peering",
				Optional:    true,
				Computed:    true, // If CIDRs are not provided for VPC peering, we will ask the AWS SDK for them (default peer VPC CIDRs).
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
				Description: "AWS ID for the VPC to be paired. Mutually exclusive with peer_tgw_id",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"peer_tgw_id": schema.StringAttribute{
				Description: "AWS ID for the Transit Gateway to be paired. Mutually exclusive with peer_vpc_id",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
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
			"peering_type": schema.StringAttribute{
				Description: "Type of peering connection (vpc or tgw)",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}
