package provider

import (
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"testing"
)

func TestVPCDataSource(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "timescale_vpcs" "test" {
  name        = "test-vpc-for-data-source"
  cidr        = "10.0.0.0/16"
  region_code = "us-east-1"
}
data "timescale_vpcs" "data_source" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.timescale_vpcs.data_source", "vpcs.0.id"),
					resource.TestCheckResourceAttrSet("data.timescale_vpcs.data_source", "vpcs.0.name"),
					resource.TestCheckResourceAttrSet("data.timescale_vpcs.data_source", "vpcs.0.region_code"),
					resource.TestCheckResourceAttrSet("data.timescale_vpcs.data_source", "vpcs.0.created"),
					resource.TestCheckResourceAttrSet("data.timescale_vpcs.data_source", "vpcs.0.cidr"),
					resource.TestCheckResourceAttrSet("data.timescale_vpcs.data_source", "vpcs.0.status"),
					resource.TestCheckResourceAttrSet("data.timescale_vpcs.data_source", "vpcs.0.project_id"),
				),
			},
		},
	})
}
