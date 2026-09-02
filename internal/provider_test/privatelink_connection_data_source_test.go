package provider_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPrivateLinkConnectionDataSource_byConnectionID(t *testing.T) {
	server := NewMockServer(t)
	defer server.Close()

	server.Handle("ListPrivateLinkConnections", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{
			"data": map[string]interface{}{
				"listPrivateLinkConnections": []map[string]interface{}{
					{
						"connectionId":         "conn-other",
						"providerConnectionId": "vpce-9999999999999999",
						"cloudProvider":        "aws",
						"region":               "us-east-1",
						"linkIdentifier":       "vpce-9999999999999999",
						"principalId":          "123456789012",
						"state":                "approved",
						"name":                 "other",
						"createdAt":            "2024-01-01T00:00:00Z",
						"updatedAt":            "2024-01-01T00:00:00Z",
					},
					{
						"connectionId":         "conn-123",
						"providerConnectionId": "my-pe.f91412e6-1111-2222-3333-444455556666",
						"cloudProvider":        "azure",
						"region":               "az-eastus2",
						"linkIdentifier":       "link-789",
						"principalId":          "sub-abc",
						"state":                "approved",
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
  connection_id = "conn-123"
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: TestProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.timescale_privatelink_connection.test", "connection_id", "conn-123"),
					resource.TestCheckResourceAttr("data.timescale_privatelink_connection.test", "cloud_provider", "azure"),
					resource.TestCheckResourceAttr("data.timescale_privatelink_connection.test", "region", "az-eastus2"),
					resource.TestCheckResourceAttr("data.timescale_privatelink_connection.test", "state", "approved"),
					resource.TestCheckResourceAttr("data.timescale_privatelink_connection.test", "name", "My Connection"),
					resource.TestCheckResourceAttr("data.timescale_privatelink_connection.test", "provider_connection_id", "my-pe.f91412e6-1111-2222-3333-444455556666"),
					resource.TestCheckResourceAttr("data.timescale_privatelink_connection.test", "principal_id", "sub-abc"),
				),
			},
		},
	})
}

// An unclaimed connection belongs to no project, so it never appears in the
// project-scoped listing and the lookup must say so rather than return empty.
func TestAccPrivateLinkConnectionDataSource_notFound(t *testing.T) {
	server := NewMockServer(t)
	defer server.Close()

	server.Handle("ListPrivateLinkConnections", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{
			"data": map[string]interface{}{
				"listPrivateLinkConnections": []map[string]interface{}{},
			},
		}
	})

	server.SetupEnv(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: TestProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: ProviderConfig + `
data "timescale_privatelink_connection" "test" {
  connection_id = "conn-unclaimed"
}
`,
				ExpectError: regexp.MustCompile(`Connection not found`),
			},
		},
	})
}
