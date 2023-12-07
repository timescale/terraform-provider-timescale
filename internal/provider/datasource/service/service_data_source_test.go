package service_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/timescale/terraform-provider-timescale/internal/test"
)

func TestServiceDataSource(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: test.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: newServiceDataSource(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.timescale_service.data_source", "id"),
					resource.TestCheckResourceAttrSet("data.timescale_service.data_source", "name"),
					resource.TestCheckResourceAttrSet("data.timescale_service.data_source", "region_code"),
					resource.TestCheckResourceAttrSet("data.timescale_service.data_source", "created"),
					resource.TestCheckResourceAttrSet("data.timescale_service.data_source", "spec.hostname"),
					resource.TestCheckResourceAttrSet("data.timescale_service.data_source", "spec.username"),
					resource.TestCheckResourceAttrSet("data.timescale_service.data_source", "spec.port"),
					resource.TestCheckResourceAttrSet("data.timescale_service.data_source", "resources.0.id"),
					resource.TestCheckResourceAttrSet("data.timescale_service.data_source", "resources.0.spec.milli_cpu"),
					resource.TestCheckResourceAttrSet("data.timescale_service.data_source", "resources.0.spec.memory_gb"),
				),
			},
		},
	})
}

func newServiceDataSource() string {
	return test.ProviderConfig + `
				resource "timescale_service" "resource" {
					name = "newServiceDataSource test"
				}
				data "timescale_service" "data_source" {
					id = timescale_service.resource.id
				}
`
}
