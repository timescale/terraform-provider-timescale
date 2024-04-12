package provider

import (
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

var (
	config = &VPCConfig{
		ResourceName: "resource",
	}
)

func TestVPCResource_Default_Success(t *testing.T) {
	// Test resource creation succeeds
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Create the VPC
			{
				Config: getVPCConfig(t, config.WithName("vpc-1").WithCIDR("10.0.0.0/21").WithRegionCode("us-east-1")),
				Check: resource.ComposeAggregateTestCheckFunc(
					func(s *terraform.State) error {
						time.Sleep(10 * time.Second)
						return nil
					},
					resource.TestCheckResourceAttrSet("timescale_vpcs.resource", "project_id"),
					resource.TestCheckResourceAttrSet("timescale_vpcs.resource", "cidr"),
					resource.TestCheckResourceAttrSet("timescale_vpcs.resource", "id"),
					resource.TestCheckResourceAttr("timescale_vpcs.resource", "status", "CREATED"),
					resource.TestCheckResourceAttr("timescale_vpcs.resource", "name", "vpc-1"),
					// resource.TestCheckResourceAttrSet("timescale_vpcs.resource", "updated"),
					// resource.TestCheckNoResourceAttr("timescale_vpcs.resource", "created"),
				),
			},
			// Rename
			{
				Config: getVPCConfig(t, config.WithName("vpc-renamed").WithCIDR("10.0.0.0/21").WithRegionCode("us-east-1")),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("timescale_vpcs.resource", "project_id"),
					resource.TestCheckResourceAttrSet("timescale_vpcs.resource", "cidr"),
					resource.TestCheckResourceAttrSet("timescale_vpcs.resource", "created"),
					resource.TestCheckResourceAttrSet("timescale_vpcs.resource", "id"),
					resource.TestCheckResourceAttrSet("timescale_vpcs.resource", "status"),
					resource.TestCheckResourceAttr("timescale_vpcs.resource", "name", "vpc-renamed"),
					// resource.TestCheckNoResourceAttr("timescale_vpcs.resource", "updated"), // rename returns a success and not a vpc so we only get this at refresh
				),
			},
		},
	})
}

func TestVPCResource_Import(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Create the VPC to import
			{
				Config: getVPCConfig(t, config.WithName("import-test").WithCIDR("10.0.0.0/21").WithRegionCode("us-east-1")),
				Check: func(s *terraform.State) error {
					time.Sleep(10 * time.Second)
					return nil
				},
			},
			{
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"created", "status", "provisioned_id"},
				ImportStateId:           "import-test",
				ResourceName:            "timescale_vpcs.resource_import",
				Config: getVPCConfig(t, config.WithName("import-test").WithCIDR("10.0.0.0/21").WithRegionCode("us-east-1")) + `
				resource "timescale_vpcs" "resource_import" {}
				`,
			},
		},
	})
}
