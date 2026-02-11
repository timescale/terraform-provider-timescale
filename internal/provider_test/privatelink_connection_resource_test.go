package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
)

func TestAccPrivateLinkConnectionResource_basic(t *testing.T) {
	server := NewMockServer(t)
	defer server.Close()

	// Track state for the connection
	connectionCreated := false

	server.Handle("SyncPrivateLinkConnections", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{
			"data": map[string]interface{}{
				"syncPrivateLinkConnections": "OK",
			},
		}
	})

	server.Handle("ListPrivateLinkConnections", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		vars := GetVars(req)
		assert.Equal(t, "az-eastus2", vars["region"])

		ipAddress := ""
		name := ""
		if connectionCreated {
			ipAddress = "10.0.0.5"
			name = "My Connection"
		}

		return map[string]interface{}{
			"data": map[string]interface{}{
				"listPrivateLinkConnections": []map[string]interface{}{
					{
						"connectionId":         "conn-123",
						"providerConnectionId": "my-pe.abc-guid-123",
						"cloudProvider":         "AZURE",
						"region":               "az-eastus2",
						"linkIdentifier":       "link-789",
						"state":                "APPROVED",
						"ipAddress":            ipAddress,
						"name":                 name,
						"createdAt":            "2024-01-01T00:00:00Z",
						"updatedAt":            "2024-01-01T00:00:00Z",
					},
				},
			},
		}
	})

	server.Handle("UpdatePrivateLinkConnection", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		vars := GetVars(req)
		assert.Equal(t, "conn-123", vars["connectionId"])
		assert.Equal(t, "10.0.0.5", vars["ipAddress"])
		assert.Equal(t, "My Connection", vars["name"])

		connectionCreated = true

		return map[string]interface{}{
			"data": map[string]interface{}{
				"updatePrivateLinkConnection": map[string]interface{}{
					"connectionId":         "conn-123",
					"providerConnectionId": "my-pe.abc-guid-123",
					"cloudProvider":         "AZURE",
					"region":               "az-eastus2",
					"linkIdentifier":       "link-789",
					"state":                "APPROVED",
					"ipAddress":            "10.0.0.5",
					"name":                 "My Connection",
					"createdAt":            "2024-01-01T00:00:00Z",
					"updatedAt":            "2024-01-01T00:00:00Z",
				},
			},
		}
	})

	server.Handle("DeletePrivateLinkConnection", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		vars := GetVars(req)
		assert.Equal(t, "conn-123", vars["connectionId"])

		return map[string]interface{}{
			"data": map[string]interface{}{
				"deletePrivateLinkConnection": "OK",
			},
		}
	})

	server.SetupEnv(t)

	config := ProviderConfig + `
resource "timescale_privatelink_connection" "test" {
  provider_connection_id = "my-pe"
  cloud_provider         = "AZURE"
  region                 = "az-eastus2"
  ip_address             = "10.0.0.5"
  name                   = "My Connection"
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: TestProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_privatelink_connection.test", "connection_id", "conn-123"),
					resource.TestCheckResourceAttr("timescale_privatelink_connection.test", "provider_connection_id", "my-pe"),
					resource.TestCheckResourceAttr("timescale_privatelink_connection.test", "cloud_provider", "AZURE"),
					resource.TestCheckResourceAttr("timescale_privatelink_connection.test", "region", "az-eastus2"),
					resource.TestCheckResourceAttr("timescale_privatelink_connection.test", "state", "APPROVED"),
					resource.TestCheckResourceAttr("timescale_privatelink_connection.test", "ip_address", "10.0.0.5"),
					resource.TestCheckResourceAttr("timescale_privatelink_connection.test", "name", "My Connection"),
				),
			},
			// Step 2: Import and verify state matches
			{
				ResourceName:            "timescale_privatelink_connection.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateId:           "az-eastus2,conn-123",
				ImportStateVerifyIgnore: []string{"provider_connection_id", "cloud_provider", "timeout"},
			},
		},
	})
}

func TestAccPrivateLinkConnectionResource_aws(t *testing.T) {
	server := NewMockServer(t)
	defer server.Close()

	connectionCreated := false

	server.Handle("SyncPrivateLinkConnections", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{
			"data": map[string]interface{}{
				"syncPrivateLinkConnections": "OK",
			},
		}
	})

	server.Handle("ListPrivateLinkConnections", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		vars := GetVars(req)
		assert.Equal(t, "us-east-1", vars["region"])

		ipAddress := ""
		name := ""
		if connectionCreated {
			ipAddress = "10.0.1.10"
			name = "AWS Connection"
		}

		return map[string]interface{}{
			"data": map[string]interface{}{
				"listPrivateLinkConnections": []map[string]interface{}{
					{
						"connectionId":         "conn-aws-456",
						"providerConnectionId": "vpce-0123456789abcdef0",
						"cloudProvider":         "AWS",
						"region":               "us-east-1",
						"linkIdentifier":       "link-aws-789",
						"state":                "APPROVED",
						"ipAddress":            ipAddress,
						"name":                 name,

						"createdAt":            "2024-01-01T00:00:00Z",
						"updatedAt":            "2024-01-01T00:00:00Z",
					},
				},
			},
		}
	})

	server.Handle("UpdatePrivateLinkConnection", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		vars := GetVars(req)
		assert.Equal(t, "conn-aws-456", vars["connectionId"])
		assert.Equal(t, "10.0.1.10", vars["ipAddress"])
		assert.Equal(t, "AWS Connection", vars["name"])

		connectionCreated = true

		return map[string]interface{}{
			"data": map[string]interface{}{
				"updatePrivateLinkConnection": map[string]interface{}{
					"connectionId":         "conn-aws-456",
					"providerConnectionId": "vpce-0123456789abcdef0",
					"cloudProvider":         "AWS",
					"region":               "us-east-1",
					"linkIdentifier":       "link-aws-789",
					"state":                "APPROVED",
					"ipAddress":            "10.0.1.10",
					"name":                 "AWS Connection",
					"createdAt":            "2024-01-01T00:00:00Z",
					"updatedAt":            "2024-01-01T00:00:00Z",
				},
			},
		}
	})

	server.Handle("DeletePrivateLinkConnection", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{
			"data": map[string]interface{}{
				"deletePrivateLinkConnection": "OK",
			},
		}
	})

	server.SetupEnv(t)

	config := ProviderConfig + `
