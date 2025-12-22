terraform {
  required_providers {
    timescale = {
      source  = "timescale/timescale"
      version = "~> 2.7"
    }
  }
}

# Authenticate using client credentials.
# They are issued through the Timescale UI.
# When required, they will exchange for a short-lived JWT to do the calls.
provider "timescale" {
  project_id = var.ts_project_id
  access_key = var.ts_access_key
  secret_key = var.ts_secret_key
}

variable "ts_project_id" {
  type = string
}

variable "ts_access_key" {
  type = string
}

variable "ts_secret_key" {
  type      = string
  sensitive = true
}

resource "timescale_vpcs" "ts-test" {
  cidr        = "10.0.0.0/24"
  name        = "tf-test"
  region_code = "us-east-1"
}
