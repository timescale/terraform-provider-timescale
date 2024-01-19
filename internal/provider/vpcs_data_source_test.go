package provider

import (
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestVPCDataSource(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: getVPCConfig(t, config.WithName("data-source-test").WithCIDR("10.0.0.0/21").WithRegionCode("us-east-1")),
				Check: func(s *terraform.State) error {
					time.Sleep(5 * time.Second)
					return nil
				},
			},
			{
				Config: getVPCConfig(t, config.WithName("data-source-test").WithCIDR("10.0.0.0/21").WithRegionCode("us-east-1")) + `
				data "timescale_vpcs" "data_source" {}`,
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
