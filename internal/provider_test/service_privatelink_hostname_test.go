package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// Attaching or detaching a Private Link connection reallocates the port but
// keeps the hostname: the private path is reached on the same name as the
// public one, which is what lets the service certificate validate under
// sslmode=verify-full.
//
// The distinction matters beyond cosmetics. Examples derive a private DNS zone
// name from the hostname, so a hostname that goes unknown at plan time forces
// the zone -- and every record in it -- to be replaced for what is only a port
// change.
func TestAccServiceResource_privateLinkKeepsHostname(t *testing.T) {
	server := NewMockServer(t)
	defer server.Close()

	const hostname = "test-service.tsdb.cloud.timescale.com"

	// The port is allocated per binding: 5432 on the public path, and a
	// reserved private-link port once a connection is attached.
	const publicPort = 5432
	const privateLinkPort = 5433

	serviceCreated := false
	attachedConnectionIDs := map[string]bool{}

	currentPort := func() int {
		if len(attachedConnectionIDs) > 0 {
			return privateLinkPort
		}
		return publicPort
	}

	spec := func() map[string]interface{} {
		return map[string]interface{}{
			"hostname":                hostname,
			"username":                "tsdbadmin",
			"port":                    currentPort(),
			"connectionPoolerEnabled": false,
		}
	}

	// hostname and port are read from endpoints.primary, not from spec. The
	// replica and pooler endpoints stay absent: this service has neither, so
	// their attributes must remain null across every plan.
	endpoints := func() map[string]interface{} {
		return map[string]interface{}{
			"primary": map[string]interface{}{
				"host": hostname,
				"port": currentPort(),
			},
			"replica": nil,
			"pooler":  nil,
		}
	}

	resources := []map[string]interface{}{
		{
			"id": "res-1",
			"spec": map[string]interface{}{
				"milliCPU":  500,
				"memoryGB":  2,
				"storageGB": 10,
			},
		},
	}

	server.Handle("CreateService", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		serviceCreated = true
		return map[string]interface{}{
			"data": map[string]interface{}{
				"createService": map[string]interface{}{
					"id":            "svc-123",
					"name":          "test-service",
					"regionCode":    "az-eastus2",
					"status":        "READY",
					"created":       "2024-01-01T00:00:00Z",
					"password":      "secret-password",
					"resources":     resources,
					"replicaStatus": nil,
					"spec":          spec(),
					"endpoints":     endpoints(),
				},
			},
		}
	})

	server.Handle("GetService", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		if !serviceCreated {
			return map[string]interface{}{"data": map[string]interface{}{"getService": nil}}
		}

		ids := []string{}
		for id := range attachedConnectionIDs {
			ids = append(ids, id)
		}

		return map[string]interface{}{
			"data": map[string]interface{}{
				"getService": map[string]interface{}{
					"id":                       "svc-123",
					"name":                     "test-service",
					"regionCode":               "az-eastus2",
					"status":                   "READY",
					"created":                  "2024-01-01T00:00:00Z",
					"replicaStatus":            nil,
					"resources":                resources,
					"spec":                     spec(),
					"endpoints":                endpoints(),
					"privateLinkConnectionIds": ids,
				},
			},
		}
	})

	server.Handle("AttachServiceToPrivateLinkConnection", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		for _, id := range GetStringSlice(GetVars(req), "connectionIds") {
			attachedConnectionIDs[id] = true
		}
		return map[string]interface{}{
			"data": map[string]interface{}{"attachServiceToPrivateLinkConnection": "OK"},
		}
	})

	server.Handle("DetachServiceFromPrivateLinkConnection", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		for _, id := range GetStringSlice(GetVars(req), "connectionIds") {
			delete(attachedConnectionIDs, id)
		}
		return map[string]interface{}{
			"data": map[string]interface{}{"detachServiceFromPrivateLinkConnection": "OK"},
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

	config := func(connectionIDs string) string {
		return ProviderConfig + `
resource "timescale_service" "test" {
  name                            = "test-service"
  milli_cpu                       = 500
  memory_gb                       = 2
  region_code                     = "az-eastus2"
  private_endpoint_connection_ids = ` + connectionIDs + `
}
`
	}

	detached := config(`[]`)
	attached := config(`["conn-123"]`)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: TestProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: detached,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_service.test", "hostname", hostname),
					resource.TestCheckResourceAttr("timescale_service.test", "port", "5432"),
				),
			},
			// Attaching must leave the hostname known and equal to the prior
			// value, while the port is free to be reallocated.
			{
				Config: attached,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectKnownValue(
							"timescale_service.test",
							tfjsonpath.New("hostname"),
							knownvalue.StringExact(hostname),
						),
						plancheck.ExpectUnknownValue(
							"timescale_service.test",
							tfjsonpath.New("port"),
						),
						plancheck.ExpectResourceAction(
							"timescale_service.test",
							plancheck.ResourceActionUpdate,
						),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_service.test", "hostname", hostname),
					resource.TestCheckResourceAttr("timescale_service.test", "port", "5433"),
				),
			},
			{
				Config:             attached,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Detaching is the same contract in reverse.
			{
				Config: detached,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectKnownValue(
							"timescale_service.test",
							tfjsonpath.New("hostname"),
							knownvalue.StringExact(hostname),
						),
						plancheck.ExpectUnknownValue(
							"timescale_service.test",
							tfjsonpath.New("port"),
						),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_service.test", "hostname", hostname),
					resource.TestCheckResourceAttr("timescale_service.test", "port", "5432"),
				),
			},
		},
	})
}
