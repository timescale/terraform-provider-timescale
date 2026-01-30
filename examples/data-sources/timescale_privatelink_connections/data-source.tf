terraform {
  required_providers {
    timescale = {
      source  = "timescale/timescale"
      version = "~> 2.8"
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

# List all private link connections
data "timescale_privatelink_connections" "all" {}

# List private link connections filtered by region
data "timescale_privatelink_connections" "eastus2" {
  region = "az-eastus2"
}

# Example: Use the first approved connection to attach a service
resource "timescale_service" "with_private_link" {
  name        = "private-link-service"
  milli_cpu   = 1000
  memory_gb   = 4
  region_code = "az-eastus2"

  # Use connection_id from the data source
  private_endpoint_connection_id = data.timescale_privatelink_connections.eastus2.connections[0].connection_id
}

output "all_connections" {
  description = "All private link connections"
  value       = data.timescale_privatelink_connections.all.connections
}

output "eastus2_connection_ids" {
  description = "Connection IDs for eastus2 region"
  value       = [for c in data.timescale_privatelink_connections.eastus2.connections : c.connection_id]
}
