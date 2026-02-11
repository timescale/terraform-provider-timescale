terraform {
  required_version = ">= 1.3.0"

  required_providers {
    timescale = {
      source  = "timescale/timescale"
      version = "~> 2.8"
    }
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.0"
    }
  }
}

# =============================================================================
# Variables
# =============================================================================

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

variable "aws_region" {
  type        = string
  description = "AWS region for infrastructure"
  default     = "us-east-1"
}

variable "timescale_region" {
  type        = string
  description = "Timescale region for the service (e.g., us-east-1)"
  default     = "us-east-1"
}

variable "resource_prefix" {
  type        = string
  description = "Prefix for all resource names"
  default     = "tspl-demo"
}

# =============================================================================
# Providers
# =============================================================================

provider "timescale" {
  access_key = var.ts_access_key
  secret_key = var.ts_secret_key
  project_id = var.ts_project_id
}

provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Environment = "Development"
      Project     = "Timescale Private Link Connection"
      Owner       = "vperez"
      CreatedBy   = "Terraform"
    }
  }
}

# =============================================================================
# Data Sources
# =============================================================================

data "aws_caller_identity" "current" {}

data "timescale_privatelink_available_regions" "all" {}

locals {
  vpc_endpoint_service_name = data.timescale_privatelink_available_regions.all.regions[var.timescale_region].service_name
}

# =============================================================================
# AWS Infrastructure
# =============================================================================

resource "aws_vpc" "main" {
  cidr_block           = "10.0.0.0/16"
  enable_dns_support   = true
  enable_dns_hostnames = true

  tags = {
    Name = "${var.resource_prefix}-vpc"
  }
}

resource "aws_subnet" "endpoint" {
  vpc_id            = aws_vpc.main.id
  cidr_block        = "10.0.1.0/24"
  availability_zone = "${var.aws_region}a"

  tags = {
    Name = "${var.resource_prefix}-endpoint-subnet"
  }
}

resource "aws_subnet" "vm" {
  vpc_id            = aws_vpc.main.id
  cidr_block        = "10.0.2.0/24"
  availability_zone = "${var.aws_region}a"

  tags = {
    Name = "${var.resource_prefix}-vm-subnet"
  }
}

resource "aws_internet_gateway" "main" {
  vpc_id = aws_vpc.main.id

  tags = {
    Name = "${var.resource_prefix}-igw"
  }
}

resource "aws_route_table" "public" {
  vpc_id = aws_vpc.main.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.main.id
  }

  tags = {
    Name = "${var.resource_prefix}-public-rt"
  }
}

resource "aws_route_table_association" "vm" {
  subnet_id      = aws_subnet.vm.id
  route_table_id = aws_route_table.public.id
}

resource "aws_security_group" "endpoint" {
  name_prefix = "${var.resource_prefix}-vpce-"
  vpc_id      = aws_vpc.main.id

  ingress {
    from_port   = 5432
    to_port     = 5432
    protocol    = "tcp"
    cidr_blocks = [aws_vpc.main.cidr_block]
  }

  tags = {
    Name = "${var.resource_prefix}-vpce-sg"
  }
}

