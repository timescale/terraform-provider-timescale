package service_test

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

func TestServiceDataSource(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: TestAccProtoV6ProviderFactories,
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
