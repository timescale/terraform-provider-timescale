package provider

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestGeneralScenario(t *testing.T) {
	const (
		primaryName = "primary"
		extraName   = "extra"
		replicaName = "read_replica"
		primaryFQID = "timescale_service." + primaryName
		extraFQID   = "timescale_service." + extraName
		replicaFQID = "timescale_service." + replicaName
	)
	var (
		primaryConfig = &ServiceConfig{
			ResourceName: primaryName,
			Name:         "service resource test init",
		}
		replicaConfig = &ServiceConfig{
			ResourceName:      replicaName,
			ReadReplicaSource: primaryFQID + ".id",
			MilliCPU:          500,
			MemoryGB:          2,
		}
		config = &ServiceConfig{
			ResourceName: "resource",
		}
		vpcConfig = &VPCConfig{
			ResourceName: "resource",
		}
	)
	var vpcID int64
	var vpcIDStr string
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Create VPC
			{
				Config: getVPCConfig(t, vpcConfig.WithName("import-test").WithCIDR("10.0.0.0/21").WithRegionCode("us-east-1")),
				Check: func(s *terraform.State) error {
					time.Sleep(10 * time.Second)
					rs, ok := s.RootModule().Resources["timescale_vpcs.resource"]
					if !ok {
						return fmt.Errorf("Not found: %s", "timescale_vpcs.resource")
					}

					if rs.Primary.ID == "" {
						return fmt.Errorf("Widget ID is not set")
					}
					var err error
					vpcIDStr = rs.Primary.ID
					vpcID, err = strconv.ParseInt(rs.Primary.ID, 10, 64)
					if err != nil {
						return fmt.Errorf("Could not parse ID")
					}
					return nil
				},
			},
			// Create with HA and VPC attached
			{
				Config: newServiceCustomVpcConfig("hareplica", ServiceConfig{
					Name:            "service resource test HA",
					RegionCode:      "us-east-1",
					MilliCPU:        500,
					MemoryGB:        2,
					EnableHAReplica: true,
					VpcID:           vpcID,
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_service.hareplica", "name", "service resource test HA"),
					resource.TestCheckResourceAttr("timescale_service.hareplica", "enable_ha_replica", "true"),
					resource.TestCheckResourceAttr("timescale_service.hareplica", "vpc_id", vpcIDStr),
				),
			},
			// Create default and Read testing
			{
				Config: getServiceConfig(t, config),
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

			// Add VPC
			{
				Config: getServiceConfig(t, config.WithVPC(vpcID)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_service.resource", "vpc_id", vpcIDStr),
				),
			},
			// Add HA replica and remove VPC
			{
				Config: getServiceConfig(t, config.WithVPC(0).WithHAReplica(true)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("timescale_service.resource", "vpc_id"),
					resource.TestCheckResourceAttr("timescale_service.resource", "enable_ha_replica", "true"),
				),
			},
			// Create with read replica
			{
				Config: getServiceConfig(t, primaryConfig, replicaConfig),
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
			// Add VPC to the read replica
			{
				Config: getServiceConfig(t, primaryConfig, replicaConfig.WithVPC(vpcID)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(replicaFQID, "vpc_id", vpcIDStr),
				),
			},
			// Remove VPC
			{
				Config: getServiceConfig(t, primaryConfig, replicaConfig.WithVPC(0)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr(primaryFQID, "vpc_id"),
				),
			},
		},
	})
}
