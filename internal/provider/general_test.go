package provider

const (
	//datadogMetricExporterName    = "datadog-metrics-ci"
	cloudwatchMetricExporterName = "cloudwatch-metrics-ci"
	cloudwatchMetricExporterID   = "cloudwatch-metrics-ci"
	cloudwatchLogExporterID      = "31377f38-5b42-4a5c-81a0-aee4c338a250"
	cloudwatchLogExporterName    = "cloudwatch-logs-ci"
)

// func TestGeneralScenario(t *testing.T) {
// 	const (
// 		primaryName = "primary"
// 		extraName   = "extra"
// 		replicaName = "read_replica"
// 		primaryFQID = "timescale_service." + primaryName
// 		extraFQID   = "timescale_service." + extraName
// 		replicaFQID = "timescale_service." + replicaName
// 	)
// 	var (
// 		primaryConfig = &ServiceConfig{
// 			ResourceName: primaryName,
// 			Name:         "service resource test init",
// 		}
// 		replicaConfig = &ServiceConfig{
// 			ResourceName:      replicaName,
// 			ReadReplicaSource: primaryFQID + ".id",
// 			MilliCPU:          500,
// 			MemoryGB:          2,
// 		}
// 		config = &ServiceConfig{
// 			ResourceName: "resource",
// 		}
// 		vpcConfig = &VPCConfig{
// 			ResourceName: "resource",
// 		}
// 	)
// 	var vpcID int64
// 	var vpcIDStr string
// 	resource.ParallelTest(t, resource.TestCase{
// 		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
// 		PreCheck:                 func() { testAccPreCheck(t) },
// 		Steps: []resource.TestStep{
// 			// Create VPC
// 			{
// 				Config: getVPCConfig(t, vpcConfig.WithName("global-test").WithCIDR("10.0.0.0/21").WithRegionCode("us-east-1")),
// 				Check: func(s *terraform.State) error {
// 					time.Sleep(10 * time.Second)
// 					rs, ok := s.RootModule().Resources["timescale_vpcs.resource"]
// 					if !ok {
// 						return fmt.Errorf("Not found: %s", "timescale_vpcs.resource")
// 					}
// 					if rs.Primary.ID == "" {
// 						return fmt.Errorf("Widget ID is not set")
// 					}
// 					var err error
// 					vpcIDStr = rs.Primary.ID
// 					vpcID, err = strconv.ParseInt(rs.Primary.ID, 10, 64)
// 					if err != nil {
// 						return fmt.Errorf("Could not parse ID")
// 					}
// 					return nil
// 				},
// 			},
// 			// Create with VPC attached
// 			{
// 				Config: newServiceCustomVpcConfig("with_vpc", ServiceConfig{
// 					Name:       "service-with-vpc",
// 					RegionCode: "us-east-1",
// 					MilliCPU:   500,
// 					MemoryGB:   2,
// 					VpcID:      vpcID,
// 				}),
// 				Check: resource.ComposeAggregateTestCheckFunc(
// 					resource.TestCheckResourceAttr("timescale_service.with_vpc", "name", "service resource test HA"),
// 					resource.TestCheckResourceAttr("timescale_service.with_vpc", "vpc_id", vpcIDStr),
// 				),
// 			},
// 			// Create default and Read testing
// 			{
// 				Config: getServiceConfig(t, config.WithName("create default")),
// 				Check: resource.ComposeAggregateTestCheckFunc(
// 					// Verify the name is set.
// 					resource.TestCheckResourceAttrSet("timescale_service.resource", "name"),
// 					// Verify ID value is set in state.
// 					resource.TestCheckResourceAttrSet("timescale_service.resource", "id"),
// 					resource.TestCheckResourceAttrSet("timescale_service.resource", "password"),
// 					resource.TestCheckResourceAttrSet("timescale_service.resource", "hostname"),
// 					resource.TestCheckResourceAttrSet("timescale_service.resource", "username"),
// 					resource.TestCheckResourceAttrSet("timescale_service.resource", "port"),
// 					resource.TestCheckResourceAttr("timescale_service.resource", "milli_cpu", "500"),
// 					resource.TestCheckResourceAttr("timescale_service.resource", "memory_gb", "2"),
// 					resource.TestCheckResourceAttr("timescale_service.resource", "region_code", "us-east-1"),
// 					resource.TestCheckResourceAttr("timescale_service.resource", "enable_ha_replica", "false"),
// 					resource.TestCheckNoResourceAttr("timescale_service.resource", "vpc_id"),
// 				),
// 			},

// 			// Add VPC
// 			{
// 				Config: getVPCConfig(t, vpcConfig.WithName("global-test").WithCIDR("10.0.0.0/21").WithRegionCode("us-east-1")) + getServiceNoProviderConfig(t, config.WithName("create default").WithVPC(vpcID)),
// 				Check: resource.ComposeAggregateTestCheckFunc(
// 					resource.TestCheckResourceAttr("timescale_service.resource", "vpc_id", vpcIDStr),
// 				),
// 			},
// 			// Add HA replica and remove VPC
// 			{
// 				Config: getServiceConfig(t, config.WithVPC(0).WithHAReplica(true)),
// 				Check: resource.ComposeAggregateTestCheckFunc(
// 					resource.TestCheckNoResourceAttr("timescale_service.resource", "vpc_id"),
// 					resource.TestCheckResourceAttr("timescale_service.resource", "enable_ha_replica", "true"),
// 				),
// 			},
// 			// Remove VPC
// 			{
// 				Config: getServiceConfig(t, primaryConfig, replicaConfig.WithVPC(0)),
// 				Check: resource.ComposeAggregateTestCheckFunc(
// 					resource.TestCheckNoResourceAttr(primaryFQID, "vpc_id"),
// 				),
// 			},
// 		},
// 	})
// }

// func newServiceCustomVpcConfig(resourceName string, config ServiceConfig) string {
// 	if config.Timeouts.Create == "" {
// 		config.Timeouts.Create = "30m"
// 	}
// 	return providerConfig + fmt.Sprintf(`
// 		resource "timescale_service" "%s" {
// 			name = %q
// 			timeouts = {
// 				create = %q
// 			}
// 			milli_cpu  = %d
// 			memory_gb  = %d
// 			region_code = %q
// 			vpc_id = %d
// 			enable_ha_replica = %t
// 		}`, resourceName, config.Name, config.Timeouts.Create, config.MilliCPU, config.MemoryGB, config.RegionCode, config.VpcID, config.EnableHAReplica)
// }
