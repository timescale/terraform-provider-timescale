package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestProductDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Read datasource
			{
				Config: providerConfig + `data "timescale_products" "test" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify the products id is set.
					resource.TestCheckResourceAttr("data.timescale_products.test", "id", "placeholder"),
				),
			},
		},
	})
}
