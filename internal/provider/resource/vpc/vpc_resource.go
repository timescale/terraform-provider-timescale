package vpc

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	tsClient "github.com/timescale/terraform-provider-timescale/internal/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &vpcsResource{}
	_ resource.ResourceWithConfigure = &vpcsResource{}

	regionCodes = []string{"us-east-1", "eu-west-1", "us-west-2", "eu-central-1", "ap-southeast-2"}
)

// NewVpcsResource is a helper function to simplify the provider implementation.
func NewVpcsResource() resource.Resource {
	return &vpcsResource{}
}

// vpcsResource is the data source implementation.
type vpcsResource struct {
	client *tsClient.Client
}

// vpcsResourceModel maps the data source schema data.
type vpcsResourceModel struct {
	Vpcs []vpcResourceModel `tfsdk:"vpcs"`
	// following is a placeholder, required by terraform to run test suite
	ID types.String `tfsdk:"id"`
}

// vpcResourceModel maps vpcs schema data.
type vpcResourceModel struct {
	ID                 types.Int64  `tfsdk:"id"`
	ProvisionedID      types.String `tfsdk:"provisioned_id"`
	ProjectID          types.String `tfsdk:"project_id"`
	CIDR               types.String `tfsdk:"cidr"`
	Name               types.String `tfsdk:"name"`
	RegionCode         types.String `tfsdk:"region_code"`
	Status             types.String `tfsdk:"status"`
	ErrorMessage       types.String `tfsdk:"error_message"`
	Created            types.String `tfsdk:"created"`
	Updated            types.String `tfsdk:"updated"`
	PeeringConnections types.List   `tfsdk:"peering_connections"`
}

type peeringConnectionResourceModel struct {
	ID           types.Int64  `tfsdk:"id"`
	VpcID        types.Int64  `tfsdk:"vpc_id"`
	Status       types.String `tfsdk:"status"`
	ErrorMessage types.String `tfsdk:"error_message"`
	PeerVpcs     types.List   `tfsdk:"peer_vpc"`
}

type peerVpcModel struct {
	ID         types.Int64  `tfsdk:"id"`
	CIDR       types.String `tfsdk:"cidr"`
	AccountID  types.String `tfsdk:"account_id"`
	RegionCode types.String `tfsdk:"region_code"`
}

var (
	PeerVpcType = types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"id":          types.Int64Type,
			"cidr":        types.StringType,
			"account_id":  types.StringType,
			"region_code": types.StringType,
		},
	}

	PeeringConnectionsType = types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"id":            types.Int64Type,
			"vpc_id":        types.Int64Type,
			"status":        types.StringType,
			"error_message": types.StringType,
			"peer_vpc":      types.ListType{ElemType: PeerVpcType},
		},
	}
)

// Metadata returns the data source type name.
func (r *vpcsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpcs"
}

// Read refreshes the Terraform state with the latest data.
func (r *vpcsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "VpcResource.Read")
	var state vpcsResourceModel

	vpcs, err := r.client.GetVPCs(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Vpcs",
			err.Error(),
		)
		return
	}
	// Map response body to model
	for _, vpc := range vpcs {
		vpcId, err := strconv.ParseInt(vpc.ID, 10, 64)
		if err != nil {
			resp.Diagnostics.AddError("Unable to Convert Vpc ID", err.Error())
			return
		}
		vpcState := vpcResourceModel{
			ID:                 types.Int64Value(vpcId),
			Name:               types.StringValue(vpc.Name),
			ProvisionedID:      types.StringValue(vpc.ProvisionedID),
			ProjectID:          types.StringValue(vpc.ProjectID),
			CIDR:               types.StringValue(vpc.CIDR),
			RegionCode:         types.StringValue(vpc.RegionCode),
			Status:             types.StringValue(vpc.Status),
			ErrorMessage:       types.StringValue(vpc.ErrorMessage),
			Created:            types.StringValue(vpc.Created),
			Updated:            types.StringValue(vpc.Updated),
			PeeringConnections: types.ListUnknown(PeeringConnectionsType),
		}

		for _, peeringConn := range vpc.PeeringConnections {
			peeringConnID, err := strconv.ParseInt(peeringConn.ID, 10, 64)
			if err != nil {
				resp.Diagnostics.AddError("Unable to Convert Vpc ID", err.Error())
				return
			}
			peeringConnVpcID, err := strconv.ParseInt(peeringConn.VpcID, 10, 64)
			if err != nil {
				resp.Diagnostics.AddError("Unable to Convert Vpc ID", err.Error())
				return
			}
			peerConn := peeringConnectionResourceModel{
				ID:           types.Int64Value(peeringConnID),
				VpcID:        types.Int64Value(peeringConnVpcID),
				Status:       types.StringValue(peeringConn.Status),
				ErrorMessage: types.StringValue(peeringConn.ErrorMessage),
			}
			for _, peerVpc := range peeringConn.PeerVpcs {
				peerVpcId, err := strconv.ParseInt(peerVpc.ID, 10, 64)
				if err != nil {
					resp.Diagnostics.AddError("Unable to Convert Vpc ID", err.Error())
					return
				}
				peerConn.PeerVpcs = append(peerConn.PeerVpcs, &peerVpcModel{
					ID:         types.Int64Value(peerVpcId),
					AccountID:  types.StringValue(peerVpc.AccountID),
					CIDR:       types.StringValue(peerVpc.CIDR),
					RegionCode: types.StringValue(peerVpc.RegionCode),
				})
			}
			vpcState.PeeringConnections = append(vpcState.PeeringConnections, peerConn)
		}
		state.Vpcs = append(state.Vpcs, vpcState)
	}
	// this is a placeholder, required by terraform to run test suite
	state.ID = types.StringValue(fmt.Sprintf("placeholder %v", len(vpcs)))
	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Create creates a VPC shell
