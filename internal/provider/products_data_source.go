package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	tsClient "github.com/timescale/terraform-provider-timescale/internal/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &productsDataSource{}
	_ datasource.DataSourceWithConfigure = &productsDataSource{}
)

// NewProductsDataSource is a helper function to simplify the provider implementation.
func NewProductsDataSource() datasource.DataSource {
	return &productsDataSource{}
}

// productsDataSource is the data source implementation.
type productsDataSource struct {
	client *tsClient.Client
}

// productsDataSourceModel maps the data source schema data.
type productsDataSourceModel struct {
	Products []productsModel `tfsdk:"products"`
	// following is a placeholder, required by terraform to run test suite
	ID types.String `tfsdk:"id"`
}

// productsModel maps products schema data.
type productsModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Plans       []*planModel `tfsdk:"plans"`
}

type planModel struct {
	ID         types.String  `tfsdk:"id"`
	ProductID  types.String  `tfsdk:"product_id"`
	Price      types.Float64 `tfsdk:"price"`
	MilliCPU   types.Int64   `tfsdk:"milli_cpu"`
	MemoryGB   types.Int64   `tfsdk:"memory_gb"`
	RegionCode types.String  `tfsdk:"region_code"`
}

// Metadata returns the data source type name.
func (d *productsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_products"
}

// Read refreshes the Terraform state with the latest data.
func (d *productsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state productsDataSourceModel

	products, err := d.client.GetProducts(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Products",
			err.Error(),
		)
		return
	}

	// Map response body to model
	for _, product := range products {
		// hide vanilla PG plans as this is a product experiment
		if strings.Contains(product.ID, "product_pg") {
			continue
		}
		productState := productsModel{
			ID:          types.StringValue(product.ID),
			Name:        types.StringValue(product.Name),
			Description: types.StringValue(product.Description),
		}

		for _, plan := range product.Plans {
			// O.25 CPU instances are not available anymore
			if plan.MilliCPU == 250 {
				continue
			}
			productState.Plans = append(productState.Plans, &planModel{
				ID:         types.StringValue(plan.ID),
				ProductID:  types.StringValue(plan.ProductID),
				RegionCode: types.StringValue(plan.RegionCode),
				Price:      types.Float64Value(plan.Price),
				MilliCPU:   types.Int64Value(plan.MilliCPU),
				MemoryGB:   types.Int64Value(plan.MemoryGB),
			})
		}
		state.Products = append(state.Products, productState)
	}
	// this is a placeholder, required by terraform to run test suite
	state.ID = types.StringValue("placeholder")
	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Configure adds the provider configured client to the data source.
func (d *productsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	d.client = req.ProviderData.(*tsClient.Client)
}

// Schema defines the schema for the data source.
func (d *productsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"products": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed: true,
						},
						"name": schema.StringAttribute{
							Computed: true,
						},
						"description": schema.StringAttribute{
							Computed: true,
						},
						"plans": schema.ListNestedAttribute{
							Computed: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"id": schema.StringAttribute{
										Computed: true,
									},
									"product_id": schema.StringAttribute{
										Computed: true,
									},
									"region_code": schema.StringAttribute{
										Computed: true,
									},
									"price": schema.Float64Attribute{
										Computed: true,
									},
									"milli_cpu": schema.Int64Attribute{
										Computed: true,
									},
									"memory_gb": schema.Int64Attribute{
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
