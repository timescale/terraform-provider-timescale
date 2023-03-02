package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestServiceResource_Default_Success(t *testing.T) {
	// Test resource creation succeeds and update is not allowed
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Create default and Read testing
			{
				Config: newServiceConfig(Config{
					Name: "service resource test init",
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify the name is set.
					resource.TestCheckResourceAttr("timescale_service.resource", "name", "service resource test init"),
					// Verify ID value is set in state.
					resource.TestCheckResourceAttrSet("timescale_service.resource", "id"),
				),
			},
			// Update service name failing
			{
				Config: newServiceConfig(Config{
					Name: "service resource test update",
				}),
				ExpectError: regexp.MustCompile(ErrUpdateService),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_service.resource", "name", "service resource test init"),
				),
			},
		},
	})
}

func TestServiceResource_CustomConf(t *testing.T) {
	// Test resource creation succeeds and update is not allowed
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Invalid conf millicpu & memory invalid ratio
			{
				Config: newServiceCustomConfig(Config{
					Name:      "service resource test conf",
					MilliCPU:  2000,
					MemoryGB:  2,
					StorageGB: 10,
				}),
				ExpectError: regexp.MustCompile(ErrInvalidAttribute),
			},
			// Invalid conf storage invalid value
			{
				Config: newServiceCustomConfig(Config{
					Name:      "service resource test conf",
					MilliCPU:  500,
					MemoryGB:  2,
					StorageGB: 11,
				}),
				ExpectError: regexp.MustCompile(ErrInvalidAttribute),
			},
			// Create with custom conf
			{
				Config: newServiceCustomConfig(Config{
					Name:      "service resource test conf",
					MilliCPU:  1000,
					MemoryGB:  4,
					StorageGB: 25,
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_service.resource", "name", "service resource test conf"),
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
				resource "timescale_service" "resource" {
					name = %q
					enable_storage_autoscaling = false
					timeouts = {
						create = %q
					}
				}`, config.Name, config.Timeouts.Create)
}

func newServiceCustomConfig(config Config) string {
	if config.Timeouts.Create == "" {
		config.Timeouts.Create = "10m"
	}
	return providerConfig + fmt.Sprintf(`
				resource "timescale_service" "resource" {
					name = %q
					enable_storage_autoscaling = false
					timeouts = {
						create = %q
					}
					milli_cpu  = %d
					memory_gb  = %d
					storage_gb = %d
				}`, config.Name, config.Timeouts.Create, config.MilliCPU, config.MemoryGB, config.StorageGB)
}
