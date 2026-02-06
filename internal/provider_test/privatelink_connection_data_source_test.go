package provider_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
)

func TestAccPrivateLinkConnectionDataSource_byAzureName(t *testing.T) {
	server := NewMockServer(t)
	defer server.Close()

	server.Handle("ListPrivateLinkConnections", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{
			"data": map[string]interface{}{
				"listPrivateLinkConnections": []map[string]interface{}{
					{
						"connectionId":        "conn-123",
						"azureConnectionName": "my-endpoint.abc-123",
						"region":              "az-eastus2",
						"subscriptionId":      "sub-456",
						"linkIdentifier":      "link-789",
						"state":               "APPROVED",
						"ipAddress":           "10.0.0.5",
						"name":                "My Connection",
						"createdAt":           "2024-01-01T00:00:00Z",
						"updatedAt":           "2024-01-01T00:00:00Z",
					},
				},
			},
		}
	})

	server.SetupEnv(t)

	config := ProviderConfig + `
data "timescale_privatelink_connection" "test" {
  azure_connection_name = "my-endpoint"
  region                = "az-eastus2"
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: TestProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.timescale_privatelink_connection.test", "connection_id", "conn-123"),
					resource.TestCheckResourceAttr("data.timescale_privatelink_connection.test", "azure_connection_name", "my-endpoint.abc-123"),
					resource.TestCheckResourceAttr("data.timescale_privatelink_connection.test", "region", "az-eastus2"),
					resource.TestCheckResourceAttr("data.timescale_privatelink_connection.test", "subscription_id", "sub-456"),
					resource.TestCheckResourceAttr("data.timescale_privatelink_connection.test", "state", "APPROVED"),
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
  connection_id         = "conn-123"
  azure_connection_name = "my-endpoint"
  region                = "az-eastus2"
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

	// Returns available regions - the data source will iterate through these
	server.Handle("ListPrivateLinkAvailableRegions", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{
			"data": map[string]interface{}{
				"listPrivateLinkAvailableRegions": []map[string]interface{}{
					{"region": "az-eastus", "privateLinkServiceAlias": "alias-eastus"},
					{"region": "az-eastus2", "privateLinkServiceAlias": "alias-eastus2"},
				},
			},
		}
	})

	// Returns connections for each region - connection is in second region
	server.Handle("ListPrivateLinkConnections", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		vars := req["variables"].(map[string]interface{})
		region := vars["region"].(string)

		if region == "az-eastus" {
			// First region: no matching connection
			return map[string]interface{}{
				"data": map[string]interface{}{
					"listPrivateLinkConnections": []map[string]interface{}{},
				},
			}
		}

		// Second region (az-eastus2): has the connection
		assert.Equal(t, "az-eastus2", region)
		return map[string]interface{}{
			"data": map[string]interface{}{
				"listPrivateLinkConnections": []map[string]interface{}{
					{
						"connectionId":        "conn-456",
						"azureConnectionName": "other-endpoint.xyz-789",
						"region":              "az-eastus2",
						"subscriptionId":      "sub-abc",
						"linkIdentifier":      "link-def",
						"state":               "APPROVED",
						"ipAddress":           "10.0.1.10",
						"name":                "Found Connection",
						"createdAt":           "2024-01-01T00:00:00Z",
						"updatedAt":           "2024-01-01T00:00:00Z",
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
					resource.TestCheckResourceAttr("data.timescale_privatelink_connection.test", "azure_connection_name", "other-endpoint.xyz-789"),
					resource.TestCheckResourceAttr("data.timescale_privatelink_connection.test", "region", "az-eastus2"),
					resource.TestCheckResourceAttr("data.timescale_privatelink_connection.test", "subscription_id", "sub-abc"),
					resource.TestCheckResourceAttr("data.timescale_privatelink_connection.test", "state", "APPROVED"),
					resource.TestCheckResourceAttr("data.timescale_privatelink_connection.test", "ip_address", "10.0.1.10"),
					resource.TestCheckResourceAttr("data.timescale_privatelink_connection.test", "name", "Found Connection"),
				),
			},
		},
	})
}