resource "timescale_privatelink_connection" "test" {
  provider_connection_id = "vpce-0123456789abcdef0"
  cloud_provider         = "AWS"
  region                 = "us-east-1"
  ip_address             = "10.0.1.10"
  name                   = "AWS Connection"
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: TestProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_privatelink_connection.test", "connection_id", "conn-aws-456"),
					resource.TestCheckResourceAttr("timescale_privatelink_connection.test", "provider_connection_id", "vpce-0123456789abcdef0"),
					resource.TestCheckResourceAttr("timescale_privatelink_connection.test", "cloud_provider", "AWS"),
					resource.TestCheckResourceAttr("timescale_privatelink_connection.test", "region", "us-east-1"),
					resource.TestCheckResourceAttr("timescale_privatelink_connection.test", "state", "APPROVED"),
					resource.TestCheckResourceAttr("timescale_privatelink_connection.test", "ip_address", "10.0.1.10"),
					resource.TestCheckResourceAttr("timescale_privatelink_connection.test", "name", "AWS Connection"),
				),
			},
		},
	})
}

func TestAccPrivateLinkConnectionResource_update(t *testing.T) {
	server := NewMockServer(t)
	defer server.Close()

	// Track current state
	currentIP := ""
	currentName := ""

	server.Handle("SyncPrivateLinkConnections", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{
			"data": map[string]interface{}{
				"syncPrivateLinkConnections": "OK",
			},
		}
	})

	server.Handle("ListPrivateLinkConnections", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{
			"data": map[string]interface{}{
				"listPrivateLinkConnections": []map[string]interface{}{
					{
						"connectionId":         "conn-123",
						"providerConnectionId": "my-pe.abc-guid-123",
						"cloudProvider":         "AZURE",
						"region":               "az-eastus2",
						"linkIdentifier":       "link-789",
						"state":                "APPROVED",
						"ipAddress":            currentIP,
						"name":                 currentName,
						"createdAt":            "2024-01-01T00:00:00Z",
						"updatedAt":            "2024-01-01T00:00:00Z",
					},
				},
			},
		}
	})

	server.Handle("UpdatePrivateLinkConnection", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		vars := GetVars(req)
		currentIP = GetString(vars, "ipAddress")
		currentName = GetString(vars, "name")

		return map[string]interface{}{
			"data": map[string]interface{}{
				"updatePrivateLinkConnection": map[string]interface{}{
					"connectionId":         "conn-123",
					"providerConnectionId": "my-pe.abc-guid-123",
					"cloudProvider":         "AZURE",
					"region":               "az-eastus2",
					"linkIdentifier":       "link-789",
					"state":                "APPROVED",
					"ipAddress":            currentIP,
					"name":                 currentName,
					"createdAt":            "2024-01-01T00:00:00Z",
					"updatedAt":            "2024-01-01T00:00:00Z",
				},
			},
		}
	})

	server.Handle("DeletePrivateLinkConnection", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{
			"data": map[string]interface{}{
				"deletePrivateLinkConnection": "OK",
			},
		}
	})

	server.SetupEnv(t)

	configInitial := ProviderConfig + `
resource "timescale_privatelink_connection" "test" {
  provider_connection_id = "my-pe"
  cloud_provider         = "AZURE"
  region                 = "az-eastus2"
  ip_address             = "10.0.0.5"
  name                   = "Initial Name"
}
`

	configUpdated := ProviderConfig + `
resource "timescale_privatelink_connection" "test" {
  provider_connection_id = "my-pe"
  cloud_provider         = "AZURE"
  region                 = "az-eastus2"
  ip_address             = "10.0.0.99"
  name                   = "Updated Name"
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: TestProviderFactories(),
		Steps: []resource.TestStep{
			// Step 1: Create
			{
				Config: configInitial,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_privatelink_connection.test", "ip_address", "10.0.0.5"),
					resource.TestCheckResourceAttr("timescale_privatelink_connection.test", "name", "Initial Name"),
				),
			},
			// Step 2: Update IP and name
			{
				Config: configUpdated,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_privatelink_connection.test", "ip_address", "10.0.0.99"),
					resource.TestCheckResourceAttr("timescale_privatelink_connection.test", "name", "Updated Name"),
				),
			},
			// Step 3: Verify no drift (empty plan)
			{
				Config:             configUpdated,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}
