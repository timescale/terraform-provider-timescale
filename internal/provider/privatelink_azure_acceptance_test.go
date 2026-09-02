package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccPrivateLinkAzurePreCheck(t *testing.T) {
	testAccPreCheck(t)
	for _, key := range []string{"ARM_SUBSCRIPTION_ID", "ARM_CLIENT_ID", "ARM_CLIENT_SECRET", "ARM_TENANT_ID"} {
		if v, ok := os.LookupEnv(key); !ok || v == "" {
			t.Skipf("%s not set, skipping Azure Private Link test", key)
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
				VersionConstraint: ">= 4.0",
			},
			"azapi": {
				Source:            "Azure/azapi",
				VersionConstraint: ">= 2.0",
			},
		},
		PreCheck: func() { testAccPrivateLinkAzurePreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccPrivateLinkAzureFullConfig("test-acc-managed", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(connectionName, "connection_id"),
					resource.TestCheckResourceAttrSet(connectionName, "state"),
					resource.TestCheckResourceAttrSet(connectionName, "claim_identifier"),
					resource.TestCheckResourceAttr(connectionName, "name", "test-acc-managed"),
					resource.TestCheckResourceAttr(connectionName, "cloud_provider", "azure"),
					resource.TestCheckResourceAttr(connectionName, "region", "az-eastus2"),
					resource.TestCheckResourceAttrSet(serviceName, "id"),
					resource.TestCheckResourceAttr(serviceName, "private_endpoint_connection_ids.#", "1"),
				),
			},
			{
				Config: testAccPrivateLinkAzureFullConfig("test-acc-renamed", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(connectionName, "name", "test-acc-renamed"),
				),
			},
			{
				Config: testAccPrivateLinkAzureFullConfig("test-acc-renamed", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(serviceName, "id"),
					resource.TestCheckResourceAttr(serviceName, "private_endpoint_connection_ids.#", "0"),
				),
			},
			{
				Config: testAccPrivateLinkAzureFullConfig("test-acc-renamed", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(serviceName, "private_endpoint_connection_ids.#", "1"),
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

provider "azapi" {
  subscription_id = %q
  client_id       = %q
  client_secret   = %q
  tenant_id       = %q
}

data "timescale_privatelink_region" "test" {
  region = "az-eastus2"
}

resource "azurerm_resource_group" "test" {
  name     = "tf-acc-test-pl-rg"
  location = "eastus2"
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

resource "azurerm_private_endpoint" "test" {
  name                = "tf-acc-test-pl-pe"
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
  subnet_id           = azurerm_subnet.test.id

  private_service_connection {
    name                              = "tf-acc-test-pl-psc"
    private_connection_resource_alias = data.timescale_privatelink_region.test.service_name
    is_manual_connection              = true
    request_message                   = "terraform acceptance test"
  }
}

# azurerm does not expose resourceGuid (hashicorp/terraform-provider-azurerm#17011),
# and it is the Azure claim identifier.
data "azapi_resource" "test_pe" {
  type                   = "Microsoft.Network/privateEndpoints@2024-05-01"
  resource_id            = azurerm_private_endpoint.test.id
  response_export_values = ["properties.resourceGuid"]
}
`, subscriptionID, clientID, clientSecret, tenantID, subscriptionID, clientID, clientSecret, tenantID)
}

func testAccPrivateLinkAzureConnectionConfig(name string) string {
	return fmt.Sprintf(`
resource "timescale_privatelink_connection" "test" {
  claim_identifier  = data.azapi_resource.test_pe.output.properties.resourceGuid
  name              = %q
  reject_on_destroy = true

  timeouts = {
    create = "10m"
  }
}
`, name)
}

func testAccPrivateLinkAzureServiceConfig(attached bool) string {
	connectionIDLine := ""
	if attached {
		connectionIDLine = "\n  private_endpoint_connection_ids = [timescale_privatelink_connection.test.connection_id]"
	}
	return fmt.Sprintf(`
resource "timescale_service" "test" {
  name        = "tf-acc-test-pl-azure-svc"
  milli_cpu   = 500
  memory_gb   = 2
  region_code = "az-eastus2"
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
