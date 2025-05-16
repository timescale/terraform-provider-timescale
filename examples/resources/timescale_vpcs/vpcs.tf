terraform {
  required_providers {
    timescale = {
      source  = "timescale/timescale"
      version = "~> 2.0"
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


resource "timescale_vpcs" "main" {
  cidr        = "10.10.0.0/16"
  name        = "test-vpc"
  region_code = "us-east-1"
}

# Requester's side of the peering connection.
resource "timescale_peering_connection" "peer" {
  peer_account_id  = "000000000000"
  peer_region_code = "eu-central-1"
  peer_vpc_id      = "vpc-00000000000000000"
  timescale_vpc_id = timescale_vpcs.main.id
}

