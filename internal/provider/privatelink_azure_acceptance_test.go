package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccPrivateLinkAzurePreCheck(t *testing.T) {
	testAccPreCheck(t)
	for _, env := range []string{"ARM_CLIENT_ID", "ARM_CLIENT_SECRET", "ARM_TENANT_ID", "ARM_SUBSCRIPTION_ID"} {
		if _, ok := os.LookupEnv(env); !ok {
			t.Skipf("%s not set, skipping Azure Private Link test", env)
		}
	}
}

func TestAccPrivateLinkConnection_azure_e2e(t *testing.T) {
	connectionName := "timescale_privatelink_connection.test"
	serviceName := "timescale_service.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		ExternalProviders: map[string]resource.ExternalProvider{
			"azurerm": {
				Source:            "hashicorp/azurerm",
				VersionConstraint: ">= 3.70.0",
			},
		},
		PreCheck: func() { testAccPrivateLinkAzurePreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccPrivateLinkAzureFullConfig("Managed by Terraform", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(connectionName, "connection_id"),
					resource.TestCheckResourceAttrSet(connectionName, "state"),
					resource.TestCheckResourceAttr(connectionName, "name", "Managed by Terraform"),
					resource.TestCheckResourceAttr(connectionName, "cloud_provider", "azure"),
					resource.TestCheckResourceAttr(connectionName, "region", "az-eastus"),
					resource.TestCheckResourceAttrSet(serviceName, "id"),
					resource.TestCheckResourceAttrSet(serviceName, "hostname"),
					resource.TestCheckResourceAttrSet(serviceName, "private_endpoint_connection_id"),
				),
			},
			{
				Config: testAccPrivateLinkAzureFullConfig("Updated Name", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(connectionName, "name", "Updated Name"),
				),
			},
			{
				Config: testAccPrivateLinkAzureFullConfig("Updated Name", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(serviceName, "id"),
					resource.TestCheckResourceAttr(serviceName, "private_endpoint_connection_id", ""),
				),
			},
			{
				Config: testAccPrivateLinkAzureFullConfig("Updated Name", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(serviceName, "private_endpoint_connection_id"),
				),
			},
		},
	})
}

func testAccPrivateLinkAzureBaseConfig() string {
	subscriptionID := os.Getenv("ARM_SUBSCRIPTION_ID")
	clientID := os.Getenv("ARM_CLIENT_ID")
	clientSecret := os.Getenv("ARM_CLIENT_SECRET")
	tenantID := os.Getenv("ARM_TENANT_ID")
	return providerConfig + fmt.Sprintf(`
provider "azurerm" {
  features {}
  subscription_id = %q
  client_id       = %q
  client_secret   = %q
  tenant_id       = %q
}

data "timescale_privatelink_available_regions" "all" {}

resource "azurerm_resource_group" "test" {
  name     = "tf-acc-test-pl-rg"
  location = "eastus"
}

resource "azurerm_virtual_network" "test" {
  name                = "tf-acc-test-pl-vnet"
  address_space       = ["10.3.0.0/16"]
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
}

resource "azurerm_subnet" "test" {
  name                              = "endpoint-subnet"
  resource_group_name               = azurerm_resource_group.test.name
  virtual_network_name              = azurerm_virtual_network.test.name
  address_prefixes                  = ["10.3.2.0/24"]
  private_endpoint_network_policies = "Disabled"
}

resource "timescale_privatelink_authorization" "test" {
  principal_id   = %q
  cloud_provider = "azure"
  name           = "Terraform managed - acceptance test"
}

resource "azurerm_private_endpoint" "test" {
  name                = "tf-acc-test-pl-pe"
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
  subnet_id           = azurerm_subnet.test.id

  private_service_connection {
    name                              = "tf-acc-test-pl-psc"
    private_connection_resource_alias = data.timescale_privatelink_available_regions.all.regions["az-eastus"].service_name
    is_manual_connection              = true
    request_message                   = var.ts_project_id
  }

  depends_on = [timescale_privatelink_authorization.test]
}
`, subscriptionID, clientID, clientSecret, tenantID, subscriptionID)
}

func testAccPrivateLinkAzureConnectionConfig(name string) string {
	return fmt.Sprintf(`
resource "timescale_privatelink_connection" "test" {
  provider_connection_id = azurerm_private_endpoint.test.name
  cloud_provider         = "azure"
  region                 = "az-eastus"
  ip_address             = azurerm_private_endpoint.test.private_service_connection[0].private_ip_address
  name                   = %q
  timeout                = "5m"

  depends_on = [azurerm_private_endpoint.test]

  lifecycle {
    create_before_destroy = true
  }
}
`, name)
}

func testAccPrivateLinkAzureServiceConfig(attached bool) string {
	connectionIDLine := ""
	if attached {
		connectionIDLine = "\n  private_endpoint_connection_id = timescale_privatelink_connection.test.connection_id"
	}
	return fmt.Sprintf(`
resource "timescale_service" "test" {
  name        = "tf-acc-test-pl-azure-svc"
  milli_cpu   = 500
  memory_gb   = 2
  region_code = "az-eastus"
  timeouts = {
    create = "15m"
  }%s
}
`, connectionIDLine)
}

func testAccPrivateLinkAzureFullConfig(connectionName string, attached bool) string {
	return testAccPrivateLinkAzureBaseConfig() +
		testAccPrivateLinkAzureConnectionConfig(connectionName) +
		testAccPrivateLinkAzureServiceConfig(attached)
}
