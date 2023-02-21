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
					// Verify we have the 3 expected types of product
					resource.TestCheckResourceAttr("data.timescale_products.products", "products.#", "3"),
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
