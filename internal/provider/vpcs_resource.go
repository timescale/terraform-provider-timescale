package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
	Vpcs []vpcsModel `tfsdk:"vpcs"`
	// following is a placeholder, required by terraform to run test suite
	ID types.String `tfsdk:"id"`
}

// Metadata returns the data source type name.
func (d *vpcsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpc"
}

// Read refreshes the Terraform state with the latest data.
func (d *vpcsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "VpcResource.Read")
	var state vpcsResourceModel

	vpcs, err := d.client.GetVPCs(ctx)
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
		vpcState := vpcsModel{
			ID:            types.Int64Value(vpcId),
			Name:          types.StringValue(vpc.Name),
			ProvisionedID: types.StringValue(vpc.ProvisionedID),
			CIDR:          types.StringValue(vpc.CIDR),
			RegionCode:    types.StringValue(vpc.RegionCode),
			Status:        types.StringValue(vpc.Status),
			ErrorMessage:  types.StringValue(vpc.ErrorMessage),
			Created:       types.StringValue(vpc.Created),
			Updated:       types.StringValue(vpc.Updated),
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
			peerConn := peeringConnectionModel{
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
			// vpcState.PeeringConnections = append(vpcState.PeeringConnections, peerConn)
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
func (d *vpcsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "VpcResource.Create")
	var plan vpcsModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	vpc, err := d.client.CreateVPC(ctx, plan.Name.ValueString(), plan.CIDR.ValueString(), plan.RegionCode.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Vpc",
			err.Error(),
		)
		return
	}

	vpcId, err := strconv.ParseInt(vpc.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Convert Vpc ID", err.Error())
		return
	}
	vpcState := vpcsModel{
		ID:            types.Int64Value(vpcId),
		Name:          types.StringValue(vpc.Name),
		ProvisionedID: types.StringValue(vpc.ProvisionedID),
		CIDR:          types.StringValue(vpc.CIDR),
		RegionCode:    types.StringValue(vpc.RegionCode),
		Status:        types.StringValue(vpc.Status),
		ErrorMessage:  types.StringValue(vpc.ErrorMessage),
		Created:       types.StringValue(vpc.Created),
		Updated:       types.StringValue(vpc.Updated),
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
		peerConn := peeringConnectionModel{
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
		//	vpcState.PeeringConnections = append(vpcState.PeeringConnections, peerConn)
	}

	// Set state
	diags := resp.State.Set(ctx, &vpcState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes a VPC shell
func (d *vpcsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// TODO unimplemented
}

// Update updates a VPC shell
func (d *vpcsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// TODO unimplemented
}

// Configure adds the provider configured client to the data source.
func (d *vpcsResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	d.client = req.ProviderData.(*tsClient.Client)
}

// Schema defines the schema for the data source.
func (d *vpcsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed: true,
			},
			"provisioned_id": schema.StringAttribute{
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
			// "peering_connections": schema.ListNestedAttribute{
			// 	Optional: true,
			// 	Computed: true,
			// 	NestedObject: schema.NestedAttributeObject{
			// 		Attributes: map[string]schema.Attribute{
			// 			"id": schema.Int64Attribute{
			// 				Computed: true,
			// 			},
			// 			"vpc_id": schema.Int64Attribute{
			// 				Computed: true,
			// 			},
			// 			"status": schema.StringAttribute{
			// 				Computed: true,
			// 			},
			// 			"error_message": schema.StringAttribute{
			// 				Computed: true,
			// 			},
			// 			"peer_vpc": schema.ListNestedAttribute{
			// 				Computed: true,
			// 				NestedObject: schema.NestedAttributeObject{
			// 					Attributes: map[string]schema.Attribute{
			// 						"id": schema.Int64Attribute{
			// 							Computed: true,
			// 						},
			// 						"cidr": schema.StringAttribute{
			// 							Computed: true,
			// 						},
			// 						"region_code": schema.StringAttribute{
			// 							Computed: true,
			// 						},
			// 						"account_id": schema.StringAttribute{
			// 							Computed: true,
			// 						},
			// 					},
			// 				},
			// 			},
			// 		},
			// 	},
			// },
		},
	}
}
