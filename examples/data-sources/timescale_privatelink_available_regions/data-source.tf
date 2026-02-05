terraform {
  required_version = ">= 1.3.0"

  required_providers {
    timescale = {
      source  = "timescale/timescale"
      version = "~> 2.8"
    }
  }
}

variable "ts_access_key" {
  type        = string
  description = "Timescale access key"
}

variable "ts_secret_key" {
  type        = string
  sensitive   = true
  description = "Timescale secret key"
}

variable "ts_project_id" {
  type        = string
  description = "Timescale project ID"
}

provider "timescale" {
  access_key = var.ts_access_key
  secret_key = var.ts_secret_key
  project_id = var.ts_project_id
}

# List all regions where Azure Private Link is available
data "timescale_privatelink_available_regions" "all" {}

output "available_regions" {
  description = "All regions where Azure Private Link is available"
  value = [
    for r in data.timescale_privatelink_available_regions.all.regions : {
      region = r.region
      alias  = r.private_link_service_alias
    }
  ]
}

# Example: Get the alias for a specific region
output "eastus2_alias" {
  description = "Private Link Service alias for az-eastus2"
  value = [
    for r in data.timescale_privatelink_available_regions.all.regions : r.private_link_service_alias
    if r.region == "az-eastus2"
  ]
}
