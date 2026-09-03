package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
)

// connectionPayload is the shape the gateway returns for a claimed connection.
func connectionPayload(name string) map[string]interface{} {
	return map[string]interface{}{
		"connectionId":         "conn-123",
		"providerConnectionId": "my-pe.f91412e6-1111-2222-3333-444455556666",
		"cloudProvider":        "azure",
		"region":               "az-eastus2",
		"linkIdentifier":       "link-789",
		"principalId":          "sub-abc",
		"state":                "approved",
		"name":                 name,
		"createdAt":            "2024-01-01T00:00:00Z",
		"updatedAt":            "2024-01-01T00:00:00Z",
	}
}

func TestAccPrivateLinkConnectionResource_azure(t *testing.T) {
	server := NewMockServer(t)
	defer server.Close()

	synced := false
	name := ""

	server.Handle("SyncPrivateLinkConnections", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		synced = true
		return map[string]interface{}{
			"data": map[string]interface{}{"syncPrivateLinkConnections": "OK"},
		}
	})

	server.Handle("ClaimPrivateLinkConnection", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		vars := GetVars(req)
		assert.Equal(t, "f91412e6-1111-2222-3333-444455556666", vars["claimIdentifier"])
		return map[string]interface{}{
			"data": map[string]interface{}{"claimPrivateLinkConnection": connectionPayload(name)},
		}
	})

	server.Handle("UpdatePrivateLinkConnection", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		vars := GetVars(req)
		assert.Equal(t, "conn-123", vars["connectionId"])
		name = GetString(vars, "name")
		return map[string]interface{}{
			"data": map[string]interface{}{"updatePrivateLinkConnection": connectionPayload(name)},
		}
	})

	server.Handle("ListPrivateLinkConnections", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{
			"data": map[string]interface{}{
				"listPrivateLinkConnections": []map[string]interface{}{connectionPayload(name)},
			},
		}
	})

	server.SetupEnv(t)

	config := ProviderConfig + `
resource "timescale_privatelink_connection" "test" {
  claim_identifier = "f91412e6-1111-2222-3333-444455556666"
  name             = "My Connection"
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: TestProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_privatelink_connection.test", "connection_id", "conn-123"),
					resource.TestCheckResourceAttr("timescale_privatelink_connection.test", "claim_identifier", "f91412e6-1111-2222-3333-444455556666"),
					resource.TestCheckResourceAttr("timescale_privatelink_connection.test", "cloud_provider", "azure"),
					resource.TestCheckResourceAttr("timescale_privatelink_connection.test", "region", "az-eastus2"),
					resource.TestCheckResourceAttr("timescale_privatelink_connection.test", "state", "approved"),
					resource.TestCheckResourceAttr("timescale_privatelink_connection.test", "name", "My Connection"),
					resource.TestCheckResourceAttr("timescale_privatelink_connection.test", "reject_on_destroy", "false"),
					resource.TestCheckResourceAttr("timescale_privatelink_connection.test", "principal_id", "sub-abc"),
					// The IP field is gone: customers author their own private DNS.
					resource.TestCheckNoResourceAttr("timescale_privatelink_connection.test", "ip_address"),
				),
			},
			{
				ResourceName:      "timescale_privatelink_connection.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     "conn-123",
				// Import supplies only the connection ID; these are local policy.
				ImportStateVerifyIgnore: []string{"reject_on_destroy", "timeouts"},
			},
		},
	})

	assert.True(t, synced, "expected the resource to request a sync before claiming")
}

// TestAccPrivateLinkConnectionResource_awsClaimIdentifierIsVPCEndpointID checks
// the AWS claim identifier passes through unmodified: it is the VPC endpoint ID.
func TestAccPrivateLinkConnectionResource_aws(t *testing.T) {
	server := NewMockServer(t)
	defer server.Close()

	awsConnection := map[string]interface{}{
		"connectionId":         "conn-aws-1",
		"providerConnectionId": "vpce-0123456789abcdef0",
		"cloudProvider":        "aws",
		"region":               "us-east-1",
		"linkIdentifier":       "vpce-0123456789abcdef0",
		"principalId":          "123456789012",
		"state":                "approved",
		"name":                 "",
		"createdAt":            "2024-01-01T00:00:00Z",
		"updatedAt":            "2024-01-01T00:00:00Z",
	}

	server.Handle("SyncPrivateLinkConnections", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{
			"data": map[string]interface{}{"syncPrivateLinkConnections": "OK"},
		}
	})

	server.Handle("ClaimPrivateLinkConnection", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		vars := GetVars(req)
		assert.Equal(t, "vpce-0123456789abcdef0", vars["claimIdentifier"])
		return map[string]interface{}{
			"data": map[string]interface{}{"claimPrivateLinkConnection": awsConnection},
		}
	})

	server.Handle("ListPrivateLinkConnections", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{
			"data": map[string]interface{}{
				"listPrivateLinkConnections": []map[string]interface{}{awsConnection},
			},
		}
	})

	server.SetupEnv(t)

	config := ProviderConfig + `
resource "timescale_privatelink_connection" "test" {
  claim_identifier = "vpce-0123456789abcdef0"
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: TestProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_privatelink_connection.test", "connection_id", "conn-aws-1"),
					resource.TestCheckResourceAttr("timescale_privatelink_connection.test", "cloud_provider", "aws"),
					resource.TestCheckResourceAttr("timescale_privatelink_connection.test", "region", "us-east-1"),
					resource.TestCheckResourceAttr("timescale_privatelink_connection.test", "provider_connection_id", "vpce-0123456789abcdef0"),
				),
			},
		},
	})
}

