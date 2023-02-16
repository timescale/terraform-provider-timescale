package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestServiceDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: newServiceDataSource(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.timescale_service.test", "id"),
					resource.TestCheckResourceAttrSet("data.timescale_service.test", "name"),
					resource.TestCheckResourceAttrSet("data.timescale_service.test", "region_code"),
					resource.TestCheckResourceAttrSet("data.timescale_service.test", "created"),
					resource.TestCheckResourceAttrSet("data.timescale_service.test", "spec.hostname"),
					resource.TestCheckResourceAttrSet("data.timescale_service.test", "spec.username"),
					resource.TestCheckResourceAttrSet("data.timescale_service.test", "spec.port"),
					resource.TestCheckResourceAttrSet("data.timescale_service.test", "resources.0.id"),
					resource.TestCheckResourceAttrSet("data.timescale_service.test", "resources.0.spec.milli_cpu"),
					resource.TestCheckResourceAttrSet("data.timescale_service.test", "resources.0.spec.memory_gb"),
					resource.TestCheckResourceAttrSet("data.timescale_service.test", "resources.0.spec.storage_gb"),
				),
			},
		},
	})
}

func newServiceDataSource() string {
	return providerConfig + `
				resource "timescale_service" "test" {}
				data "timescale_service" "test" {
					id = timescale_service.test.id
				}
`
}
