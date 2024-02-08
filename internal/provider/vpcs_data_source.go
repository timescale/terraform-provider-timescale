package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
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

var (
	PeeringConnectionsDSType = types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"vpc_id":           types.StringType,
			"status":           types.StringType,
			"error_message":    types.StringType,
			"peer_vpc_id":      types.StringType,
			"peer_cidr":        types.StringType,
			"peer_account_id":  types.StringType,
			"peer_region_code": types.StringType,
		},
	}
)

// vpcsDataSource is the data source implementation.
type vpcsDataSource struct {
	client *tsClient.Client
}

// vpcsDataSourceModel maps the data source schema data.
type vpcsDataSourceModel struct {
	Vpcs []vpcDSModel `tfsdk:"vpcs"`
	// following is a placeholder, required by terraform to run test suite
	ID types.String `tfsdk:"id"`
}

// vpcDataSourceModel maps vpcs schema data.
type vpcDSModel struct {
	ID                 types.Int64                `tfsdk:"id"`
	ProvisionedID      types.String               `tfsdk:"provisioned_id"`
	ProjectID          types.String               `tfsdk:"project_id"`
	CIDR               types.String               `tfsdk:"cidr"`
	Name               types.String               `tfsdk:"name"`
	RegionCode         types.String               `tfsdk:"region_code"`
	Status             types.String               `tfsdk:"status"`
	ErrorMessage       types.String               `tfsdk:"error_message"`
	Created            types.String               `tfsdk:"created"`
	Updated            types.String               `tfsdk:"updated"`
	PeeringConnections []peeringConnectionDSModel `tfsdk:"peering_connections"`
}

type peeringConnectionDSModel struct {
	VpcID          types.String `tfsdk:"vpc_id"`
	Status         types.String `tfsdk:"status"`
	ErrorMessage   types.String `tfsdk:"error_message"`
	PeerVPCID      types.String `tfsdk:"peer_vpc_id"`
	PeerCIDR       types.String `tfsdk:"peer_cidr"`
	PeerAccountID  types.String `tfsdk:"peer_account_id"`
	PeerRegionCode types.String `tfsdk:"peer_region_code"`
}

// Metadata returns the data source type name.
func (d *vpcsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpcs"
}

// Read refreshes the Terraform state with the latest data.
func (d *vpcsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
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
		vpcID, err := strconv.ParseInt(vpc.ID, 10, 64)
		if err != nil {
			resp.Diagnostics.AddError("Unable to Convert Vpc ID", err.Error())
			return
		}
		vpcState := vpcDSModel{
			ID:            types.Int64Value(vpcID),
			Name:          types.StringValue(vpc.Name),
			ProvisionedID: types.StringValue(vpc.ProvisionedID),
			ProjectID:     types.StringValue(vpc.ProjectID),
			CIDR:          types.StringValue(vpc.CIDR),
			Status:        types.StringValue(vpc.Status),
			ErrorMessage:  types.StringValue(vpc.ErrorMessage),
			Updated:       types.StringValue(vpc.Updated),
			RegionCode:    types.StringValue(vpc.RegionCode),
			Created:       types.StringValue(vpc.Created),
		}

		var pcms []peeringConnectionDSModel
		for _, pc := range vpc.PeeringConnections {
			var pcm peeringConnectionDSModel
			if pc.ErrorMessage != "" {
				pcm.ErrorMessage = types.StringValue(pc.ErrorMessage)
			}
			pcm.VpcID = types.StringValue(pc.VPCID)
			pcm.Status = types.StringValue(pc.Status)
			pcm.PeerVPCID = types.StringValue(pc.PeerVPC.ID)
			pcm.PeerAccountID = types.StringValue(pc.PeerVPC.AccountID)
			pcm.PeerCIDR = types.StringValue(pc.PeerVPC.CIDR)
			pcm.PeerRegionCode = types.StringValue(pc.PeerVPC.RegionCode)
			pcms = append(pcms, pcm)
		}
		vpcState.PeeringConnections = pcms
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
						"peering_connections": schema.ListAttribute{
							Computed: true,
							ElementType: types.ObjectType{
								AttrTypes: map[string]attr.Type{
									"vpc_id":           types.StringType,
									"status":           types.StringType,
									"error_message":    types.StringType,
									"peer_vpc_id":      types.StringType,
									"peer_cidr":        types.StringType,
									"peer_account_id":  types.StringType,
									"peer_region_code": types.StringType,
								},
							},
						},
					},
				},
			},
		},
	}
}