resource "aws_security_group" "vm" {
  name_prefix = "${var.resource_prefix}-vm-"
  vpc_id      = aws_vpc.main.id

  ingress {
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

  tags = {
    Name = "${var.resource_prefix}-vm-sg"
  }
}

# =============================================================================
# Timescale Private Link Authorization
# =============================================================================

resource "timescale_privatelink_authorization" "main" {
  principal_id   = data.aws_caller_identity.current.account_id
  cloud_provider = "AWS"
  name           = "Terraform managed - ${var.resource_prefix}"
}

# =============================================================================
# AWS VPC Endpoint
# =============================================================================

resource "aws_vpc_endpoint" "timescale" {
  vpc_id              = aws_vpc.main.id
  service_name        = local.vpc_endpoint_service_name
  vpc_endpoint_type   = "Interface"
  subnet_ids          = [aws_subnet.endpoint.id]
  security_group_ids  = [aws_security_group.endpoint.id]
  private_dns_enabled = false

  tags = {
    Name = "${var.resource_prefix}-vpce"
  }

  depends_on = [timescale_privatelink_authorization.main]
}

# =============================================================================
# Look up the VPC Endpoint's private IP
# =============================================================================

data "aws_network_interface" "endpoint" {
  id = one(aws_vpc_endpoint.timescale.network_interface_ids)
}

# =============================================================================
# Timescale Private Link Connection
# =============================================================================

resource "timescale_privatelink_connection" "main" {
  provider_connection_id = aws_vpc_endpoint.timescale.id
  cloud_provider         = "AWS"
  region                 = var.timescale_region
  ip_address             = data.aws_network_interface.endpoint.private_ip
  name                   = "Managed by Terraform"

  depends_on = [aws_vpc_endpoint.timescale]

  lifecycle {
    create_before_destroy = true
  }
}

# =============================================================================
# Timescale Service with Private Link
# =============================================================================

resource "timescale_service" "main" {
  name        = "${var.resource_prefix}-db"
  milli_cpu   = 500
  memory_gb   = 2
  region_code = var.timescale_region

  private_endpoint_connection_id = timescale_privatelink_connection.main.connection_id
}

# =============================================================================
# EC2 Instance for testing connectivity
# =============================================================================

data "aws_ami" "ubuntu" {
  most_recent = true
  owners      = ["099720109477"] # Canonical

  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-amd64-server-*"]
  }
}

resource "aws_key_pair" "vm" {
  key_name   = "${var.resource_prefix}-key"
  public_key = file("~/.ssh/id_rsa.pub")
}

resource "aws_instance" "vm" {
  ami                         = data.aws_ami.ubuntu.id
  instance_type               = "t3.micro"
  subnet_id                   = aws_subnet.vm.id
  vpc_security_group_ids      = [aws_security_group.vm.id]
  associate_public_ip_address = true
  key_name                    = aws_key_pair.vm.key_name

  user_data = base64encode(<<-EOF
    #!/bin/bash
    set -e
    export DEBIAN_FRONTEND=noninteractive
    apt-get update
    apt-get install -y postgresql-client netcat-openbsd curl
  EOF
  )

  tags = {
    Name = "${var.resource_prefix}-vm"
  }
}

# =============================================================================
# Outputs
# =============================================================================

output "vpc_endpoint_id" {
  description = "AWS VPC Endpoint ID"
  value       = aws_vpc_endpoint.timescale.id
}

output "vm_public_ip" {
  description = "Public IP of the test VM for SSH access"
  value       = aws_instance.vm.public_ip
}

output "vm_ssh_command" {
  description = "SSH command to connect to VM"
  value       = "ssh ubuntu@${aws_instance.vm.public_ip}"
}

output "timescale_hostname" {
  description = "Timescale service hostname"
  value       = timescale_service.main.hostname
}

output "timescale_port" {
  description = "Timescale service port"
  value       = timescale_service.main.port
}

output "private_link_connection_id" {
  description = "Connection ID for use with timescale_service"
  value       = timescale_privatelink_connection.main.connection_id
}

output "private_link_connection_state" {
  description = "State of the Private Link connection"
  value       = timescale_privatelink_connection.main.state
}

output "vpc_endpoint_service_name" {
  description = "The VPC Endpoint Service name used"
  value       = local.vpc_endpoint_service_name
}

output "private_endpoint_ip" {
  description = "Private IP of the VPC Endpoint ENI"
  value       = data.aws_network_interface.endpoint.private_ip
}

output "connection_test_command" {
  description = "psql command to test from VM using private IP (run after SSH)"
  value       = "PGPASSWORD='${timescale_service.main.password}' psql -h ${data.aws_network_interface.endpoint.private_ip} -p ${timescale_service.main.port} -U ${timescale_service.main.username} -d tsdb"
  sensitive   = true
}

output "ssh_select_1" {
  description = "SSH command to execute SELECT 1 on the database via private link"
  value       = "ssh ubuntu@${aws_instance.vm.public_ip} \"PGPASSWORD='${timescale_service.main.password}' psql -h ${timescale_service.main.hostname} -p ${timescale_service.main.port} -U ${timescale_service.main.username} -d tsdb -c 'SELECT 1'\""
  sensitive   = true
}
