package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestPeeringConnResource_basic(t *testing.T) {
	resourceName := "timescale_peering_connection.test"
	vpcName := fmt.Sprintf("test-vpc-for-pc-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + vpcPeeringConnectionConfig(vpcName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "timescale_vpc_id"),
					resource.TestCheckResourceAttrSet(resourceName, "vpc_id"),
					resource.TestCheckResourceAttrSet(resourceName, "status"),
					resource.TestCheckResourceAttrSet(resourceName, "provisioned_id"),
					resource.TestCheckResourceAttr(resourceName, "peer_account_id", "000000000000"),
					resource.TestCheckResourceAttr(resourceName, "peer_region_code", "us-west-2"),
					resource.TestCheckResourceAttr(resourceName, "peer_vpc_id", "vpc-12345678"),
					resource.TestCheckResourceAttr(resourceName, "peer_cidr", "deprecated"),
					resource.TestCheckResourceAttr(resourceName, "peer_cidr_blocks.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "peer_cidr_blocks.0", "10.1.0.0/16"),
					resource.TestCheckResourceAttr(resourceName, "peer_cidr_blocks.1", "10.2.0.0/16"),
				),
			},
		},
	})
}

func vpcPeeringConnectionConfig(vpcName string) string {
	return fmt.Sprintf(`
resource "timescale_vpcs" "test" {
  name      = %q
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
`, vpcName)
}
