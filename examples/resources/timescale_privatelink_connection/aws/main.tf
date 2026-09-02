# Minimal end-to-end AWS PrivateLink setup:
#
#   1. a small Tiger Cloud service
#   2. a VPC endpoint, claimed as a private link connection
#   3. a small EC2 instance to test from
#   4. a `terraform output` command that runs the test
#
# Tiger Cloud does not publish DNS for private link. The customer owns the
# records, so this example also creates a private hosted zone pointing the
# service hostname at the endpoint. That is what keeps sslmode=verify-full
# working: the client asks for the same hostname the public endpoint uses, and
# the existing certificate covers it.

terraform {
  required_version = ">= 1.3.0"

  required_providers {
    timescale = {
      source  = "timescale/timescale"
      version = "~> 2.0"
    }
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.0"
    }
  }
}

variable "ts_access_key" {
  description = "Tiger Cloud client credentials access key."
  type        = string
}

variable "ts_secret_key" {
  description = "Tiger Cloud client credentials secret key."
  type        = string
  sensitive   = true
}

variable "ts_project_id" {
  description = "Tiger Cloud project ID."
  type        = string
}

variable "region" {
  description = "AWS region. Must be a region where Tiger Cloud offers private link, and where the service lives."
  type        = string
  default     = "us-east-1"

  validation {
    # The Availability Zones Tiger Cloud serves PrivateLink from are fixed per
    # region, so a region absent from that map cannot be used here.
    condition = contains([
      "ap-northeast-1", "ap-south-1", "ap-southeast-1", "ap-southeast-2",
      "ca-central-1", "eu-central-1", "eu-central-2", "eu-west-1", "eu-west-2",
      "sa-east-1", "us-east-1", "us-east-2", "us-west-2",
    ], var.region)
    error_message = "Tiger Cloud does not offer PrivateLink in this region. See the privatelink_availability_zone_ids map in main.tf for the supported list."
  }
}

variable "vpc_cidr" {
  description = "CIDR of the test VPC. Change it if it overlaps an existing VPC in the account."
  type        = string
  default     = "10.42.0.0/16"
}

variable "name_prefix" {
  description = "Prefix for every resource name."
  type        = string
  default     = "tf-privatelink-example"
}

variable "ssh_public_key_path" {
  description = "Public key authorized on the test instance."
  type        = string
  default     = "~/.ssh/id_ed25519.pub"
}

provider "timescale" {
  access_key = var.ts_access_key
  secret_key = var.ts_secret_key
  project_id = var.ts_project_id
}

provider "aws" {
  region = var.region
}

# ============================================================================
# 1. Tiger Cloud service
# ============================================================================

resource "timescale_service" "example" {
  name        = var.name_prefix
  region_code = var.region
  milli_cpu   = 500
  memory_gb   = 2

  private_endpoint_connection_ids = [timescale_privatelink_connection.example.connection_id]

  timeouts = {
    create = "30m"
  }
}

# ============================================================================
# 2. VPC endpoint, claimed as a private link connection
# ============================================================================

# The endpoint service name to point the VPC endpoint at.
data "timescale_privatelink_region" "example" {
  region = var.region
}

# Tiger Cloud serves PrivateLink from exactly two Availability Zones per
# region. The endpoint must be created in both: one for the connection to work,
# the second so it survives the loss of a zone. An interface in any other zone
# cannot reach the service.
#
# These are Availability Zone *IDs*, not names. AWS maps names like us-east-1a
# to different physical zones in every account, so matching on the name would
# land the endpoint in the wrong place.
locals {
  privatelink_availability_zone_ids = {
    "ap-northeast-1" = ["apne1-az1", "apne1-az4"]
    "ap-south-1"     = ["aps1-az1", "aps1-az3"]
    "ap-southeast-1" = ["apse1-az1", "apse1-az2"]
    "ap-southeast-2" = ["apse2-az1", "apse2-az2"]
    "ca-central-1"   = ["cac1-az1", "cac1-az2"]
    "eu-central-1"   = ["euc1-az1", "euc1-az2"]
    "eu-central-2"   = ["euc2-az1", "euc2-az2"]
    "eu-west-1"      = ["euw1-az2", "euw1-az3"]
    "eu-west-2"      = ["euw2-az2", "euw2-az3"]
    "sa-east-1"      = ["sae1-az1", "sae1-az2"]
    "us-east-1"      = ["use1-az1", "use1-az6"]
    "us-east-2"      = ["use2-az1", "use2-az2"]
    "us-west-2"      = ["usw2-az3", "usw2-az4"]
  }

  availability_zone_ids = local.privatelink_availability_zone_ids[var.region]

  # Zone IDs are account-independent; zone names are not. Translate to the names
  # this account uses so each subnet lands in the intended physical zone.
  zone_id_to_name = zipmap(
    data.aws_availability_zones.available.zone_ids,
    data.aws_availability_zones.available.names,
  )

  # Index fixes each subnet's CIDR, so the pair stays stable.
  subnets = {
    for idx, az_id in local.availability_zone_ids : az_id => {
      zone_name  = local.zone_id_to_name[az_id]
      cidr_block = cidrsubnet(var.vpc_cidr, 8, idx + 1)
    }
  }

  # The test instance goes in the first zone. Not every instance type is offered
  # in every zone, so it must be placed explicitly rather than left to AWS.
  primary_availability_zone_id = local.availability_zone_ids[0]
}

