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
		assert.Equal(t, "test-subscription-id", vars["principalId"])
		assert.Equal(t, "AZURE", vars["cloudProvider"])
		assert.Equal(t, "test-authorization", vars["name"])
		assert.Equal(t, "test-project-id", vars["projectId"])

		return map[string]interface{}{
			"data": map[string]interface{}{
				"createPrivateLinkAuthorization": map[string]interface{}{
					"principalId":   "test-subscription-id",
					"cloudProvider": "AZURE",
					"name":          "test-authorization",
					"createdAt":     "2024-01-01T00:00:00Z",
					"updatedAt":     nil,
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
						"principalId":   "test-subscription-id",
						"cloudProvider": "AZURE",
						"name":          "test-authorization",
						"createdAt":     "2024-01-01T00:00:00Z",
						"updatedAt":     nil,
					},
				},
			},
		}
	})

	server.Handle("DeletePrivateLinkAuthorization", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		vars := GetVars(req)
		assert.Equal(t, "test-subscription-id", vars["principalId"])
		assert.Equal(t, "AZURE", vars["cloudProvider"])
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
  principal_id   = "test-subscription-id"
  cloud_provider = "AZURE"
  name           = "test-authorization"
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
					resource.TestCheckResourceAttr("timescale_privatelink_authorization.test", "principal_id", "test-subscription-id"),
					resource.TestCheckResourceAttr("timescale_privatelink_authorization.test", "cloud_provider", "AZURE"),
					resource.TestCheckResourceAttr("timescale_privatelink_authorization.test", "name", "test-authorization"),
					resource.TestCheckResourceAttr("timescale_privatelink_authorization.test", "id", "test-subscription-id"),
				),
			},
			// Step 3: Import and verify state matches
			{
				ResourceName:      "timescale_privatelink_authorization.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     "AZURE,test-subscription-id",
			},
			// Step 4: Plan again - verify no drift (empty plan)
			{
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAccPrivateLinkAuthorizationResource_aws(t *testing.T) {
	server := NewMockServer(t)
	defer server.Close()

	server.Handle("CreatePrivateLinkAuthorization", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		vars := GetVars(req)
		assert.Equal(t, "123456789012", vars["principalId"])
		assert.Equal(t, "AWS", vars["cloudProvider"])
		assert.Equal(t, "aws-test-auth", vars["name"])

		return map[string]interface{}{
			"data": map[string]interface{}{
				"createPrivateLinkAuthorization": map[string]interface{}{
					"principalId":   "123456789012",
					"cloudProvider": "AWS",
					"name":          "aws-test-auth",
					"createdAt":     "2024-01-01T00:00:00Z",
					"updatedAt":     nil,
				},
			},
		}
	})

	server.Handle("ListPrivateLinkAuthorizations", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{
			"data": map[string]interface{}{
				"listPrivateLinkAuthorizations": []map[string]interface{}{
					{
						"principalId":   "123456789012",
						"cloudProvider": "AWS",
						"name":          "aws-test-auth",
						"createdAt":     "2024-01-01T00:00:00Z",
						"updatedAt":     nil,
					},
				},
			},
		}
	})

	server.Handle("DeletePrivateLinkAuthorization", func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		vars := GetVars(req)
		assert.Equal(t, "123456789012", vars["principalId"])
		assert.Equal(t, "AWS", vars["cloudProvider"])

		return map[string]interface{}{
			"data": map[string]interface{}{
				"deletePrivateLinkAuthorization": "OK",
			},
		}
	})

	server.SetupEnv(t)

	config := ProviderConfig + `
resource "timescale_privatelink_authorization" "test" {
  principal_id   = "123456789012"
  cloud_provider = "AWS"
  name           = "aws-test-auth"
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: TestProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_privatelink_authorization.test", "principal_id", "123456789012"),
					resource.TestCheckResourceAttr("timescale_privatelink_authorization.test", "cloud_provider", "AWS"),
					resource.TestCheckResourceAttr("timescale_privatelink_authorization.test", "name", "aws-test-auth"),
					resource.TestCheckResourceAttr("timescale_privatelink_authorization.test", "id", "123456789012"),
				),
			},
			{
				ResourceName:      "timescale_privatelink_authorization.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     "AWS,123456789012",
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
  principal_id   = "invalid-subscription"
  cloud_provider = "AZURE"
  name           = "test-authorization"
}
`,
				ExpectError: regexp.MustCompile("Invalid subscription ID format"),
			},
		},
	})
}
