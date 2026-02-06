package provider_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
)

func TestAccPrivateLinkAuthorizationResource_basic(t *testing.T) {
	server := NewMockServer(t)
	defer server.Close()

	server.Handle("CreatePrivateLinkAuthorization", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		vars := GetVars(req)
		assert.Equal(t, "test-subscription-id", vars["subscriptionId"])
		assert.Equal(t, "test-authorization", vars["name"])
		assert.Equal(t, "test-project-id", vars["projectId"])

		return map[string]interface{}{
			"data": map[string]interface{}{
				"createPrivateLinkAuthorization": map[string]interface{}{
					"subscriptionId": "test-subscription-id",
					"name":           "test-authorization",
					"createdAt":      "2024-01-01T00:00:00Z",
					"updatedAt":      nil,
				},
			},
		}
	})

	server.Handle("ListPrivateLinkAuthorizations", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		vars := GetVars(req)
		assert.Equal(t, "test-project-id", vars["projectId"])

		return map[string]interface{}{
			"data": map[string]interface{}{
				"listPrivateLinkAuthorizations": []map[string]interface{}{
					{
						"subscriptionId": "test-subscription-id",
						"name":           "test-authorization",
						"createdAt":      "2024-01-01T00:00:00Z",
						"updatedAt":      nil,
					},
				},
			},
		}
	})

	server.Handle("DeletePrivateLinkAuthorization", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		vars := GetVars(req)
		assert.Equal(t, "test-subscription-id", vars["subscriptionId"])
		assert.Equal(t, "test-project-id", vars["projectId"])

		return map[string]interface{}{
			"data": map[string]interface{}{
				"deletePrivateLinkAuthorization": "OK",
			},
		}
	})

	server.SetupEnv(t)

	config := ProviderConfig + `
resource "timescale_privatelink_authorization" "test" {
  subscription_id = "test-subscription-id"
  name            = "test-authorization"
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: TestProviderFactories(),
		Steps: []resource.TestStep{
			// Step 1: Plan only - verify plan shows correct changes
			{
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			// Step 2: Apply and verify state
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_privatelink_authorization.test", "subscription_id", "test-subscription-id"),
					resource.TestCheckResourceAttr("timescale_privatelink_authorization.test", "name", "test-authorization"),
					resource.TestCheckResourceAttr("timescale_privatelink_authorization.test", "id", "test-subscription-id"),
				),
			},
			// Step 3: Plan again - verify no drift (empty plan)
			{
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAccPrivateLinkAuthorizationResource_invalidSubscription(t *testing.T) {
	server := NewMockServer(t)
	defer server.Close()

	server.Handle("CreatePrivateLinkAuthorization", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{
			"data": nil,
			"errors": []map[string]string{
				{"message": "Invalid subscription ID format"},
			},
		}
	})

	server.SetupEnv(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: TestProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: ProviderConfig + `
resource "timescale_privatelink_authorization" "test" {
  subscription_id = "invalid-subscription"
  name            = "test-authorization"
}
`,
				ExpectError: regexp.MustCompile("Invalid subscription ID format"),
			},
		},
	})
}
