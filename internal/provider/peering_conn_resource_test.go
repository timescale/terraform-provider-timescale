package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestPeeringConnResource_vpc_basic(t *testing.T) {
	resourceName := "timescale_peering_connection.test"
	vpcName := fmt.Sprintf("test-vpc-for-pc-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + vpcPeeringConnectionConfig(vpcName, peerAccountID, peerRegion, peerVPCID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "timescale_vpc_id"),
					resource.TestCheckResourceAttrSet(resourceName, "vpc_id"),
					resource.TestCheckResourceAttrSet(resourceName, "status"),
					resource.TestCheckResourceAttrSet(resourceName, "provisioned_id"),
					resource.TestCheckResourceAttrSet(resourceName, "accepter_provisioned_id"),
					resource.TestCheckResourceAttr(resourceName, "peer_account_id", peerAccountID),
					resource.TestCheckResourceAttr(resourceName, "peer_region_code", peerRegion),
					resource.TestCheckResourceAttr(resourceName, "peer_vpc_id", peerVPCID),
					resource.TestCheckResourceAttr(resourceName, "peering_type", "vpc"),
					resource.TestCheckNoResourceAttr(resourceName, "peer_tgw_id"),
					resource.TestCheckResourceAttr(resourceName, "peer_cidr", "deprecated"),
					resource.TestCheckResourceAttr(resourceName, "peer_cidr_blocks.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "peer_cidr_blocks.0", "10.1.0.0/16"),
					resource.TestCheckResourceAttr(resourceName, "peer_cidr_blocks.1", "10.2.0.0/16"),
				),
			},
		},
	})
}

func TestPeeringConnResource_mutuallyExclusive(t *testing.T) {
	vpcName := fmt.Sprintf("test-vpc-for-pc-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      providerConfig + vpcPeeringConnectionConfigMutuallyExclusive(vpcName),
				ExpectError: regexp.MustCompile("Only one of peer_vpc_id or peer_tgw_id can be provided"),
			},
		},
	})
}

func TestPeeringConnResource_tgw_basic(t *testing.T) {
	resourceName := "timescale_peering_connection.test"
	vpcName := fmt.Sprintf("test-vpc-for-tgw-pc-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + tgwPeeringConnectionConfig(vpcName, peerAccountID, peerRegion, peerTGWID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "timescale_vpc_id"),
					resource.TestCheckResourceAttrSet(resourceName, "vpc_id"),
					resource.TestCheckResourceAttrSet(resourceName, "status"),
					resource.TestCheckResourceAttrSet(resourceName, "provisioned_id"),
					resource.TestCheckResourceAttrSet(resourceName, "accepter_provisioned_id"),
					resource.TestCheckResourceAttr(resourceName, "peer_account_id", peerAccountID),
					resource.TestCheckResourceAttr(resourceName, "peer_region_code", peerRegion),
					resource.TestCheckResourceAttr(resourceName, "peer_tgw_id", peerTGWID),
					resource.TestCheckResourceAttr(resourceName, "peering_type", "tgw"),
					resource.TestCheckNoResourceAttr(resourceName, "peer_vpc_id"),
					resource.TestCheckResourceAttr(resourceName, "peer_cidr_blocks.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "peer_cidr_blocks.0", "12.1.0.0/16"),
					resource.TestCheckResourceAttr(resourceName, "peer_cidr_blocks.1", "12.2.0.0/16"),
				),
			},
		},
	})
}

// Two peerings on one VPC are created and destroyed in parallel, so the second change hits the busy VPC.
func TestPeeringConnResource_concurrentOnOneVPC(t *testing.T) {
	vpcName := fmt.Sprintf("test-vpc-for-concurrent-pc-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + concurrentPeeringConnectionsConfig(vpcName, peerAccountID, peerRegion, peerVPCID, peerTGWID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("timescale_peering_connection.vpc", "id"),
					resource.TestCheckResourceAttr("timescale_peering_connection.vpc", "peering_type", "vpc"),
					resource.TestCheckResourceAttrSet("timescale_peering_connection.tgw", "id"),
					resource.TestCheckResourceAttr("timescale_peering_connection.tgw", "peering_type", "tgw"),
				),
			},
		},
	})
}

func concurrentPeeringConnectionsConfig(vpcName, peerAccountId, peerRegion, peerVPCID, peerTGWID string) string {
	return fmt.Sprintf(`
resource "timescale_vpcs" "test" {
  name        = %q
  cidr        = "10.0.0.0/16"
  region_code = "us-east-1"
}

resource "timescale_peering_connection" "vpc" {
  timescale_vpc_id = timescale_vpcs.test.id
  peer_account_id  = %q
  peer_region_code = %q
  peer_vpc_id      = %q
  peer_cidr_blocks = ["10.1.0.0/16", "10.2.0.0/16"]
}

resource "timescale_peering_connection" "tgw" {
  timescale_vpc_id = timescale_vpcs.test.id
  peer_account_id  = %[2]q
  peer_region_code = %[3]q
  peer_tgw_id      = %[5]q
  peer_cidr_blocks = ["12.1.0.0/16", "12.2.0.0/16"]
}
`, vpcName, peerAccountId, peerRegion, peerVPCID, peerTGWID)
}

func vpcPeeringConnectionConfig(vpcName, peerAccountId, peerRegion, peerVPCID string) string {
	return fmt.Sprintf(`
resource "timescale_vpcs" "test" {
  name        = %q
  cidr        = "10.0.0.0/16"
  region_code = "us-east-1"
}

resource "timescale_peering_connection" "test" {
  timescale_vpc_id = timescale_vpcs.test.id
  peer_account_id  = %q
  peer_region_code = %q
  peer_vpc_id      = %q
  peer_cidr_blocks = ["10.1.0.0/16", "10.2.0.0/16"]
}
`, vpcName, peerAccountId, peerRegion, peerVPCID)
}

func vpcPeeringConnectionConfigMutuallyExclusive(vpcName string) string {
	return fmt.Sprintf(`
resource "timescale_vpcs" "test" {
  name        = %q
  cidr        = "10.0.0.0/16"
  region_code = "us-east-1"
}

resource "timescale_peering_connection" "test" {
  timescale_vpc_id = timescale_vpcs.test.id
  peer_account_id  = "000000000000"
  peer_region_code = "us-west-2"
  peer_vpc_id      = "vpc-12345678"      # Both VPC ID
  peer_tgw_id      = "tgw-12345678"      # and TGW ID provided - should error
  peer_cidr_blocks = ["10.1.0.0/16", "10.2.0.0/16"]
}
`, vpcName)
}

func tgwPeeringConnectionConfig(vpcName, peerAccountId, peerRegion, peerTGWID string) string {
	return fmt.Sprintf(`
resource "timescale_vpcs" "test" {
  name        = %q
  cidr        = "10.0.0.0/16"
  region_code = "us-east-1"
}

resource "timescale_peering_connection" "test" {
  timescale_vpc_id = timescale_vpcs.test.id
  peer_account_id  = %q
  peer_region_code = %q
  peer_tgw_id      = %q
  peer_cidr_blocks = ["12.1.0.0/16", "12.2.0.0/16"] # Required for TGW
}
`, vpcName, peerAccountId, peerRegion, peerTGWID)
}
