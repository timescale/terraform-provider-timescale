package provider_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
)

func TestAccPrivateLinkConnectionDataSource_byProviderConnectionID(t *testing.T) {
	server := NewMockServer(t)
	defer server.Close()

	server.Handle("ListPrivateLinkConnections", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{
			"data": map[string]interface{}{
				"listPrivateLinkConnections": []map[string]interface{}{
					{
						"connectionId":         "conn-123",
						"providerConnectionId": "my-endpoint.abc-123",
						"cloudProvider":        "azure",
						"region":               "az-eastus2",
						"linkIdentifier":       "link-789",
						"state":                "approved",
						"ipAddress":            "10.0.0.5",
						"name":                 "My Connection",
						"createdAt":            "2024-01-01T00:00:00Z",
						"updatedAt":            "2024-01-01T00:00:00Z",
					},
				},
			},
		}
	})

	server.SetupEnv(t)

	config := ProviderConfig + `
data "timescale_privatelink_connection" "test" {
  provider_connection_id = "my-endpoint"
  cloud_provider         = "azure"
  region                 = "az-eastus2"
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: TestProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.timescale_privatelink_connection.test", "connection_id", "conn-123"),
					resource.TestCheckResourceAttr("data.timescale_privatelink_connection.test", "provider_connection_id", "my-endpoint.abc-123"),
					resource.TestCheckResourceAttr("data.timescale_privatelink_connection.test", "cloud_provider", "azure"),
					resource.TestCheckResourceAttr("data.timescale_privatelink_connection.test", "region", "az-eastus2"),
					resource.TestCheckResourceAttr("data.timescale_privatelink_connection.test", "state", "approved"),
					resource.TestCheckResourceAttr("data.timescale_privatelink_connection.test", "ip_address", "10.0.0.5"),
					resource.TestCheckResourceAttr("data.timescale_privatelink_connection.test", "name", "My Connection"),
				),
			},
		},
	})
}

func TestAccPrivateLinkConnectionDataSource_bothSpecified(t *testing.T) {
	server := NewMockServer(t)
	defer server.Close()
	server.SetupEnv(t)

	config := ProviderConfig + `
data "timescale_privatelink_connection" "test" {
  connection_id          = "conn-123"
  provider_connection_id = "my-endpoint"
  cloud_provider         = "azure"
  region                 = "az-eastus2"
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: TestProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile("Conflicting attributes"),
			},
		},
	})
}

func TestAccPrivateLinkConnectionDataSource_byConnectionID(t *testing.T) {
	server := NewMockServer(t)
	defer server.Close()

	server.Handle("ListPrivateLinkAvailableRegions", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{
			"data": map[string]interface{}{
				"listPrivateLinkAvailableRegions": []map[string]interface{}{
					{"region": "az-eastus", "cloudProvider": "azure", "serviceName": "alias-eastus"},
					{"region": "az-eastus2", "cloudProvider": "azure", "serviceName": "alias-eastus2"},
				},
			},
		}
	})

	server.Handle("ListPrivateLinkConnections", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		vars := GetVars(req)
		region := GetString(vars, "region")

		if region == "az-eastus" {
			return map[string]interface{}{
				"data": map[string]interface{}{
					"listPrivateLinkConnections": []map[string]interface{}{},
				},
			}
		}

		assert.Equal(t, "az-eastus2", region)
		return map[string]interface{}{
			"data": map[string]interface{}{
				"listPrivateLinkConnections": []map[string]interface{}{
					{
						"connectionId":         "conn-456",
						"providerConnectionId": "other-endpoint.xyz-789",
						"cloudProvider":        "azure",
						"region":               "az-eastus2",
						"linkIdentifier":       "link-def",
						"state":                "approved",
						"ipAddress":            "10.0.1.10",
						"name":                 "Found Connection",
						"createdAt":            "2024-01-01T00:00:00Z",
						"updatedAt":            "2024-01-01T00:00:00Z",
					},
				},
			},
		}
	})

	server.SetupEnv(t)

	config := ProviderConfig + `
data "timescale_privatelink_connection" "test" {
  connection_id = "conn-456"
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: TestProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.timescale_privatelink_connection.test", "connection_id", "conn-456"),
					resource.TestCheckResourceAttr("data.timescale_privatelink_connection.test", "provider_connection_id", "other-endpoint.xyz-789"),
					resource.TestCheckResourceAttr("data.timescale_privatelink_connection.test", "cloud_provider", "azure"),
					resource.TestCheckResourceAttr("data.timescale_privatelink_connection.test", "region", "az-eastus2"),
					resource.TestCheckResourceAttr("data.timescale_privatelink_connection.test", "state", "approved"),
					resource.TestCheckResourceAttr("data.timescale_privatelink_connection.test", "ip_address", "10.0.1.10"),
					resource.TestCheckResourceAttr("data.timescale_privatelink_connection.test", "name", "Found Connection"),
				),
			},
		},
	})
}
