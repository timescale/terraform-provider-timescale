package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestServiceResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: newServiceConfig("demoservice"),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify the name is set.
					resource.TestCheckResourceAttr("timescale_service.test", "name", "demoservice"),
					// Verify ID value is set in state.
					resource.TestCheckResourceAttrSet("timescale_service.test", "id"),
				),
			},
			// Update and Read testing
			{
				Config:      newServiceConfig("demoservice_update"),
				ExpectError: regexp.MustCompile(ErrUpdateService),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_service.test", "name", "demoservice"),
				),
			},
		},
	})
}

func newServiceConfig(name string) string {
	return providerConfig + fmt.Sprintf(`
				resource "timescale_service" "test" {
					name = %q
				}`, name)
}
