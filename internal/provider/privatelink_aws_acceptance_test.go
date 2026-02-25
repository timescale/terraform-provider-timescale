package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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
				Config: testAccPrivateLinkAWSFullConfig("Managed by Terraform", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(connectionName, "connection_id"),
					resource.TestCheckResourceAttrSet(connectionName, "state"),
					resource.TestCheckResourceAttr(connectionName, "name", "Managed by Terraform"),
					resource.TestCheckResourceAttr(connectionName, "cloud_provider", "aws"),
					resource.TestCheckResourceAttr(connectionName, "region", "us-east-1"),
					resource.TestCheckResourceAttrSet(serviceName, "id"),
					resource.TestCheckResourceAttrSet(serviceName, "hostname"),
					resource.TestCheckResourceAttrSet(serviceName, "private_endpoint_connection_id"),
				),
			},
			{
				Config: testAccPrivateLinkAWSFullConfig("Updated Name", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(connectionName, "name", "Updated Name"),
				),
			},
			{
				Config: testAccPrivateLinkAWSFullConfig("Updated Name", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(serviceName, "id"),
					resource.TestCheckResourceAttr(serviceName, "private_endpoint_connection_id", ""),
				),
			},
			{
				Config: testAccPrivateLinkAWSFullConfig("Updated Name", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(serviceName, "private_endpoint_connection_id"),
				),
			},
		},
	})
}

func testAccPrivateLinkAWSBaseConfig() string {
	return providerConfig + `
provider "aws" {
  region = "us-east-1"
}

data "aws_caller_identity" "current" {}

data "timescale_privatelink_available_regions" "all" {}

resource "timescale_privatelink_authorization" "test" {
  principal_id   = data.aws_caller_identity.current.account_id
  cloud_provider = "aws"
  name           = "Terraform managed - acceptance test"
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
  service_name      = data.timescale_privatelink_available_regions.all.regions["us-east-1"].service_name
  vpc_endpoint_type = "GatewayLoadBalancer"
  subnet_ids        = [aws_subnet.test.id]

  tags = {
    Name = "tf-acc-test-pl-vpce"
  }

  depends_on = [timescale_privatelink_authorization.test]
}

data "aws_network_interface" "test" {
  id = one(aws_vpc_endpoint.test.network_interface_ids)
}
`
}

func testAccPrivateLinkAWSConnectionConfig(name string) string {
	return fmt.Sprintf(`
resource "timescale_privatelink_connection" "test" {
  provider_connection_id = aws_vpc_endpoint.test.id
  cloud_provider         = "aws"
  region                 = "us-east-1"
  ip_address             = data.aws_network_interface.test.private_ip
  name                   = %q
  timeout                = "5m"

  depends_on = [aws_vpc_endpoint.test]

  lifecycle {
    create_before_destroy = true
  }
}
`, name)
}

func testAccPrivateLinkAWSServiceConfig(attached bool) string {
	connectionIDLine := ""
	if attached {
		connectionIDLine = "\n  private_endpoint_connection_id = timescale_privatelink_connection.test.connection_id"
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
