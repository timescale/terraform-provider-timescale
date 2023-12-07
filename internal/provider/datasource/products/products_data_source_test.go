package products_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/timescale/terraform-provider-timescale/internal/test"
)

func TestProductDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: test.TestAccProtoV6ProviderFactories,
		PreCheck:                 func() { test.TestAccPreCheck(t) },
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
	return test.ProviderConfig + `
		data "timescale_products" "products" {
		}`
}
