package provider_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPrivateLinkRegionDataSource_basic(t *testing.T) {
	server := NewMockServer(t)
	defer server.Close()

	server.Handle("ListPrivateLinkAvailableRegions", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{
			"data": map[string]interface{}{
				"listPrivateLinkAvailableRegions": []map[string]interface{}{
					{"region": "az-eastus", "cloudProvider": "azure", "serviceName": "alias-eastus.guid.azure"},
					{"region": "us-east-1", "cloudProvider": "aws", "serviceName": "com.amazonaws.vpce.us-east-1.vpce-svc-example"},
				},
			},
		}
	})

	server.SetupEnv(t)

	config := ProviderConfig + `
data "timescale_privatelink_region" "aws_region" {
  region = "us-east-1"
}

data "timescale_privatelink_region" "azure_region" {
  region = "az-eastus"
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: TestProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.timescale_privatelink_region.aws_region", "cloud_provider", "aws"),
					resource.TestCheckResourceAttr("data.timescale_privatelink_region.aws_region", "service_name", "com.amazonaws.vpce.us-east-1.vpce-svc-example"),
					resource.TestCheckResourceAttr("data.timescale_privatelink_region.azure_region", "cloud_provider", "azure"),
					resource.TestCheckResourceAttr("data.timescale_privatelink_region.azure_region", "service_name", "alias-eastus.guid.azure"),
				),
			},
		},
	})
}

func TestAccPrivateLinkRegionDataSource_invalidRegion(t *testing.T) {
	server := NewMockServer(t)
	defer server.Close()

	server.Handle("ListPrivateLinkAvailableRegions", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{
			"data": map[string]interface{}{
				"listPrivateLinkAvailableRegions": []map[string]interface{}{
					{"region": "az-eastus", "cloudProvider": "azure", "serviceName": "alias-eastus.guid.azure"},
					{"region": "us-east-1", "cloudProvider": "aws", "serviceName": "com.amazonaws.vpce.us-east-1.vpce-svc-example"},
				},
			},
		}
	})

	server.SetupEnv(t)

	config := ProviderConfig + `
data "timescale_privatelink_region" "selected" {
  region = "us-east-3"
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: TestProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`(?s)Region "us-east-3" is not available.*Available regions.*aws: us-east-1.*azure: az-eastus`),
			},
		},
	})
}
