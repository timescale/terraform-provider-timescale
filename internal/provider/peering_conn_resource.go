package provider

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
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

	ErrPeeringConnRead   = "Error reading Peering Connection"
	ErrPeeringConnCreate = "Error creating Peering Connection"
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
	PeerCIDR       types.String `tfsdk:"peer_cidr"`
	PeerAccountID  types.String `tfsdk:"peer_account_id"`
	PeerRegionCode types.String `tfsdk:"peer_region_code"`
	TimescaleVPCID types.Int64  `tfsdk:"timescale_vpc_id"`
}

var (
	PeeringConnectionsType = types.ObjectType{
		AttrTypes: PeeringConnectionType,
	}

	PeeringConnectionType = map[string]attr.Type{
		"id":               types.Int64Type,
		"vpc_id":           types.StringType,
		"provisioned_id":   types.StringType,
		"status":           types.StringType,
		"error_message":    types.StringType,
		"peer_vpc_id":      types.StringType,
		"peer_cidr":        types.StringType,
		"peer_account_id":  types.StringType,
		"peer_region_code": types.StringType,
		"timescale_vpc_id": types.Int64Type,
	}
)

// Metadata returns the data source type name.
func (r *peeringConnectionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_peering_connection"
}

// Read refreshes the Terraform state with the latest data.
func (r *peeringConnectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Fetch these attributes, used in case of import.
	var TimescaleVPCID int64
	var PeerAccountID, PeerRegionCode, PeerVPCID string
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("timescale_vpc_id"), &TimescaleVPCID)...)
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("peer_account_id"), &PeerAccountID)...)
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("peer_region_code"), &PeerRegionCode)...)
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("peer_vpc_id"), &PeerVPCID)...)

	// Read model
	var state peeringConnectionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// replace model with import attributes
	if PeerAccountID != "" {
		state.PeerAccountID = types.StringValue(PeerAccountID)
	}
	if TimescaleVPCID > 0 {
		state.TimescaleVPCID = types.Int64Value(TimescaleVPCID)
	}
	if PeerRegionCode != "" {
		state.PeerRegionCode = types.StringValue(PeerRegionCode)
	}
	if PeerVPCID != "" {
		state.PeerVPCID = types.StringValue(PeerVPCID)
	}
	var vpc *tsClient.VPC
	var err error

	if state.TimescaleVPCID.IsNull() {
		resp.Diagnostics.AddError(ErrPeeringConnRead, "error must provide TimescaleVpcID")
		return
	}
	vpc, err = r.client.GetVPCByID(ctx, state.TimescaleVPCID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(ErrPeeringConnRead, err.Error())
		return
	}

	var pcm peeringConnectionResourceModel
	for _, pc := range vpc.PeeringConnections {
		if state.PeerAccountID.ValueString() == pc.PeerVPC.AccountID && state.PeerRegionCode.ValueString() == pc.PeerVPC.RegionCode && state.PeerVPCID.ValueString() == pc.PeerVPC.ID {
			peeringConnID, err := strconv.ParseInt(pc.ID, 10, 64)
			if err != nil {
				resp.Diagnostics.AddError("Parse Error", "could not parse peering connection ID")
			}
			pcm.ID = types.Int64Value(peeringConnID)
			pcm.VpcID = types.StringValue(pc.VPCID)
			pcm.ProvisionedID = types.StringValue(pc.ProvisionedID)
			pcm.Status = types.StringValue(pc.Status)
			pcm.PeerAccountID = state.PeerAccountID
			pcm.PeerRegionCode = state.PeerRegionCode
			pcm.PeerVPCID = state.PeerVPCID
			pcm.TimescaleVPCID = state.TimescaleVPCID
			if pc.ErrorMessage != "" {
				pcm.ErrorMessage = types.StringValue(pc.ErrorMessage)
			}
			if pc.PeerVPC.CIDR != "" {
				pcm.PeerCIDR = types.StringValue(pc.PeerVPC.CIDR)
			} else {
				pcm.PeerCIDR = types.StringNull()
			}
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

	err := r.client.OpenPeerRequest(ctx, plan.TimescaleVPCID.ValueInt64(), plan.PeerVPCID.ValueString(), plan.PeerAccountID.ValueString(), plan.PeerRegionCode.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(ErrPeeringConnCreate, err.Error())
		return
	}

	var pcm peeringConnectionResourceModel
	pcm.PeerRegionCode = plan.PeerRegionCode
	pcm.PeerVPCID = plan.PeerVPCID
	pcm.PeerAccountID = plan.PeerAccountID
	pcm.TimescaleVPCID = plan.TimescaleVPCID
	pcm.ErrorMessage = types.StringNull()
	pcm.ID = types.Int64Null()
	pcm.PeerCIDR = types.StringNull()
	pcm.Status = types.StringNull()
	pcm.VpcID = types.StringNull()

	// Set state
	resp.Diagnostics.Append(resp.State.Set(ctx, pcm)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *peeringConnectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "PeeringConnectionResource.Delete")
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
	var plan, state peeringConnectionResourceModel
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if plan != state {
		resp.Diagnostics.AddError("Error updating Peering Connection", "Do not support peering connection updates")
		return
	}
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

	if len(idParts) != 4 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" || idParts[3] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: timescale_vpc_id,peer_account_id,peer_region_code,peer_vpc_id Got: %q", req.ID),
		)
		return
	}

	vpcID, err := strconv.ParseInt(idParts[0], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Parse Error", "could not parse peering connection ID")
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("timescale_vpc_id"), vpcID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("peer_account_id"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("peer_region_code"), idParts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("peer_vpc_id"), idParts[3])...)
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
			"peer_cidr": schema.StringAttribute{
				Description: "CIDR for the VPC to be paired",
				Computed:    true,
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
