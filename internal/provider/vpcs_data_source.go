package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	tsClient "github.com/timescale/terraform-provider-timescale/internal/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &vpcsDataSource{}
	_ datasource.DataSourceWithConfigure = &vpcsDataSource{}
)

// NewVpcsDataSource is a helper function to simplify the provider implementation.
func NewVpcsDataSource() datasource.DataSource {
	return &vpcsDataSource{}
}

// vpcsDataSource is the data source implementation.
type vpcsDataSource struct {
	client *tsClient.Client
}

// vpcsDataSourceModel maps the data source schema data.
type vpcsDataSourceModel struct {
	Vpcs []vpcsModel `tfsdk:"vpcs"`
	// following is a placeholder, required by terraform to run test suite
	ID types.String `tfsdk:"id"`
}

// vpcsModel maps vpcs schema data.
type vpcsModel struct {
	ID                 types.Int64               `tfsdk:"id"`
	ProvisionedID      types.String              `tfsdk:"provisioned_id"`
	ProjectID          types.String              `tfsdk:"project_id"`
	CIDR               types.String              `tfsdk:"cidr"`
	Name               types.String              `tfsdk:"name"`
	RegionCode         types.String              `tfsdk:"region_code"`
	Status             types.String              `tfsdk:"status"`
	ErrorMessage       types.String              `tfsdk:"error_message"`
	Created            types.String              `tfsdk:"created"`
	Updated            types.String              `tfsdk:"updated"`
	PeeringConnections []*peeringConnectionModel `tfsdk:"peering_connections"`
}

type peeringConnectionModel struct {
	ID           types.Int64     `tfsdk:"id"`
	VpcID        types.Int64     `tfsdk:"vpc_id"`
	Status       types.String    `tfsdk:"status"`
	ErrorMessage types.String    `tfsdk:"error_message"`
	PeerVpcs     []*peerVpcModel `tfsdk:"peer_vpc"`
}

type peerVpcModel struct {
	ID         types.Int64  `tfsdk:"id"`
	CIDR       types.String `tfsdk:"cidr"`
	AccountID  types.String `tfsdk:"account_id"`
	RegionCode types.String `tfsdk:"region_code"`
}

// Metadata returns the data source type name.
func (d *vpcsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpcs"
}

// Read refreshes the Terraform state with the latest data.
func (d *vpcsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state vpcsDataSourceModel

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
			ProjectID:     types.StringValue(vpc.ProjectID),
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
			peerConn := &peeringConnectionModel{
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

// Configure adds the provider configured client to the data source.
func (d *vpcsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	d.client = req.ProviderData.(*tsClient.Client)
}

// Schema defines the schema for the data source.
func (d *vpcsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"vpcs": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
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
							Computed: true,
						},
						"name": schema.StringAttribute{
							Computed: true,
						},
						"region_code": schema.StringAttribute{
							Computed: true,
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
				},
			},
		},
	}
}
