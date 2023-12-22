package provider

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const DEFAULT_VPC_ID = 2074 // Default vpc id for test acc

func TestServiceResource_Default_Success(t *testing.T) {
	// Test resource creation succeeds
	config := &Config{
		ResourceName: "resource",
	}
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Create default and Read testing
			{
				Config: getConfig(t, config),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify the name is set.
					resource.TestCheckResourceAttrSet("timescale_service.resource", "name"),
					// Verify ID value is set in state.
					resource.TestCheckResourceAttrSet("timescale_service.resource", "id"),
					resource.TestCheckResourceAttrSet("timescale_service.resource", "password"),
					resource.TestCheckResourceAttrSet("timescale_service.resource", "hostname"),
					resource.TestCheckResourceAttrSet("timescale_service.resource", "username"),
					resource.TestCheckResourceAttrSet("timescale_service.resource", "port"),
					resource.TestCheckResourceAttr("timescale_service.resource", "milli_cpu", "500"),
					resource.TestCheckResourceAttr("timescale_service.resource", "memory_gb", "2"),
					resource.TestCheckResourceAttr("timescale_service.resource", "region_code", "us-east-1"),
					resource.TestCheckResourceAttr("timescale_service.resource", "enable_ha_replica", "false"),
					resource.TestCheckNoResourceAttr("timescale_service.resource", "vpc_id"),
				),
			},
			// Do a compute resize
			{
				Config: getConfig(t, config.WithSpec(1000, 4)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_service.resource", "milli_cpu", "1000"),
					resource.TestCheckResourceAttr("timescale_service.resource", "memory_gb", "4"),
				),
			},
			// Update service name
			{
				Config: getConfig(t, config.WithName("service resource test update")),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_service.resource", "name", "service resource test update"),
				),
			},
			// Add VPC
			{
				Config: getConfig(t, config.WithVPC(DEFAULT_VPC_ID)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_service.resource", "vpc_id", "2074"),
				),
			},
			// Add HA replica and remove VPC
			{
				Config: getConfig(t, config.WithVPC(0).WithHAReplica(true)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("timescale_service.resource", "vpc_id"),
					resource.TestCheckResourceAttr("timescale_service.resource", "enable_ha_replica", "true"),
				),
			},
		},
	})
}

