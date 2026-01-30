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


resource "timescale_service" "test" {
  name        = "test"
  milli_cpu   = 1000
  memory_gb   = 4
  region_code = "us-east-1"
}

# Read replica
resource "timescale_service" "read_replica" {
  read_replica_source = timescale_service.test.id
}

# Service with Azure Private Link
# Prerequisites:
# 1. Create a Private Link authorization in the Timescale UI for your Azure subscription
# 2. Create a Private Endpoint in Azure pointing to the Timescale Private Link Service
# 3. Get the private_endpoint_connection_id from the Timescale UI
resource "timescale_service" "with_private_link" {
  name                            = "private-link-service"
  milli_cpu                       = 1000
  memory_gb                       = 4
  region_code                     = "az-eastus2"
  private_endpoint_connection_id  = "your-private-endpoint-connection-id"
}