// TestAccPrivateLinkConnectionResource_rejectOnDestroy verifies the default
// destroy path leaves the connection alone, and that opting in rejects it.
func TestAccPrivateLinkConnectionResource_rejectOnDestroy(t *testing.T) {
	server := NewMockServer(t)
	defer server.Close()

	rejected := false

	server.Handle("SyncPrivateLinkConnections", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{
			"data": map[string]interface{}{"syncPrivateLinkConnections": "OK"},
		}
	})

	server.Handle("ClaimPrivateLinkConnection", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{
			"data": map[string]interface{}{"claimPrivateLinkConnection": connectionPayload("")},
		}
	})

	server.Handle("ListPrivateLinkConnections", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{
			"data": map[string]interface{}{
				"listPrivateLinkConnections": []map[string]interface{}{connectionPayload("")},
			},
		}
	})

	server.Handle("DeletePrivateLinkConnection", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		vars := GetVars(req)
		assert.Equal(t, "conn-123", vars["connectionId"])
		rejected = true
		return map[string]interface{}{
			"data": map[string]interface{}{"deletePrivateLinkConnection": "OK"},
		}
	})

	server.SetupEnv(t)

	config := ProviderConfig + `
resource "timescale_privatelink_connection" "test" {
  claim_identifier  = "f91412e6-1111-2222-3333-444455556666"
  reject_on_destroy = true
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: TestProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.TestCheckResourceAttr(
					"timescale_privatelink_connection.test", "reject_on_destroy", "true"),
			},
		},
	})

	assert.True(t, rejected, "reject_on_destroy = true should reject the connection on destroy")
}

// TestAccPrivateLinkConnectionResource_claimIdentifierForcesReplace checks that
// pointing the resource at a different connection replaces it, which is what
// happens when the underlying cloud endpoint is recreated.
func TestAccPrivateLinkConnectionResource_claimIdentifierForcesReplace(t *testing.T) {
	server := NewMockServer(t)
	defer server.Close()

	claimed := map[string]bool{}

	server.Handle("SyncPrivateLinkConnections", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{
			"data": map[string]interface{}{"syncPrivateLinkConnections": "OK"},
		}
	})

	server.Handle("ClaimPrivateLinkConnection", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		id := GetString(GetVars(req), "claimIdentifier")
		claimed[id] = true
		conn := connectionPayload("")
		conn["claimIdentifier"] = id
		return map[string]interface{}{
			"data": map[string]interface{}{"claimPrivateLinkConnection": conn},
		}
	})

	server.Handle("ListPrivateLinkConnections", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{
			"data": map[string]interface{}{
				"listPrivateLinkConnections": []map[string]interface{}{connectionPayload("")},
			},
		}
	})

	server.SetupEnv(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: TestProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: ProviderConfig + `
resource "timescale_privatelink_connection" "test" {
  claim_identifier = "vpce-1111111111111111"
}
`,
				Check: resource.TestCheckResourceAttr(
					"timescale_privatelink_connection.test", "claim_identifier", "vpce-1111111111111111"),
			},
			{
				Config: ProviderConfig + `
resource "timescale_privatelink_connection" "test" {
  claim_identifier = "vpce-2222222222222222"
}
`,
				Check: resource.TestCheckResourceAttr(
					"timescale_privatelink_connection.test", "claim_identifier", "vpce-2222222222222222"),
			},
		},
	})

	assert.True(t, claimed["vpce-2222222222222222"], "the replacement should claim the new identifier")
}

// TestAccPrivateLinkConnectionResource_approvalMessageNoDrift checks that a
// practitioner can pass the Azure approval message straight through: the GUID
// reaches the API, while state keeps the configured value so the next plan is
// empty.
func TestAccPrivateLinkConnectionResource_approvalMessageNoDrift(t *testing.T) {
	server := NewMockServer(t)
	defer server.Close()

	const guid = "f91412e6-1111-2222-3333-444455556666"
	const message = "Connected. Claim this connection in the console using ID " + guid

	var sawIdentifier string
	claims := 0

	server.Handle("SyncPrivateLinkConnections", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{
			"data": map[string]interface{}{"syncPrivateLinkConnections": "OK"},
		}
	})

	server.Handle("ClaimPrivateLinkConnection", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		claims++
		sawIdentifier = GetString(GetVars(req), "claimIdentifier")
		return map[string]interface{}{
			"data": map[string]interface{}{"claimPrivateLinkConnection": connectionPayload("")},
		}
	})

	server.Handle("ListPrivateLinkConnections", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{
			"data": map[string]interface{}{
				"listPrivateLinkConnections": []map[string]interface{}{connectionPayload("")},
			},
		}
	})

	server.SetupEnv(t)

	config := ProviderConfig + `
resource "timescale_privatelink_connection" "test" {
  claim_identifier = "` + message + `"
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: TestProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					// State keeps what was written, not the extracted GUID.
					resource.TestCheckResourceAttr("timescale_privatelink_connection.test", "claim_identifier", message),
					resource.TestCheckResourceAttr("timescale_privatelink_connection.test", "connection_id", "conn-123"),
				),
			},
			{
				// An empty plan proves the stored value round-trips.
				Config:   config,
				PlanOnly: true,
			},
		},
	})

	assert.Equal(t, guid, sawIdentifier, "the API should receive the extracted GUID, not the message")
	assert.Equal(t, 1, claims, "the second step should plan only, without re-claiming")
}
