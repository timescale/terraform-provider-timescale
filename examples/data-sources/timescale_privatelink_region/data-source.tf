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

variable "timescale_region" {
  type        = string
  description = "Timescale region for the service (e.g., us-east-1, az-eastus)"
  default     = "us-east-1"
}

provider "timescale" {
  access_key = var.ts_access_key
  secret_key = var.ts_secret_key
  project_id = var.ts_project_id
}

# Look up Private Link availability for a single region. If the region is not
# available, terraform plan fails with an error listing the available regions.
data "timescale_privatelink_region" "selected" {
  region = var.timescale_region
}

output "service_name" {
  description = "Service name to use when creating the cloud-side endpoint (AWS VPC Endpoint Service name or Azure Private Link Service alias)"
  value       = data.timescale_privatelink_region.selected.service_name
}

output "cloud_provider" {
  description = "Cloud provider for the selected region"
  value       = data.timescale_privatelink_region.selected.cloud_provider
}
