package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func testAccPrivateLinkAWSPreCheck(t *testing.T) {
	testAccPreCheck(t)
	if _, ok := os.LookupEnv("AWS_ACCESS_KEY_ID"); !ok {
		t.Skip("AWS_ACCESS_KEY_ID not set, skipping AWS Private Link test")
	}
	if _, ok := os.LookupEnv("AWS_SECRET_ACCESS_KEY"); !ok {
		t.Skip("AWS_SECRET_ACCESS_KEY not set, skipping AWS Private Link test")
	}
}

func TestAccPrivateLinkConnection_aws_e2e(t *testing.T) {
	connectionName := "timescale_privatelink_connection.test"
	serviceName := "timescale_service.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		ExternalProviders: map[string]resource.ExternalProvider{
			"aws": {
				Source:            "hashicorp/aws",
				VersionConstraint: ">= 5.0",
			},
		},
		PreCheck: func() { testAccPrivateLinkAWSPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccPrivateLinkAWSFullConfig("test-acc-managed", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(connectionName, "connection_id"),
					resource.TestCheckResourceAttrSet(connectionName, "state"),
					resource.TestCheckResourceAttrSet(connectionName, "claim_identifier"),
					resource.TestCheckResourceAttr(connectionName, "name", "test-acc-managed"),
					resource.TestCheckResourceAttr(connectionName, "cloud_provider", "aws"),
					resource.TestCheckResourceAttr(connectionName, "region", "us-east-1"),
					resource.TestCheckResourceAttrSet(serviceName, "id"),
					resource.TestCheckResourceAttrSet(serviceName, "hostname"),
					resource.TestCheckResourceAttr(serviceName, "private_endpoint_connection_ids.#", "1"),
				),
			},
			{
				Config: testAccPrivateLinkAWSFullConfig("test-acc-renamed", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(connectionName, "name", "test-acc-renamed"),
				),
			},
			{
				Config: testAccPrivateLinkAWSFullConfig("test-acc-renamed", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(serviceName, "id"),
					resource.TestCheckResourceAttr(serviceName, "private_endpoint_connection_ids.#", "0"),
				),
			},
			{
				Config: testAccPrivateLinkAWSFullConfig("test-acc-renamed", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(serviceName, "private_endpoint_connection_ids.#", "1"),
				),
			},
			{
				ResourceName:      connectionName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					rs, ok := state.RootModule().Resources[connectionName]
					if !ok {
						return "", fmt.Errorf("resource %s not found in state", connectionName)
					}
					return rs.Primary.Attributes["connection_id"], nil
				},
				// reject_on_destroy is provider-side policy, not remote state.
				ImportStateVerifyIgnore: []string{"reject_on_destroy", "timeouts"},
			},
		},
	})
}

func testAccPrivateLinkAWSBaseConfig() string {
	return providerConfig + `
provider "aws" {
  region = "us-east-1"
}

data "timescale_privatelink_region" "test" {
  region = "us-east-1"
}

resource "aws_vpc" "test" {
  cidr_block           = "10.0.0.0/16"
  enable_dns_support   = true
  enable_dns_hostnames = true

  tags = {
    Name = "tf-acc-test-pl-vpc"
  }
}

resource "aws_subnet" "test" {
  vpc_id            = aws_vpc.test.id
  cidr_block        = "10.0.1.0/24"
  availability_zone = "us-east-1a"

  tags = {
    Name = "tf-acc-test-pl-subnet"
  }
}

resource "aws_vpc_endpoint" "test" {
  vpc_id            = aws_vpc.test.id
  service_name      = data.timescale_privatelink_region.test.service_name
  vpc_endpoint_type = "Interface"
  subnet_ids        = [aws_subnet.test.id]

  tags = {
    Name = "tf-acc-test-pl-vpce"
  }
}
`
}

func testAccPrivateLinkAWSConnectionConfig(name string) string {
	return fmt.Sprintf(`
resource "timescale_privatelink_connection" "test" {
  claim_identifier  = aws_vpc_endpoint.test.id
  name              = %q
  reject_on_destroy = true

  timeouts = {
    create = "10m"
  }
}
`, name)
}

func testAccPrivateLinkAWSServiceConfig(attached bool) string {
	connectionIDLine := ""
	if attached {
		connectionIDLine = "\n  private_endpoint_connection_ids = [timescale_privatelink_connection.test.connection_id]"
	}
	return fmt.Sprintf(`
resource "timescale_service" "test" {
  name        = "tf-acc-test-pl-service"
  milli_cpu   = 500
  memory_gb   = 2
  region_code = "us-east-1"
  timeouts = {
    create = "15m"
  }%s
}
`, connectionIDLine)
}

func testAccPrivateLinkAWSFullConfig(connectionName string, attached bool) string {
	return testAccPrivateLinkAWSBaseConfig() +
		testAccPrivateLinkAWSConnectionConfig(connectionName) +
		testAccPrivateLinkAWSServiceConfig(attached)
}
