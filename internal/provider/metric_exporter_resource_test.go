package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_MetricExporterResource_Datadog(t *testing.T) {
	const datadogConfig = `
resource "timescale_metric_exporter" "test_datadog" {
  name   = "tf-acc-test-datadog"
  region = "us-east-1"
  datadog = {
    api_key = "test"
    site    = "datadoghq.com"
  }
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + datadogConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_metric_exporter.test_datadog", "name", "tf-acc-test-datadog"),
					resource.TestCheckResourceAttr("timescale_metric_exporter.test_datadog", "region", "us-east-1"),
					resource.TestCheckResourceAttr("timescale_metric_exporter.test_datadog", "type", "datadog"),
					resource.TestCheckResourceAttr("timescale_metric_exporter.test_datadog", "datadog.site", "datadoghq.com"),
					resource.TestCheckResourceAttr("timescale_metric_exporter.test_datadog", "datadog.api_key", "test"),
					resource.TestCheckResourceAttrSet("timescale_metric_exporter.test_datadog", "id"),
					resource.TestCheckResourceAttrSet("timescale_metric_exporter.test_datadog", "created"),
				),
			},
			{
				ResourceName:      "timescale_metric_exporter.test_datadog",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAcc_MetricExporterResource_Prometheus(t *testing.T) {
	const prometheusConfig = `
resource "timescale_metric_exporter" "test_prometheus" {
  name   = "tf-acc-test-prometheus"
  region = "us-east-1"
  prometheus = {
    username = "testuser"
    password = "testpassword"
  }
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + prometheusConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_metric_exporter.test_prometheus", "name", "tf-acc-test-prometheus"),
					resource.TestCheckResourceAttr("timescale_metric_exporter.test_prometheus", "region", "us-east-1"),
					resource.TestCheckResourceAttr("timescale_metric_exporter.test_prometheus", "type", "prometheus"),
					resource.TestCheckResourceAttr("timescale_metric_exporter.test_prometheus", "prometheus.username", "testuser"),
					resource.TestCheckResourceAttrSet("timescale_metric_exporter.test_prometheus", "id"),
					resource.TestCheckResourceAttrSet("timescale_metric_exporter.test_prometheus", "created"),
				),
			},
			{
				ResourceName:      "timescale_metric_exporter.test_prometheus",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAcc_MetricExporterResource_Cloudwatch_RoleAuth(t *testing.T) {
	const cloudwatchRoleConfig = `
resource "timescale_metric_exporter" "test_cloudwatch_role" {
  name   = "tf-acc-test-cw-role"
  region = "us-east-1"
  cloudwatch = {
    region          = "us-east-1"
    log_group_name  = "test-log-group"
    log_stream_name = "test-log-stream"
    namespace       = "test-namespace"
    role_arn        = "arn:aws:iam::123456789012:role/test-role"
  }
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + cloudwatchRoleConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_metric_exporter.test_cloudwatch_role", "name", "tf-acc-test-cw-role"),
					resource.TestCheckResourceAttr("timescale_metric_exporter.test_cloudwatch_role", "region", "us-east-1"),
					resource.TestCheckResourceAttr("timescale_metric_exporter.test_cloudwatch_role", "type", "cloudwatch"),
					resource.TestCheckResourceAttr("timescale_metric_exporter.test_cloudwatch_role", "cloudwatch.region", "us-east-1"),
					resource.TestCheckResourceAttr("timescale_metric_exporter.test_cloudwatch_role", "cloudwatch.log_group_name", "test-log-group"),
					resource.TestCheckResourceAttr("timescale_metric_exporter.test_cloudwatch_role", "cloudwatch.log_stream_name", "test-log-stream"),
					resource.TestCheckResourceAttr("timescale_metric_exporter.test_cloudwatch_role", "cloudwatch.namespace", "test-namespace"),
					resource.TestCheckResourceAttr("timescale_metric_exporter.test_cloudwatch_role", "cloudwatch.role_arn", "arn:aws:iam::123456789012:role/test-role"),
					resource.TestCheckResourceAttrSet("timescale_metric_exporter.test_cloudwatch_role", "id"),
					resource.TestCheckResourceAttrSet("timescale_metric_exporter.test_cloudwatch_role", "created"),
				),
			},
			{
				ResourceName:      "timescale_metric_exporter.test_cloudwatch_role",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAcc_MetricExporterResource_Cloudwatch_KeyAuth(t *testing.T) {
	const cloudwatchKeyConfig = `
resource "timescale_metric_exporter" "test_cloudwatch_key" {
  name   = "tf-acc-test-cw-key"
  region = "us-east-1"
  cloudwatch = {
    region          = "us-east-1"
    log_group_name  = "key-log-group"
    log_stream_name = "key-log-stream"
    namespace       = "key-namespace"
    access_key      = "XXXXXXXXXXXXXXXXXXXX"
    secret_key      = "XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"
  }
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + cloudwatchKeyConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_metric_exporter.test_cloudwatch_key", "name", "tf-acc-test-cw-key"),
					resource.TestCheckResourceAttr("timescale_metric_exporter.test_cloudwatch_key", "region", "us-east-1"),
					resource.TestCheckResourceAttr("timescale_metric_exporter.test_cloudwatch_key", "type", "cloudwatch"),
					resource.TestCheckResourceAttr("timescale_metric_exporter.test_cloudwatch_key", "cloudwatch.region", "us-east-1"),
					resource.TestCheckResourceAttr("timescale_metric_exporter.test_cloudwatch_key", "cloudwatch.log_group_name", "key-log-group"),
					resource.TestCheckResourceAttr("timescale_metric_exporter.test_cloudwatch_key", "cloudwatch.log_stream_name", "key-log-stream"),
					resource.TestCheckResourceAttr("timescale_metric_exporter.test_cloudwatch_key", "cloudwatch.namespace", "key-namespace"),
					resource.TestCheckResourceAttrSet("timescale_metric_exporter.test_cloudwatch_key", "id"),
					resource.TestCheckResourceAttrSet("timescale_metric_exporter.test_cloudwatch_key", "created"),
				),
			},
			{
				ResourceName:      "timescale_metric_exporter.test_cloudwatch_key",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccMetricExporterResource_validations(t *testing.T) {
	testCases := []struct {
		name        string
		config      string
		expectError *regexp.Regexp
		check       resource.TestCheckFunc
	}{
		{
			name: "MissingExporterConfiguration",
			config: `
resource "timescale_metric_exporter" "test" {
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
resource "timescale_metric_exporter" "test" {
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
resource "timescale_metric_exporter" "test" {
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
resource "timescale_metric_exporter" "test" {
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
resource "timescale_metric_exporter" "test" {
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
resource "timescale_metric_exporter" "test" {
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
