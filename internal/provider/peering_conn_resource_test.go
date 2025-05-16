package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestPeeringConnResource_basic(t *testing.T) {
	resourceName := "timescale_peering_connection.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + vpcPeeringConnectionConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "timescale_vpc_id"),
					resource.TestCheckResourceAttrSet(resourceName, "vpc_id"),
					resource.TestCheckResourceAttrSet(resourceName, "status"),
					resource.TestCheckResourceAttr(resourceName, "peer_account_id", "000000000000"),
					resource.TestCheckResourceAttr(resourceName, "peer_region_code", "us-west-2"),
					resource.TestCheckResourceAttr(resourceName, "peer_vpc_id", "vpc-12345678"),
				),
			},
		},
	})
}

func vpcPeeringConnectionConfig() string {
	return `
resource "timescale_vpcs" "test" {
  name      = "test-vpc-for-pc"
  cidr        = "10.0.0.0/16"
  region_code = "us-east-1"
}

resource "timescale_peering_connection" "test" {
  timescale_vpc_id = timescale_vpcs.test.id
  peer_account_id  = "000000000000"
  peer_region_code = "us-west-2"
  peer_vpc_id      = "vpc-12345678"
}
`
}
