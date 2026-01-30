terraform {
  required_providers {
    timescale = {
      source  = "timescale/timescale"
      version = "~> 2.8"
    }
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.0"
    }
  }
}

variable "ts_access_key" {
  type = string
}

variable "ts_secret_key" {
  type      = string
  sensitive = true
}

variable "ts_project_id" {
  type = string
}

provider "timescale" {
  access_key = var.ts_access_key
  secret_key = var.ts_secret_key
  project_id = var.ts_project_id
}

provider "azurerm" {
  features {}
}

# =============================================================================
# Example 1: List all private link connections
# =============================================================================
data "timescale_privatelink_connections" "all" {}

output "all_connections" {
  description = "All private link connections"
  value       = data.timescale_privatelink_connections.all.connections
}

# =============================================================================
# Example 2: Filter by region
# =============================================================================
data "timescale_privatelink_connections" "eastus" {
  region = "az-eastus"
}

# =============================================================================
# Example 3: Match by Azure connection name (recommended for automation)
# =============================================================================
# When you create a Private Endpoint in Azure, you specify a connection name
# in the private_service_connection block. Azure appends a resource GUID to
# this name (e.g., "my-pe-connection" becomes "my-pe-connection.<guid>").
#
# The azure_connection_name filter matches your connection by this name,
# without requiring you to know the GUID.

# First, create the Azure Private Endpoint
resource "azurerm_private_endpoint" "timescale" {
  name                = "my-timescale-pe"
  location            = "eastus"
  resource_group_name = "my-resource-group"
  subnet_id           = "/subscriptions/.../subnets/default"

  private_service_connection {
    name                              = "my-pe-connection" # This is the name to use in the filter
    private_connection_resource_alias = "timescaledb-...-pls.azure.privatelinkservice"
    is_manual_connection              = true
    request_message                   = var.ts_project_id
  }
}

# Then, look up the connection using the same name you specified above
data "timescale_privatelink_connections" "my_connection" {
  azure_connection_name = azurerm_private_endpoint.timescale.private_service_connection[0].name

  # The data source will find the connection where azure_connection_name
  # starts with "my-pe-connection." (your name + dot + GUID)
}

# Finally, create a service attached to this private link connection
resource "timescale_service" "with_private_link" {
  name        = "private-link-service"
  milli_cpu   = 1000
  memory_gb   = 4
  region_code = "az-eastus"

  # Use connection_id from the matched connection
  private_endpoint_connection_id = data.timescale_privatelink_connections.my_connection.connections[0].connection_id
}

output "service_hostname" {
  value = timescale_service.with_private_link.hostname
}
