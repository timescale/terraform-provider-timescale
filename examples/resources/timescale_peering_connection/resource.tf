terraform {
  required_providers {
    timescale = {
      source  = "timescale/timescale"
      version = "~> 2.5"
    }
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
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

# Create a Timescale VPC
resource "timescale_vpcs" "ts-test" {
  cidr        = "10.0.0.0/24"
  name        = "tf-test"
  region_code = "us-east-1"
}

# If you have multiple regions, you'll need to use multiple `provider` configurations.
provider "aws" {
  region = "eu-central-1"
}

# ========================================
# VPC Peering Example (Cross-Region)
# ========================================

# Creating a test VPC in eu-central-1
resource "aws_vpc" "main" {
  cidr_block = "11.0.0.0/24"

  tags = {
    Name = "tf-test-vpc-peering"
  }
}

# Requester's side of the VPC peering connection (Timescale).
resource "timescale_peering_connection" "vpc_peer" {
  peer_account_id  = "000000000000"
  peer_region_code = "eu-central-1"
  peer_vpc_id      = aws_vpc.main.id
  peer_cidr_blocks = ["12.0.0.0/24", "12.1.0.0/24"] # Optional for VPC peering
  timescale_vpc_id = timescale_vpcs.ts-test.id
}

# Acceptor's side of the VPC peering connection (AWS).
resource "aws_vpc_peering_connection_accepter" "vpc_peer" {
  vpc_peering_connection_id = timescale_peering_connection.vpc_peer.accepter_provisioned_id
  auto_accept               = true
}

# ========================================
# Transit Gateway Peering Example
# ========================================

# Wait for VPC peering to be fully established before creating TGW peering
resource "time_sleep" "wait_for_vpc_peering" {
  depends_on = [aws_vpc_peering_connection_accepter.vpc_peer]

  create_duration = "120s"
}

# Create a test Transit Gateway in eu-central-1
resource "aws_ec2_transit_gateway" "tgw" {
  description = "TGW for Timescale peering"

  tags = {
    Name = "tf-test-tgw"
  }
}

# Create Transit Gateway peering with Timescale
resource "timescale_peering_connection" "tgw_peer" {
  peer_account_id  = "000000000000"
  peer_region_code = "eu-central-1"
  peer_tgw_id      = aws_ec2_transit_gateway.tgw.id
  peer_cidr_blocks = ["16.0.0.0/24", "16.1.0.0/24"] # Required for TGW peering.
  timescale_vpc_id = timescale_vpcs.ts-test.id

  # We need to wait for previous peering to be completed because we are using the same Timescale VPC for both peerings
  depends_on = [time_sleep.wait_for_vpc_peering]
}

# Wait for TGW peering attachment to propagate to AWS
resource "time_sleep" "wait_for_tgw_attachment" {
  depends_on = [timescale_peering_connection.tgw_peer]

  create_duration = "120s"
}

# Accept the Transit Gateway attachment
resource "aws_ec2_transit_gateway_peering_attachment_accepter" "tgw_peer" {
  transit_gateway_attachment_id = timescale_peering_connection.tgw_peer.accepter_provisioned_id

  depends_on = [time_sleep.wait_for_tgw_attachment]
}
