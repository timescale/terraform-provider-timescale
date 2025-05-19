terraform {
  required_providers {
    timescale = {
      source  = "timescale/timescale"
      version = "~> 2.0"
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

# If you have multiple regions, you’ll need to use multiple `provider` configurations.
provider "aws" {
  region = "eu-central-1"
}

# Creating a test VPC. Change to your VPC if you already have one in your AWS account.
resource "aws_vpc" "main" {
  cidr_block = "11.0.0.0/24"
}

# Requester's side of the peering connection (Timescale).
resource "timescale_peering_connection" "peer" {
  peer_account_id  = "000000000000"
  peer_region_code = "eu-central-1"
  peer_vpc_id      = aws_vpc.main.id
  timescale_vpc_id = timescale_vpcs.ts-test.id
}

# Acceptor's side of the peering connection (AWS).
resource "aws_vpc_peering_connection_accepter" "peer" {
  vpc_peering_connection_id = timescale_peering_connection.peer.provisioned_id
  auto_accept               = true
  depends_on                = [timescale_peering_connection.peer]
}
