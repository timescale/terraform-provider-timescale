package provider

import (
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

var (
	pcConfig = &PeeringConnConfig{
		ResourceName:   "peering_conn",
		PeerRegionCode: "us-east-1",
		PeerVPCID:      "vpc-fake-test",
	}
)

func TestPeeringConnResource_Default_Success(t *testing.T) {
	// Test resource creation succeeds
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Create the VPC
			{
				Config: getVPCConfig(t, config.WithName("vpc-for-pc").WithCIDR("10.0.0.0/21").WithRegionCode("us-east-1")),
				Check: func(s *terraform.State) error {
					time.Sleep(10 * time.Second)
					return nil
				},
			},
			// create the peering connection
			{
				Config: getVPCConfig(t, config.WithName("vpc-for-pc").WithCIDR("10.0.0.0/21").WithRegionCode("us-east-1")) + getPeeringConnConfig(t, pcConfig.WithVpcResourceName(config.ResourceName)),
				Check: resource.ComposeAggregateTestCheckFunc(
					func(s *terraform.State) error {
						time.Sleep(10 * time.Second)
						return nil
					},
					resource.TestCheckResourceAttrSet("timescale_vpcs.resource", "project_id"),
					resource.TestCheckResourceAttrSet("timescale_vpcs.resource", "cidr"),
					resource.TestCheckResourceAttrSet("timescale_vpcs.resource", "id"),
					resource.TestCheckResourceAttrSet("timescale_vpcs.resource", "status"),
					resource.TestCheckResourceAttr("timescale_vpcs.resource", "name", "vpc-for-pc"),
					resource.TestCheckResourceAttr("timescale_peering_connection.peering_conn", "peer_region_code", "us-east-1"),
					resource.TestCheckResourceAttr("timescale_peering_connection.peering_conn", "peer_vpc_id", "vpc-fake-test"),
				),
			},
			// delete the peering conn because it must be deleted before the vpc
			{
				Config: getVPCConfig(t, config.WithName("vpc-for-pc").WithCIDR("10.0.0.0/21").WithRegionCode("us-east-1")),
				Check: resource.ComposeAggregateTestCheckFunc(
					func(s *terraform.State) error {
						time.Sleep(10 * time.Second)
						return nil
					},
					resource.TestCheckResourceAttrSet("timescale_vpcs.resource", "project_id"),
					resource.TestCheckResourceAttrSet("timescale_vpcs.resource", "cidr"),
					resource.TestCheckResourceAttrSet("timescale_vpcs.resource", "id"),
					resource.TestCheckResourceAttrSet("timescale_vpcs.resource", "status"),
					resource.TestCheckResourceAttr("timescale_vpcs.resource", "name", "vpc-for-pc"),
				),
			},
		},
	})
}
