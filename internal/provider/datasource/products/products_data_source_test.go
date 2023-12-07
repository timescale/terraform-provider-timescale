package products_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/timescale/terraform-provider-timescale/internal/provider"
	"github.com/timescale/terraform-provider-timescale/internal/test"
)

// TestAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var TestAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"timescale": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func TestProductDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: TestAccProtoV6ProviderFactories,
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
