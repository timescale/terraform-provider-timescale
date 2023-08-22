package provider

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestServiceResource_Default_Success(t *testing.T) {
	// Test resource creation succeeds and update is not allowed
	resource.ParallelTest(t, resource.TestCase{
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
					resource.TestCheckResourceAttrSet("timescale_service.resource", "password"),
					resource.TestCheckResourceAttrSet("timescale_service.resource", "hostname"),
					resource.TestCheckResourceAttrSet("timescale_service.resource", "username"),
					resource.TestCheckResourceAttrSet("timescale_service.resource", "port"),
					resource.TestCheckResourceAttr("timescale_service.resource", "milli_cpu", "500"),
					resource.TestCheckResourceAttr("timescale_service.resource", "storage_gb", "10"),
					resource.TestCheckResourceAttr("timescale_service.resource", "memory_gb", "2"),
					resource.TestCheckResourceAttr("timescale_service.resource", "region_code", "us-east-1"),
				),
			},
			// Update service name
			{
				Config: newServiceConfig(Config{
					Name: "service resource test update",
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_service.resource", "name", "service resource test update"),
				),
			},
			// Do a compute resize
			{
				Config: newServiceComputeResizeConfig(Config{
					Name:     "service resource test update",
					MilliCPU: 1000,
					MemoryGB: 4,
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_service.resource", "milli_cpu", "1000"),
					resource.TestCheckResourceAttr("timescale_service.resource", "memory_gb", "4"),
				),
			},
		},
	})
}

func TestServiceResource_Timeout(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: newServiceConfig(Config{
					Name: "service resource test timeout",
					Timeouts: Timeouts{
						Create: "1s",
					},
				}),
				ExpectError: regexp.MustCompile(ErrCreateTimeout),
			},
		},
	})
}

func TestServiceResource_CustomConf(t *testing.T) {
	// Test resource creation succeeds and update is not allowed
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Invalid conf millicpu & memory invalid ratio
			{
				Config: newServiceCustomConfig("invalid", Config{
					Name:      "service resource test conf",
					MilliCPU:  2000,
					MemoryGB:  2,
					StorageGB: 10,
				}),
				ExpectError: regexp.MustCompile(ErrInvalidAttribute),
			},
			// Invalid conf storage invalid value
			{
				Config: newServiceCustomConfig("invalid", Config{
					Name:      "service resource test conf",
					MilliCPU:  500,
					MemoryGB:  2,
					StorageGB: 11,
				}),
				ExpectError: regexp.MustCompile(ErrInvalidAttribute),
			},
			// Invalid conf storage invalid region
			{
				Config: newServiceCustomConfig("invalid", Config{
					RegionCode: "test-invalid-region",
				}),
				ExpectError: regexp.MustCompile(ErrInvalidAttribute),
			},
			// Create with custom conf and region
			{
				Config: newServiceCustomConfig("custom", Config{
					Name:       "service resource test conf",
					RegionCode: "eu-central-1",
					MilliCPU:   1000,
					MemoryGB:   4,
					StorageGB:  25,
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_service.custom", "name", "service resource test conf"),
					resource.TestCheckResourceAttr("timescale_service.custom", "region_code", "eu-central-1"),
				),
			},
			// Create with HA
			{
				Config: newServiceCustomConfig("hareplica", Config{
					Name:            "service resource test HA",
					RegionCode:      "us-east-1",
					MilliCPU:        500,
					MemoryGB:        2,
					StorageGB:       10,
					EnableHAReplica: true,
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_service.hareplica", "name", "service resource test HA"),
					resource.TestCheckResourceAttr("timescale_service.hareplica", "enable_ha_replica", "true"),
				),
			},
		},
	})
}

func TestServiceResource_Import(t *testing.T) {
	config := newServiceConfig(Config{Name: "import test"})
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Create the service to import
			{
				Config: config,
			},
			// Import the resource. This step compares the resource attributes for "test" defined above with the imported resource
			// "test_import" defined in the config for this step. This check is done by specifying the ImportStateVerify configuration option.
			{
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					for name, r := range resources {
						if name == "timescale_service.resource" {
							return r.Primary.ID, nil
						}
					}
					return "", errors.New("import ID not found")
				},
				ResourceName: "timescale_service.resource_import",
				Config: config + ` 
				resource "timescale_service" "resource_import" {}
				`,
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
					timeouts = {
						create = %q
					}
				}`, config.Name, config.Timeouts.Create)
}

func newServiceComputeResizeConfig(config Config) string {
	if config.Timeouts.Create == "" {
		config.Timeouts.Create = "10m"
	}
	return providerConfig + fmt.Sprintf(`
		resource "timescale_service" "resource" {
			name = %q
			milli_cpu  = %d
			memory_gb  = %d
			timeouts = {
				create = %q
			}
		}`, config.Name, config.MilliCPU, config.MemoryGB, config.Timeouts.Create)
}

func newServiceCustomConfig(resourceName string, config Config) string {
	if config.Timeouts.Create == "" {
		config.Timeouts.Create = "30m"
	}
	return providerConfig + fmt.Sprintf(`
		resource "timescale_service" "%s" {
			name = %q
			timeouts = {
				create = %q
			}
			milli_cpu  = %d
			memory_gb  = %d
			storage_gb = %d
			region_code = %q
			enable_ha_replica = %t
		}`, resourceName, config.Name, config.Timeouts.Create, config.MilliCPU, config.MemoryGB, config.StorageGB, config.RegionCode, config.EnableHAReplica)
}
