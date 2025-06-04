package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"testing"
	"time"
)

func TestVPCDataSourceWithVPCPeering(t *testing.T) {
	resourceName := "data.timescale_vpcs.data_source"
	vpcName := fmt.Sprintf("test-vpc-for-data-source-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccVPCDataSourceConfigWithVPCPeering(vpcName, peerAccountID, peerRegion, peerVPCID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "vpcs.0.id"),
					resource.TestCheckResourceAttrSet(resourceName, "vpcs.0.provisioned_id"),
					resource.TestCheckResourceAttrSet(resourceName, "vpcs.0.project_id"),
					resource.TestCheckResourceAttrSet(resourceName, "vpcs.0.cidr"),
					resource.TestCheckResourceAttrSet(resourceName, "vpcs.0.region_code"),
					resource.TestCheckResourceAttrSet(resourceName, "vpcs.0.status"),
					resource.TestCheckResourceAttrSet(resourceName, "vpcs.0.created"),
					resource.TestCheckResourceAttr(resourceName, "vpcs.0.name", vpcName),

					// Check all peering connection fields
					resource.TestCheckResourceAttrSet(resourceName, "vpcs.0.peering_connections.0.id"),
					resource.TestCheckResourceAttrSet(resourceName, "vpcs.0.peering_connections.0.vpc_id"),
					resource.TestCheckResourceAttrSet(resourceName, "vpcs.0.peering_connections.0.provisioned_id"),
					resource.TestCheckResourceAttrSet(resourceName, "vpcs.0.peering_connections.0.status"),
					resource.TestCheckResourceAttr(resourceName, "vpcs.0.peering_connections.0.peer_account_id", peerAccountID),
					resource.TestCheckResourceAttr(resourceName, "vpcs.0.peering_connections.0.peer_region_code", peerRegion),
					resource.TestCheckResourceAttr(resourceName, "vpcs.0.peering_connections.0.peer_vpc_id", peerVPCID),
					resource.TestCheckResourceAttr(resourceName, "vpcs.0.peering_connections.0.peering_type", "vpc"),
					resource.TestCheckResourceAttr(resourceName, "vpcs.0.peering_connections.0.peer_cidr_blocks.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "vpcs.0.peering_connections.0.peer_cidr_blocks.0", "10.1.0.0/16"),
					resource.TestCheckResourceAttr(resourceName, "vpcs.0.peering_connections.0.peer_cidr_blocks.1", "10.2.0.0/16"),
				),
			},
		},
	})
}

func testAccVPCDataSourceConfigWithVPCPeering(vpcName, peerAccountId, peerRegion, peerVPCID string) string {
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

data "timescale_vpcs" "data_source" {
  depends_on = [
    timescale_vpcs.test,
    timescale_peering_connection.test
  ]
}
`, vpcName, peerAccountId, peerRegion, peerVPCID)
}

func TestVPCDataSourceWithTGWPeering(t *testing.T) {
	resourceName := "data.timescale_vpcs.data_source"
	vpcName := fmt.Sprintf("test-vpc-for-data-source-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					// Workaround to give the previous test's VPC to completely delete (async).
					time.Sleep(30 * time.Second)
				},
				Config: providerConfig + testAccVPCDataSourceConfigWithTGWPeering(vpcName, peerAccountID, peerRegion, peerTGWID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "vpcs.0.id"),
					resource.TestCheckResourceAttrSet(resourceName, "vpcs.0.provisioned_id"),
					resource.TestCheckResourceAttrSet(resourceName, "vpcs.0.project_id"),
					resource.TestCheckResourceAttrSet(resourceName, "vpcs.0.cidr"),
					resource.TestCheckResourceAttrSet(resourceName, "vpcs.0.region_code"),
					resource.TestCheckResourceAttrSet(resourceName, "vpcs.0.status"),
					resource.TestCheckResourceAttrSet(resourceName, "vpcs.0.created"),
					resource.TestCheckResourceAttr(resourceName, "vpcs.0.name", vpcName),

					// Check all peering connection fields
					resource.TestCheckResourceAttrSet(resourceName, "vpcs.0.peering_connections.0.id"),
					resource.TestCheckResourceAttrSet(resourceName, "vpcs.0.peering_connections.0.vpc_id"),
					resource.TestCheckResourceAttrSet(resourceName, "vpcs.0.peering_connections.0.provisioned_id"),
					resource.TestCheckResourceAttrSet(resourceName, "vpcs.0.peering_connections.0.status"),
					resource.TestCheckResourceAttr(resourceName, "vpcs.0.peering_connections.0.peer_account_id", peerAccountID),
					resource.TestCheckResourceAttr(resourceName, "vpcs.0.peering_connections.0.peer_region_code", peerRegion),
					resource.TestCheckResourceAttr(resourceName, "vpcs.0.peering_connections.0.peer_tgw_id", peerTGWID),
					resource.TestCheckResourceAttr(resourceName, "vpcs.0.peering_connections.0.peering_type", "tgw"),
					resource.TestCheckResourceAttr(resourceName, "vpcs.0.peering_connections.0.peer_cidr_blocks.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "vpcs.0.peering_connections.0.peer_cidr_blocks.0", "10.1.0.0/16"),
					resource.TestCheckResourceAttr(resourceName, "vpcs.0.peering_connections.0.peer_cidr_blocks.1", "10.2.0.0/16"),
				),
			},
		},
	})
}

func testAccVPCDataSourceConfigWithTGWPeering(vpcName, peerAccountId, peerRegion, peerTGWID string) string {
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
  peer_cidr_blocks = ["10.1.0.0/16", "10.2.0.0/16"]
}

data "timescale_vpcs" "data_source" {
  depends_on = [
    timescale_vpcs.test,
    timescale_peering_connection.test
  ]
}
	`, vpcName, peerAccountId, peerRegion, peerTGWID)
}
