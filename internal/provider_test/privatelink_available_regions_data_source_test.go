package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPrivateLinkAvailableRegionsDataSource_basic(t *testing.T) {
	server := NewMockServer(t)
	defer server.Close()

	server.Handle("ListPrivateLinkAvailableRegions", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{
			"data": map[string]interface{}{
				"listPrivateLinkAvailableRegions": []map[string]interface{}{
					{"region": "az-eastus", "privateLinkServiceAlias": "alias-eastus.guid.azure"},
					{"region": "az-eastus2", "privateLinkServiceAlias": "alias-eastus2.guid.azure"},
					{"region": "az-westus", "privateLinkServiceAlias": "alias-westus.guid.azure"},
				},
			},
		}
	})

	server.SetupEnv(t)

	config := ProviderConfig + `
data "timescale_privatelink_available_regions" "all" {}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: TestProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.timescale_privatelink_available_regions.all", "regions.az-eastus.private_link_service_alias", "alias-eastus.guid.azure"),
					resource.TestCheckResourceAttr("data.timescale_privatelink_available_regions.all", "regions.az-eastus2.private_link_service_alias", "alias-eastus2.guid.azure"),
					resource.TestCheckResourceAttr("data.timescale_privatelink_available_regions.all", "regions.az-westus.private_link_service_alias", "alias-westus.guid.azure"),
				),
			},
		},
	})
}