data "aws_availability_zones" "available" {
  state = "available"
}

resource "aws_vpc" "example" {
  cidr_block           = var.vpc_cidr
  enable_dns_support   = true
  enable_dns_hostnames = true

  tags = { Name = var.name_prefix }
}

resource "aws_subnet" "example" {
  for_each = local.subnets

  vpc_id            = aws_vpc.example.id
  cidr_block        = each.value.cidr_block
  availability_zone = each.value.zone_name

  map_public_ip_on_launch = true

  tags = {
    Name = "${var.name_prefix}-${each.key}"
    AzId = each.key
  }
}

resource "aws_internet_gateway" "example" {
  vpc_id = aws_vpc.example.id

  tags = { Name = var.name_prefix }
}

resource "aws_route_table" "example" {
  vpc_id = aws_vpc.example.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.example.id
  }

  tags = { Name = var.name_prefix }
}

resource "aws_route_table_association" "example" {
  for_each = aws_subnet.example

  subnet_id      = each.value.id
  route_table_id = aws_route_table.example.id
}

# A binding takes a reserved port by role: 5432 primary, 5433 replica,
# 6432 pooler.
locals {
  endpoint_ports = [5432, 5433, 6432]
}

resource "aws_security_group" "endpoint" {
  name        = "${var.name_prefix}-endpoint"
  description = "Allow Postgres from the test instance to the private link endpoint"
  vpc_id      = aws_vpc.example.id

  dynamic "ingress" {
    for_each = local.endpoint_ports

    content {
      description     = "Postgres over private link (port ${ingress.value})"
      from_port       = ingress.value
      to_port         = ingress.value
      protocol        = "tcp"
      security_groups = [aws_security_group.instance.id]
    }
  }

  tags = { Name = "${var.name_prefix}-endpoint" }
}

# One interface per Availability Zone, so the endpoint survives losing a zone.
resource "aws_vpc_endpoint" "example" {
  vpc_id             = aws_vpc.example.id
  service_name       = data.timescale_privatelink_region.example.service_name
  vpc_endpoint_type  = "Interface"
  subnet_ids         = [for s in aws_subnet.example : s.id]
  security_group_ids = [aws_security_group.endpoint.id]

  tags = { Name = var.name_prefix }
}

# Tiger Cloud's endpoint service accepts every connection but leaves it
# unowned. Claiming it assigns it to this project. On AWS the claim identifier
# is the VPC endpoint ID.
resource "timescale_privatelink_connection" "example" {
  claim_identifier = aws_vpc_endpoint.example.id
  name             = var.name_prefix
}

# ============================================================================
# Private DNS: the customer's own zone, pointing the service hostname at the
# endpoint. An alias A record to the endpoint's regional DNS name, which AWS
# balances across every healthy ENI, so no IP is hardcoded.
# ============================================================================

locals {
  hostname_labels = split(".", timescale_service.example.hostname)

  # Drop the leading service ID to get the zone, so every service in this
  # project resolves under one zone:
  #
  #   hostname = "abc123xyz.myproject.tsdb.cloud.timescale.com"  (the A record)
  #   zone     =           "myproject.tsdb.cloud.timescale.com"  (the zone)
  #
  # A wildcard record is not possible: Tiger Cloud issues per-service
  # certificates, so only exact hostnames validate under verify-full. Add one
  # record per service.
  service_domain = join(".", slice(local.hostname_labels, 1, length(local.hostname_labels)))
}

