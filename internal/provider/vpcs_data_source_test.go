package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"testing"
)

func TestVPCDataSource(t *testing.T) {
	vpcName := fmt.Sprintf("test-vpc-for-data-source-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccVPCDataSourceConfigWithPeering(vpcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.timescale_vpcs.data_source", "id"),
					resource.TestCheckResourceAttrSet("data.timescale_vpcs.data_source", "vpcs.0.id"),
					resource.TestCheckResourceAttrSet("data.timescale_vpcs.data_source", "vpcs.0.provisioned_id"),
					resource.TestCheckResourceAttrSet("data.timescale_vpcs.data_source", "vpcs.0.project_id"),
					resource.TestCheckResourceAttrSet("data.timescale_vpcs.data_source", "vpcs.0.cidr"),
					resource.TestCheckResourceAttrSet("data.timescale_vpcs.data_source", "vpcs.0.name"),
					resource.TestCheckResourceAttrSet("data.timescale_vpcs.data_source", "vpcs.0.region_code"),
					resource.TestCheckResourceAttrSet("data.timescale_vpcs.data_source", "vpcs.0.status"),
					resource.TestCheckResourceAttrSet("data.timescale_vpcs.data_source", "vpcs.0.created"),

					// Check all peering connection fields
					resource.TestCheckResourceAttrSet("data.timescale_vpcs.data_source", "vpcs.0.peering_connections.0.id"),
					resource.TestCheckResourceAttrSet("data.timescale_vpcs.data_source", "vpcs.0.peering_connections.0.vpc_id"),
					resource.TestCheckResourceAttrSet("data.timescale_vpcs.data_source", "vpcs.0.peering_connections.0.provisioned_id"),
					resource.TestCheckResourceAttrSet("data.timescale_vpcs.data_source", "vpcs.0.peering_connections.0.status"),
					resource.TestCheckResourceAttr("data.timescale_vpcs.data_source", "vpcs.0.peering_connections.0.peer_account_id", "000000000000"),
					resource.TestCheckResourceAttr("data.timescale_vpcs.data_source", "vpcs.0.peering_connections.0.peer_region_code", "us-west-2"),
					resource.TestCheckResourceAttr("data.timescale_vpcs.data_source", "vpcs.0.peering_connections.0.peer_vpc_id", "vpc-12345678"),
					resource.TestCheckResourceAttr("data.timescale_vpcs.data_source", "vpcs.0.peering_connections.0.peer_cidr_blocks.#", "2"),
					resource.TestCheckResourceAttr("data.timescale_vpcs.data_source", "vpcs.0.peering_connections.0.peer_cidr_blocks.0", "10.1.0.0/16"),
					resource.TestCheckResourceAttr("data.timescale_vpcs.data_source", "vpcs.0.peering_connections.0.peer_cidr_blocks.1", "10.2.0.0/16"),
				),
			},
		},
	})
}

func testAccVPCDataSourceConfigWithPeering(vpcName string) string {
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
  peer_vpc_id      = "vpc-12345678"
  peer_cidr_blocks = ["10.1.0.0/16", "10.2.0.0/16"]
}

data "timescale_vpcs" "data_source" {
  depends_on = [
    timescale_vpcs.test,
    timescale_peering_connection.test
  ]
}
`, vpcName)
}
