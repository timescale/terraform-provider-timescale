package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccVPCResource_UnsupportedAzureRegion(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "timescale_vpcs" "test" {
  name        = "test-vpc-azure"
  cidr        = "10.0.0.0/16"
  region_code = "az-eastus"
}
`,
				ExpectError: regexp.MustCompile(`region az-eastus not supported`),
			},
		},
	})
}

func TestAccMetricExporterResource_UnsupportedAzureRegion(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "timescale_metric_exporter" "test" {
  name   = "test-exporter-azure"
  region = "az-westeurope"
  datadog = {
    api_key = "test"
    site    = "datadoghq.com"
  }
}
`,
				ExpectError: regexp.MustCompile(`region az-westeurope not supported`),
			},
		},
	})
}

func TestAccLogExporterResource_UnsupportedAzureRegion(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "timescale_log_exporter" "test" {
  name   = "test-log-exporter-azure"
  region = "az-australiaeast"
  cloudwatch = {
    region          = "us-east-1"
    log_group_name  = "test-group"
    log_stream_name = "test-stream"
    role_arn        = "arn:aws:iam::123456789012:role/test-role"
  }
}
`,
				ExpectError: regexp.MustCompile(`region az-australiaeast not supported`),
			},
		},
	})
}
