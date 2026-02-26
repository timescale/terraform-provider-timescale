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
					{"region": "az-eastus", "cloudProvider": "azure", "serviceName": "alias-eastus.guid.azure"},
					{"region": "az-eastus2", "cloudProvider": "azure", "serviceName": "alias-eastus2.guid.azure"},
					{"region": "az-westus", "cloudProvider": "azure", "serviceName": "alias-westus.guid.azure"},
					{"region": "us-east-1", "cloudProvider": "aws", "serviceName": "com.amazonaws.vpce.us-east-1.vpce-svc-example"},
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
					resource.TestCheckResourceAttr("data.timescale_privatelink_available_regions.all", "regions.az-eastus.cloud_provider", "azure"),
					resource.TestCheckResourceAttr("data.timescale_privatelink_available_regions.all", "regions.az-eastus.service_name", "alias-eastus.guid.azure"),
					resource.TestCheckResourceAttr("data.timescale_privatelink_available_regions.all", "regions.az-eastus2.service_name", "alias-eastus2.guid.azure"),
					resource.TestCheckResourceAttr("data.timescale_privatelink_available_regions.all", "regions.az-westus.service_name", "alias-westus.guid.azure"),
					resource.TestCheckResourceAttr("data.timescale_privatelink_available_regions.all", "regions.us-east-1.cloud_provider", "aws"),
					resource.TestCheckResourceAttr("data.timescale_privatelink_available_regions.all", "regions.us-east-1.service_name", "com.amazonaws.vpce.us-east-1.vpce-svc-example"),
				),
			},
		},
	})
}