func TestServiceResource_Read_Replica(t *testing.T) {
	const (
		primaryName = "primary"
		extraName   = "extra"
		replicaName = "read_replica"
		primaryFQID = "timescale_service." + primaryName
		extraFQID   = "timescale_service." + extraName
		replicaFQID = "timescale_service." + replicaName
	)
	var (
		primaryConfig = &Config{
			ResourceName: primaryName,
			Name:         "service resource test init",
		}
		extraConfig = &Config{
			ResourceName: extraName,
		}
		replicaConfig = &Config{
			ResourceName:      replicaName,
			ReadReplicaSource: primaryFQID + ".id",
			MilliCPU:          500,
			MemoryGB:          2,
		}
		extraReplicaConfig = &Config{
			ResourceName:      replicaName + "_2",
			ReadReplicaSource: primaryFQID + ".id",
		}
	)
	// Test creating a service with a read replica
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: getConfig(t, primaryConfig, replicaConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify service attributes
					resource.TestCheckResourceAttr(primaryFQID, "name", "service resource test init"),
					resource.TestCheckResourceAttrSet(primaryFQID, "id"),
					resource.TestCheckResourceAttrSet(primaryFQID, "password"),
					resource.TestCheckResourceAttrSet(primaryFQID, "hostname"),
					resource.TestCheckResourceAttrSet(primaryFQID, "username"),
					resource.TestCheckResourceAttrSet(primaryFQID, "port"),
					resource.TestCheckResourceAttr(primaryFQID, "milli_cpu", "500"),
					resource.TestCheckResourceAttr(primaryFQID, "memory_gb", "2"),
					resource.TestCheckResourceAttr(primaryFQID, "region_code", "us-east-1"),
					resource.TestCheckResourceAttr(primaryFQID, "enable_ha_replica", "false"),
					resource.TestCheckNoResourceAttr(primaryFQID, "vpc_id"),

					// Verify read replica attributes
					resource.TestCheckResourceAttr(replicaFQID, "name", "replica-service resource test init"),
					resource.TestCheckResourceAttrSet(replicaFQID, "id"),
					resource.TestCheckResourceAttrSet(replicaFQID, "password"),
					resource.TestCheckResourceAttrSet(replicaFQID, "hostname"),
					resource.TestCheckResourceAttrSet(replicaFQID, "username"),
					resource.TestCheckResourceAttrSet(replicaFQID, "port"),
					resource.TestCheckResourceAttr(replicaFQID, "milli_cpu", "500"),
					resource.TestCheckResourceAttr(replicaFQID, "memory_gb", "2"),
					resource.TestCheckResourceAttr(replicaFQID, "region_code", "us-east-1"),
					resource.TestCheckResourceAttr(replicaFQID, "enable_ha_replica", "false"),
					resource.TestCheckResourceAttrSet(replicaFQID, "read_replica_source"),
					resource.TestCheckNoResourceAttr(replicaFQID, "vpc_id"),
				),
			},
			// Update replica name
			{
				Config: getConfig(t, primaryConfig, replicaConfig.WithName("replica")),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(replicaFQID, "name", "replica"),
				),
			},
			// Do a compute resize
			{
				Config: getConfig(t, primaryConfig, replicaConfig.WithSpec(1000, 4)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(replicaFQID, "milli_cpu", "1000"),
					resource.TestCheckResourceAttr(replicaFQID, "memory_gb", "4"),
				),
			},
			// Add VPC to the read replica
			{
				Config: getConfig(t, primaryConfig, replicaConfig.WithVPC(DEFAULT_VPC_ID)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(replicaFQID, "vpc_id", "2074"),
				),
			},
			// Remove VPC
			{
				Config: getConfig(t, primaryConfig, replicaConfig.WithVPC(0)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr(primaryFQID, "vpc_id"),
				),
			},
			// Check adding HA returns an error
			{
				Config:      getConfig(t, primaryConfig, replicaConfig.WithHAReplica(true)),
				ExpectError: regexp.MustCompile(errReplicaWithHA),
			},
			// Check removing read_replica_source returns an error
			{
				Config:      getConfig(t, primaryConfig, replicaConfig.WithHAReplica(false).WithReadReplica("")),
				ExpectError: regexp.MustCompile(errUpdateReplicaSource),
			},
			// Check changing read_replica_source returns an error
			{
				Config:      getConfig(t, primaryConfig, extraConfig, replicaConfig.WithReadReplica(extraFQID+".id")),
				ExpectError: regexp.MustCompile(errUpdateReplicaSource),
			},
			// Check enabling read_replica_source returns an error
			{
				Config:      getConfig(t, primaryConfig.WithReadReplica(extraFQID+".id"), extraConfig, replicaConfig.WithReadReplica(primaryFQID+".id")),
				ExpectError: regexp.MustCompile(errUpdateReplicaSource),
			},
			// Check creating multiple read replicas returns an error
			{
				Config:      getConfig(t, primaryConfig.WithReadReplica(""), replicaConfig, extraReplicaConfig),
				ExpectError: regexp.MustCompile(errMultipleReadReplicas),
			},
			// Test creating a read replica from a read replica returns an error
			{
				Config:      getConfig(t, primaryConfig, replicaConfig, extraReplicaConfig.WithReadReplica(replicaFQID+".id")),
				ExpectError: regexp.MustCompile(errReplicaFromFork),
			},
			// Remove Replica
			{
				Config: getConfig(t, primaryConfig),
				Check: func(state *terraform.State) error {
					resources := state.RootModule().Resources
					if _, ok := resources[replicaFQID]; ok {
						return errors.New("expected replica to be deleted")
					}
					return nil
				},
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
					Name:     "service resource test conf",
					MilliCPU: 2000,
					MemoryGB: 2,
				}),
				ExpectError: regexp.MustCompile(ErrInvalidAttribute),
			},
			// Invalid conf storage invalid value
			{
				Config: newServiceCustomConfig("invalid", Config{
					Name:     "service resource test conf",
					MilliCPU: 500,
					MemoryGB: 3,
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
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_service.custom", "name", "service resource test conf"),
					resource.TestCheckResourceAttr("timescale_service.custom", "region_code", "eu-central-1"),
					resource.TestCheckNoResourceAttr("timescale_service.custom", "vpc_id"),
				),
			},
			// Create with HA and VPC attached
			{
				Config: newServiceCustomVpcConfig("hareplica", Config{
					Name:            "service resource test HA",
					RegionCode:      "us-east-1",
					MilliCPU:        500,
					MemoryGB:        2,
					EnableHAReplica: true,
					VpcID:           DEFAULT_VPC_ID,
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_service.hareplica", "name", "service resource test HA"),
					resource.TestCheckResourceAttr("timescale_service.hareplica", "enable_ha_replica", "true"),
					resource.TestCheckResourceAttr("timescale_service.hareplica", "vpc_id", "2074"),
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
			region_code = %q
			enable_ha_replica = %t
		}`, resourceName, config.Name, config.Timeouts.Create, config.MilliCPU, config.MemoryGB, config.RegionCode, config.EnableHAReplica)
}

func newServiceCustomVpcConfig(resourceName string, config Config) string {
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
			region_code = %q
			vpc_id = %d
			enable_ha_replica = %t
		}`, resourceName, config.Name, config.Timeouts.Create, config.MilliCPU, config.MemoryGB, config.RegionCode, config.VpcID, config.EnableHAReplica)
}
