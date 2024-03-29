terraform {
  required_providers {
    timescale = {
      source  = "registry.terraform.io/providers/timescale"
      version = "~> 1.0"
    }
  }
}

variable "ts_access_token" {
  type = string
}

variable "ts_project_id" {
  type = string
}

provider "timescale" {
  access_token = var.ts_access_token
  project_id   = var.ts_project_id
}

resource "timescale_vpcs" "new_vpc" {
  name        = "test-vpc"
  cidr        = "10.0.0.0/19"
  region_code = "us-east-1"
}