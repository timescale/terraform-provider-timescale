package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// testIPAllowListPrefix marks lists created by acceptance tests so the sweeper can find them.
const testIPAllowListPrefix = "tf-acc-"

func TestAccIPAllowListResource_basic(t *testing.T) {
	name := "timescale_ip_allow_list.test"
	desc := testIPAllowListPrefix + acctest.RandString(8)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + ipAllowListConfig(desc, `"203.0.113.0/24"`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(name, "id"),
					resource.TestCheckResourceAttr(name, "description", desc),
					resource.TestCheckResourceAttr(name, "cidr_blocks.#", "1"),
					resource.TestCheckResourceAttr(name, "service_ids.#", "0"),
				),
			},
			// Update CIDRs in place.
			{
				Config: providerConfig + ipAllowListConfig(desc, `"203.0.113.0/24", "198.51.100.7/32"`),
				Check:  resource.TestCheckResourceAttr(name, "cidr_blocks.#", "2"),
			},
			{
				ResourceName:      name,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func ipAllowListConfig(desc, cidrs string) string {
	return fmt.Sprintf(`
resource "timescale_ip_allow_list" "test" {
  description = %q
  cidr_blocks = [%s]
}
`, desc, cidrs)
}