func (r *vpcsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "VpcResource.Create")
	var plan vpcResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	vpc, err := r.client.CreateVPC(ctx, plan.Name.ValueString(), plan.CIDR.ValueString(), plan.RegionCode.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to Create Vpc %v", plan),
			err.Error(),
		)
		return
	}

	vpcId, err := strconv.ParseInt(vpc.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Convert Vpc ID", err.Error())
		return
	}
	vpcState := vpcResourceModel{
		ID:                 types.Int64Value(vpcId),
		Name:               types.StringValue(vpc.Name),
		ProvisionedID:      types.StringValue(vpc.ProvisionedID),
		CIDR:               types.StringValue(vpc.CIDR),
		RegionCode:         types.StringValue(vpc.RegionCode),
		Status:             types.StringValue(vpc.Status),
		ErrorMessage:       types.StringValue(vpc.ErrorMessage),
		Created:            types.StringValue(vpc.Created),
		Updated:            types.StringValue(vpc.Updated),
		PeeringConnections: types.ListUnknown(PeeringConnectionsType),
	}

	// for _, peeringConn := range vpc.PeeringConnections {
	// 	peeringConnID, err := strconv.ParseInt(peeringConn.ID, 10, 64)
	// 	if err != nil {
	// 		resp.Diagnostics.AddError("Unable to Convert Vpc ID", err.Error())
	// 		return
	// 	}
	// 	peeringConnVpcID, err := strconv.ParseInt(peeringConn.VpcID, 10, 64)
	// 	if err != nil {
	// 		resp.Diagnostics.AddError("Unable to Convert Vpc ID", err.Error())
	// 		return
	// 	}
	// 	peerConn := peeringConnectionModel{
	// 		ID:           types.Int64Value(peeringConnID),
	// 		VpcID:        types.Int64Value(peeringConnVpcID),
	// 		Status:       types.StringValue(peeringConn.Status),
	// 		ErrorMessage: types.StringValue(peeringConn.ErrorMessage),
	// 	}
	// 	for _, peerVpc := range peeringConn.PeerVpcs {
	// 		peerVpcId, err := strconv.ParseInt(peerVpc.ID, 10, 64)
	// 		if err != nil {
	// 			resp.Diagnostics.AddError("Unable to Convert Vpc ID", err.Error())
	// 			return
	// 		}
	// 		peerConn.PeerVpcs = append(peerConn.PeerVpcs, &peerVpcModel{
	// 			ID:         types.Int64Value(peerVpcId),
	// 			AccountID:  types.StringValue(peerVpc.AccountID),
	// 			CIDR:       types.StringValue(peerVpc.CIDR),
	// 			RegionCode: types.StringValue(peerVpc.RegionCode),
	// 		})
	// 	}
	// 	vpcState.PeeringConnections = append(vpcState.PeeringConnections, peerConn)
	// }

	// Set state
	diags := resp.State.Set(ctx, &vpcState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes a VPC shell
func (r *vpcsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "VpcsResource.Delete")
	var state vpcResourceModel
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Deleting Vpc: %v", state.ID.ValueInt64()))

	_, err := r.client.DeleteVPC(ctx, state.ID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Timescale Vpc",
			"Could not delete vpc, unexpected error: "+err.Error(),
		)
		return
	}
}

// Update updates a VPC shell
func (r *vpcsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "VpcsResource.Update")
	var plan, state vpcResourceModel
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.Name.Equal(state.Name) {
		if err := r.client.RenameVpc(ctx, state.ID.ValueInt64(), plan.Name.ValueString()); err != nil {
			resp.Diagnostics.AddError("Failed to rename a vpc", err.Error())
			return
		}
	}
}

func (r *vpcsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Configure adds the provider configured client to the data source.
func (r *vpcsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	tflog.Trace(ctx, "vpcsResource.Configure")
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

// Schema defines the schema for the data source.
func (r *vpcsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed: true,
			},
			"provisioned_id": schema.StringAttribute{
				Computed: true,
			},
			"project_id": schema.StringAttribute{
				Computed: true,
			},
			"cidr": schema.StringAttribute{
				Description:         `The IPv4 CIDR block`,
				MarkdownDescription: "The IPv4 CIDR block",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "VPC Name is the configurable name assigned to this vpc. If none is provided, a default will be generated by the provider.",
				Description:         "Vpc name",
				Optional:            true,
				// If the name attribute is absent, the provider will generate a default.
				Computed: true,
			},
			"region_code": schema.StringAttribute{
				Description:         `The region for this service`,
				MarkdownDescription: "The region for this service. Currently supported regions are us-east-1, eu-west-1, us-west-2, eu-central-1, ap-southeast-2",
				Required:            true,
				Validators:          []validator.String{stringvalidator.OneOf(regionCodes...)},
			},
			"status": schema.StringAttribute{
				Computed: true,
			},
			"error_message": schema.StringAttribute{
				Computed: true,
			},
			"created": schema.StringAttribute{
				Computed: true,
			},
			"updated": schema.StringAttribute{
				Computed: true,
			},
			"peering_connections": schema.ListNestedAttribute{
				Optional: true,
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Computed: true,
						},
						"vpc_id": schema.Int64Attribute{
							Computed: true,
						},
						"status": schema.StringAttribute{
							Computed: true,
						},
						"error_message": schema.StringAttribute{
							Computed: true,
						},
						"peer_vpc": schema.ListNestedAttribute{
							Computed: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"id": schema.Int64Attribute{
										Computed: true,
									},
									"cidr": schema.StringAttribute{
										Computed: true,
									},
									"region_code": schema.StringAttribute{
										Computed: true,
									},
									"account_id": schema.StringAttribute{
										Computed: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
