package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestServiceResource_Success(t *testing.T) {
	// Test resource creation succeeds and update is not allowed
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: newServiceConfig(Config{
					Name: "demoservice",
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify the name is set.
					resource.TestCheckResourceAttr("timescale_service.test", "name", "demoservice"),
					// Verify ID value is set in state.
					resource.TestCheckResourceAttrSet("timescale_service.test", "id"),
				),
			},
			// Update and Read testing
			{
				Config: newServiceConfig(Config{
					Name: "demoservice_update",
				}),
				ExpectError: regexp.MustCompile(ErrUpdateService),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_service.test", "name", "demoservice"),
				),
			},
		},
	})
}

func newServiceConfig(config Config) string {
	if config.Timeouts.Create == "" {
		config.Timeouts.Create = "10m"
	}
	return providerConfig + fmt.Sprintf(`
				resource "timescale_service" "test" {
					name = %q
					enable_storage_autoscaling = false
					timeouts = {
						create = %q
					}
				}`, config.Name, config.Timeouts.Create)
}
