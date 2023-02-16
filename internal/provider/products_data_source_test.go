package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestProductDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Read datasource
			{
				Config: newProductsConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify the products id is set.
					resource.TestCheckResourceAttr("data.timescale_products.products", "id", "placeholder"),
				),
			},
		},
	})
}

func newProductsConfig() string {
	return providerConfig + `
		data "timescale_products" "products" {
		}`
}
