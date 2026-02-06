package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
)

func TestAccServiceResource_withPrivateLink(t *testing.T) {
	server := NewMockServer(t)
	defer server.Close()

	// Track service state
	serviceCreated := false
	attachedConnectionID := ""

	server.Handle("CreateService", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		vars := req["variables"].(map[string]interface{})
		assert.Equal(t, "test-service", vars["name"])
		assert.Equal(t, "az-eastus2", vars["regionCode"])

		serviceCreated = true

		return map[string]interface{}{
			"data": map[string]interface{}{
				"createService": map[string]interface{}{
					"id":         "svc-123",
					"name":       "test-service",
					"regionCode": "az-eastus2",
					"status":     "READY",
					"created":    "2024-01-01T00:00:00Z",
					"password":   "secret-password",
					"resources": []map[string]interface{}{
						{
							"id": "res-1",
							"spec": map[string]interface{}{
								"milliCPU":  500,
								"memoryGB":  2,
								"storageGB": 10,
							},
						},
					},
					"replicaStatus": nil,
					"spec": map[string]interface{}{
						"hostname":                "test-service.tsdb.cloud.timescale.com",
						"username":                "tsdbadmin",
						"port":                    5432,
						"connectionPoolerEnabled": false,
					},
				},
			},
		}
	})

	server.Handle("GetService", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		if !serviceCreated {
			return map[string]interface{}{
				"data": map[string]interface{}{
					"getService": nil,
				},
			}
		}

		result := map[string]interface{}{
			"id":            "svc-123",
			"name":          "test-service",
			"regionCode":    "az-eastus2",
			"status":        "READY",
			"created":       "2024-01-01T00:00:00Z",
			"replicaStatus": nil,
			"resources": []map[string]interface{}{
				{
					"id": "res-1",
					"spec": map[string]interface{}{
						"milliCPU":  500,
						"memoryGB":  2,
						"storageGB": 10,
					},
				},
			},
			"spec": map[string]interface{}{
				"hostname":                "test-service.tsdb.cloud.timescale.com",
				"username":                "tsdbadmin",
				"port":                    5432,
				"connectionPoolerEnabled": false,
			},
		}

		if attachedConnectionID != "" {
			result["privateLinkEndpointConnectionId"] = attachedConnectionID
		}

		return map[string]interface{}{
			"data": map[string]interface{}{
				"getService": result,
			},
		}
	})

	server.Handle("AttachServiceToPrivateLink", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		vars := req["variables"].(map[string]interface{})
		assert.Equal(t, "svc-123", vars["serviceId"])
		connectionID := vars["privateEndpointConnectionId"].(string)

		attachedConnectionID = connectionID

		return map[string]interface{}{
			"data": map[string]interface{}{
				"attachServiceToPrivateEndpointConnection": "OK",
			},
		}
	})

	server.Handle("DetachServiceFromPrivateLink", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		vars := req["variables"].(map[string]interface{})
		assert.Equal(t, "svc-123", vars["serviceId"])

		attachedConnectionID = ""

		return map[string]interface{}{
			"data": map[string]interface{}{
				"detachServiceFromPrivateEndpointConnection": "OK",
			},
		}
	})

	server.Handle("DeleteService", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		serviceCreated = false
		attachedConnectionID = ""

		return map[string]interface{}{
			"data": map[string]interface{}{
				"deleteService": map[string]interface{}{
					"id":         "svc-123",
					"name":       "test-service",
					"regionCode": "az-eastus2",
					"status":     "DELETED",
				},
			},
		}
	})

	server.SetupEnv(t)

	configWithPrivateLink := ProviderConfig + `
resource "timescale_service" "test" {
  name                           = "test-service"
  milli_cpu                      = 500
  memory_gb                      = 2
  region_code                    = "az-eastus2"
  private_endpoint_connection_id = "conn-123"
}
`

	configWithoutPrivateLink := ProviderConfig + `
resource "timescale_service" "test" {
  name        = "test-service"
  milli_cpu   = 500
  memory_gb   = 2
  region_code = "az-eastus2"
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: TestProviderFactories(),
		Steps: []resource.TestStep{
			// Step 1: Plan - verify create will happen
			{
				Config:             configWithPrivateLink,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			// Step 2: Apply - create service with private link attached
			{
				Config: configWithPrivateLink,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_service.test", "id", "svc-123"),
					resource.TestCheckResourceAttr("timescale_service.test", "name", "test-service"),
					resource.TestCheckResourceAttr("timescale_service.test", "private_endpoint_connection_id", "conn-123"),
				),
			},
			// Step 3: Plan - verify no drift
			{
				Config:             configWithPrivateLink,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Step 4: Plan - verify detach will happen
			{
				Config:             configWithoutPrivateLink,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			// Step 5: Apply - detach private link
			{
				Config: configWithoutPrivateLink,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_service.test", "id", "svc-123"),
					resource.TestCheckNoResourceAttr("timescale_service.test", "private_endpoint_connection_id"),
				),
			},
			// Step 6: Plan - verify no drift after detach
			{
				Config:             configWithoutPrivateLink,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Step 7: Apply - re-attach private link
			{
				Config: configWithPrivateLink,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_service.test", "private_endpoint_connection_id", "conn-123"),
				),
			},
		},
	})
}
