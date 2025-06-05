package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccMetricsExporterResource_validations(t *testing.T) {
	testCases := []struct {
		name        string
		config      string
		expectError *regexp.Regexp
		check       resource.TestCheckFunc
	}{
		{
			name: "MissingExporterConfiguration",
			config: `
resource "timescale_metrics_exporter" "test" {
  name   = "missing-exporter"
  region = "us-east-1"
  # No datadog, prometheus, or cloudwatch block
}
`,
			expectError: regexp.MustCompile(
				"Missing Exporter Configuration",
			),
		},
		{
			name: "ConflictingExporterTypes",
			config: `
resource "timescale_metrics_exporter" "test" {
  name   = "conflicting-exporter"
  region = "us-east-1"

  datadog = {
    api_key = "dd_api_key"
    site    = "datadoghq.com"
  }
  prometheus = {
    username = "user"
    password = "password"
  }
}
`,
			expectError: regexp.MustCompile(
				"Conflicting Exporter Configuration",
			),
		},
		{
			name: "CloudWatch_ConflictingAuthentication",
			config: `
resource "timescale_metrics_exporter" "test" {
  name   = "cw-conflicting-auth"
  region = "eu-central-1"

  cloudwatch = {
    log_group_name  = "/test/group"
    log_stream_name = "test-stream"
    namespace       = "Test/Namespace"
    region          = "us-east-1"
    role_arn        = "arn:aws:iam::123456789012:role/TestRole"
    access_key      = "TESTACCESSKEY"
    secret_key      = "TESTSECRETKEY"
  }
}
`,
			expectError: regexp.MustCompile("Conflicting Authentication"),
		},
		{
			name: "CloudWatch_IncompleteKeyAuth_AccessOnly",
			config: `
resource "timescale_metrics_exporter" "test" {
  name   = "cw-incomplete-key-access"
  region = "ap-southeast-1"

  cloudwatch = {
    log_group_name  = "/test/group"
    log_stream_name = "test-stream"
    namespace       = "Test/Namespace"
    region          = "us-east-1"
    access_key      = "TESTACCESSKEY"
  }
}
`,
			expectError: regexp.MustCompile("Incomplete Key Authentication"),
		},
		{
			name: "CloudWatch_IncompleteKeyAuth_SecretOnly",
			config: `
resource "timescale_metrics_exporter" "test" {
  name   = "cw-incomplete-key-secret"
  region = "ca-central-1"

  cloudwatch = {
    log_group_name  = "/test/group"
    log_stream_name = "test-stream"
    namespace       = "Test/Namespace"
    region          = "us-east-1"
    secret_key      = "TESTSECRETKEY"
  }
}
`,
			expectError: regexp.MustCompile("Incomplete Key Authentication"),
		},
		{
			name: "CloudWatch_MissingAuthMethod",
			config: `
resource "timescale_metrics_exporter" "test" {
  name   = "cw-missing-auth"
  region = "sa-east-1"

  cloudwatch = {
    log_group_name  = "/test/group"
    log_stream_name = "test-stream"
    namespace       = "Test/Namespace"
    region          = "us-east-1"
  }
}
`,
			expectError: regexp.MustCompile("Missing Authentication Method"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			step := resource.TestStep{
				Config: providerConfig + tc.config,
			}
			if tc.expectError != nil {
				step.ExpectError = tc.expectError
			}
			if tc.check != nil {
				step.Check = tc.check
			}

			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps:                    []resource.TestStep{step},
			})
		})
	}
}
