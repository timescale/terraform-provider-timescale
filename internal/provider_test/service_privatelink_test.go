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
	attachedConnectionIDs := map[string]bool{}

	server.Handle("CreateService", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		vars := GetVars(req)
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

		ids := []string{}
		for id := range attachedConnectionIDs {
			ids = append(ids, id)
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
			"privateLinkConnectionIds": ids,
		}

		return map[string]interface{}{
			"data": map[string]interface{}{
				"getService": result,
			},
		}
	})

	server.Handle("AttachServiceToPrivateLinkConnection", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		vars := GetVars(req)
		assert.Equal(t, "svc-123", vars["serviceId"])
		connIDs := vars["connectionIds"].([]interface{})
		for _, id := range connIDs {
			attachedConnectionIDs[id.(string)] = true
		}

		return map[string]interface{}{
			"data": map[string]interface{}{
				"attachServiceToPrivateLinkConnection": "OK",
			},
		}
	})

	server.Handle("DetachServiceFromPrivateLinkConnection", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		vars := GetVars(req)
		assert.Equal(t, "svc-123", vars["serviceId"])
		connIDs := vars["connectionIds"].([]interface{})
		for _, id := range connIDs {
			delete(attachedConnectionIDs, id.(string))
		}

		return map[string]interface{}{
			"data": map[string]interface{}{
				"detachServiceFromPrivateLinkConnection": "OK",
			},
		}
	})

	server.Handle("DeleteService", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		serviceCreated = false
		attachedConnectionIDs = map[string]bool{}

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

	configWithOneConnection := ProviderConfig + `
resource "timescale_service" "test" {
  name                            = "test-service"
  milli_cpu                       = 500
  memory_gb                       = 2
  region_code                     = "az-eastus2"
  private_endpoint_connection_ids = ["conn-123"]
}
`

	configWithTwoConnections := ProviderConfig + `
resource "timescale_service" "test" {
  name                            = "test-service"
  milli_cpu                       = 500
  memory_gb                       = 2
  region_code                     = "az-eastus2"
  private_endpoint_connection_ids = ["conn-123", "conn-456"]
}
`

	configWithoutPrivateLink := ProviderConfig + `
resource "timescale_service" "test" {
  name                            = "test-service"
  milli_cpu                       = 500
  memory_gb                       = 2
  region_code                     = "az-eastus2"
  private_endpoint_connection_ids = []
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: TestProviderFactories(),
		Steps: []resource.TestStep{
			// Step 1: Plan - verify create will happen
			{
				Config:             configWithOneConnection,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			// Step 2: Apply - create service with one private link attached
			{
				Config: configWithOneConnection,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_service.test", "id", "svc-123"),
					resource.TestCheckResourceAttr("timescale_service.test", "name", "test-service"),
					resource.TestCheckResourceAttr("timescale_service.test", "private_endpoint_connection_ids.#", "1"),
					resource.TestCheckTypeSetElemAttr("timescale_service.test", "private_endpoint_connection_ids.*", "conn-123"),
				),
			},
			// Step 3: Plan - verify no drift
			{
				Config:             configWithOneConnection,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Step 4: Apply - add a second connection
			{
				Config: configWithTwoConnections,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_service.test", "private_endpoint_connection_ids.#", "2"),
					resource.TestCheckTypeSetElemAttr("timescale_service.test", "private_endpoint_connection_ids.*", "conn-123"),
					resource.TestCheckTypeSetElemAttr("timescale_service.test", "private_endpoint_connection_ids.*", "conn-456"),
				),
			},
			// Step 5: Plan - verify no drift with two connections
			{
				Config:             configWithTwoConnections,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Step 6: Apply - detach all private links
			{
				Config: configWithoutPrivateLink,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_service.test", "id", "svc-123"),
					resource.TestCheckResourceAttr("timescale_service.test", "private_endpoint_connection_ids.#", "0"),
				),
			},
			// Step 7: Plan - verify no drift after detach
			{
				Config:             configWithoutPrivateLink,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Step 8: Apply - re-attach private link
			{
				Config: configWithOneConnection,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_service.test", "private_endpoint_connection_ids.#", "1"),
					resource.TestCheckTypeSetElemAttr("timescale_service.test", "private_endpoint_connection_ids.*", "conn-123"),
				),
			},
		},
	})
}
