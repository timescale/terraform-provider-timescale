package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccVPCResource_basic(t *testing.T) {
	resourceName := "timescale_vpcs.test"
	vpcName := fmt.Sprintf("test-vpc-%s", acctest.RandString(8))
	vpcRenamed := fmt.Sprintf("test-vpc-renamed-%s", acctest.RandString(8))
	cidr := "10.0.0.0/16"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create the VPC
			{
				Config: providerConfig + vpcResourceConfig(vpcName, cidr),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", vpcName),
					resource.TestCheckResourceAttr(resourceName, "cidr", cidr),
					resource.TestCheckResourceAttr(resourceName, "region_code", "us-east-1"),
					resource.TestCheckResourceAttr(resourceName, "status", "CREATED"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "project_id"),
					resource.TestCheckResourceAttrSet(resourceName, "provisioned_id"),
					resource.TestCheckResourceAttrSet(resourceName, "created"),
				),
			},
			// Rename
			{
				Config: providerConfig + vpcResourceConfig(vpcRenamed, cidr),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", vpcRenamed),
					resource.TestCheckResourceAttr(resourceName, "cidr", cidr),
					resource.TestCheckResourceAttr(resourceName, "region_code", "us-east-1"),
					resource.TestCheckResourceAttr(resourceName, "status", "CREATED"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "project_id"),
					resource.TestCheckResourceAttrSet(resourceName, "provisioned_id"),
					resource.TestCheckResourceAttrSet(resourceName, "created"),
				),
			},
		},
	})
}

func TestAccVPCResource_import(t *testing.T) {
	resourceName := "timescale_vpcs.test"
	vpcName := fmt.Sprintf("test-import-%s", acctest.RandString(8))
	cidr := "11.0.0.0/16"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create the VPC to import
			{
				Config: providerConfig + vpcResourceConfig(vpcName, cidr),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", vpcName),
				),
			},
			{
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"created", "status", "provisioned_id"},
				ImportStateId:           vpcName,
				ResourceName:            "timescale_vpcs.resource_import",
				Config: providerConfig + vpcResourceConfig(vpcName, cidr) + `
				resource "timescale_vpcs" "resource_import" {}
				`,
			},
		},
	})
}

func vpcResourceConfig(name, cidr string) string {
	return fmt.Sprintf(`
resource "timescale_vpcs" "test" {
  name        = %q
  cidr        = %q
  region_code = "us-east-1"
}
`, name, cidr)
}
