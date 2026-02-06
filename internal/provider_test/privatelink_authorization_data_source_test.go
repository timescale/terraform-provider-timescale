package provider_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPrivateLinkAuthorizationDataSource_basic(t *testing.T) {
	server := NewMockServer(t)
	defer server.Close()

	server.Handle("ListPrivateLinkAuthorizations", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{
			"data": map[string]interface{}{
				"listPrivateLinkAuthorizations": []map[string]interface{}{
					{
						"subscriptionId": "sub-123",
						"name":           "My Authorization",
						"createdAt":      "2024-01-01T00:00:00Z",
						"updatedAt":      nil,
					},
					{
						"subscriptionId": "sub-456",
						"name":           "Another Auth",
						"createdAt":      "2024-01-02T00:00:00Z",
						"updatedAt":      nil,
					},
				},
			},
		}
	})

	server.SetupEnv(t)

	config := ProviderConfig + `
data "timescale_privatelink_authorization" "test" {
  subscription_id = "sub-123"
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: TestProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.timescale_privatelink_authorization.test", "subscription_id", "sub-123"),
					resource.TestCheckResourceAttr("data.timescale_privatelink_authorization.test", "name", "My Authorization"),
				),
			},
		},
	})
}

func TestAccPrivateLinkAuthorizationDataSource_notFound(t *testing.T) {
	server := NewMockServer(t)
	defer server.Close()

	server.Handle("ListPrivateLinkAuthorizations", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{
			"data": map[string]interface{}{
				"listPrivateLinkAuthorizations": []map[string]interface{}{
					{
						"subscriptionId": "sub-123",
						"name":           "My Authorization",
						"createdAt":      "2024-01-01T00:00:00Z",
						"updatedAt":      nil,
					},
				},
			},
		}
	})

	server.SetupEnv(t)

	config := ProviderConfig + `
data "timescale_privatelink_authorization" "test" {
  subscription_id = "non-existent-subscription"
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: TestProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile("Authorization not found"),
			},
		},
	})
}