resource "aws_route53_zone" "example" {
  name    = local.service_domain
  comment = "Tiger Cloud private link for project ${var.ts_project_id}"

  vpc {
    vpc_id = aws_vpc.example.id
  }

  tags = { Name = var.name_prefix }
}

resource "aws_route53_record" "example" {
  zone_id = aws_route53_zone.example.zone_id
  name    = timescale_service.example.hostname
  type    = "A"

  alias {
    name                   = aws_vpc_endpoint.example.dns_entry[0].dns_name
    zone_id                = aws_vpc_endpoint.example.dns_entry[0].hosted_zone_id
    evaluate_target_health = false
  }
}

# ============================================================================
# 3. Test instance
# ============================================================================

data "aws_ssm_parameter" "al2023" {
  name = "/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-default-x86_64"
}

resource "aws_security_group" "instance" {
  name        = "${var.name_prefix}-instance"
  description = "Test instance: inbound SSH"
  vpc_id      = aws_vpc.example.id

  ingress {
    description = "SSH"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = { Name = "${var.name_prefix}-instance" }
}

resource "aws_key_pair" "example" {
  key_name   = var.name_prefix
  public_key = file(pathexpand(var.ssh_public_key_path))
}

resource "aws_instance" "example" {
  ami                    = data.aws_ssm_parameter.al2023.value
  instance_type          = "t3.micro"
  subnet_id              = aws_subnet.example[local.primary_availability_zone_id].id
  vpc_security_group_ids = [aws_security_group.instance.id]
  key_name               = aws_key_pair.example.key_name

  user_data = <<-EOT
    #!/bin/bash
    dnf install -y postgresql16 bind-utils
  EOT

  tags = { Name = var.name_prefix }
}

# ============================================================================
# 4. Test the connection
# ============================================================================

# sslrootcert=system trusts the OS CA bundle. Tiger Cloud certificates come from
# a public CA, so verify-full validates without downloading a root certificate —
# and it validates the same hostname the public endpoint uses, which is the whole
# point of pointing your own DNS at the endpoint.
output "test_command" {
  description = "SSHes into the test instance and runs SELECT 1 over private link."
  value = join(" ", [
    "ssh -o StrictHostKeyChecking=no",
    "-i ${trimsuffix(pathexpand(var.ssh_public_key_path), ".pub")}",
    "ec2-user@${aws_instance.example.public_ip}",
    "\"PGPASSWORD='${timescale_service.example.password}'",
    "psql",
    "'host=${timescale_service.example.hostname}",
    "port=${timescale_service.example.port}",
    "user=${timescale_service.example.username}",
    "dbname=tsdb",
    "sslmode=verify-full",
    "sslrootcert=system'",
    "-c 'SELECT 1'\"",
  ])
  sensitive = true
}

output "dns_check_command" {
  description = "Confirms the service hostname resolves to the endpoint inside the VPC."
  value       = "ssh ec2-user@${aws_instance.example.public_ip} 'dig +short ${timescale_service.example.hostname}'"
}

output "service_hostname" {
  description = "Hostname to connect to. Same as the public endpoint; only the resolution and port differ."
  value       = timescale_service.example.hostname
}

output "service_port" {
  description = "Private link port for this service, allocated per binding."
  value       = timescale_service.example.port
}

output "connection_id" {
  description = "The claimed private link connection."
  value       = timescale_privatelink_connection.example.connection_id
}

output "availability_zones" {
  description = "Availability Zone ID => the name it maps to in this account, for the zones the endpoint spans."
  value       = { for az_id, s in local.subnets : az_id => s.zone_name }
}

output "endpoint_network_interfaces" {
  description = "One endpoint network interface per Availability Zone. Two means both zones can serve traffic; one means there is no failover."
  value       = aws_vpc_endpoint.example.network_interface_ids
}

output "endpoint_regional_dns_name" {
  description = "The endpoint's regional DNS name, which AWS resolves across every healthy interface. This is the alias target for the private DNS record."
  value       = aws_vpc_endpoint.example.dns_entry[0].dns_name
}
