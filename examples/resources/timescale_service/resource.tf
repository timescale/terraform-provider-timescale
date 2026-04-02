terraform {
  required_providers {
    timescale = {
      source  = "timescale/timescale"
      version = "~> 2.10"
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

# Read replica (single node, default)
resource "timescale_service" "read_replica" {
  read_replica_source = timescale_service.test.id
}

# Read replica with multiple nodes (1-10)
resource "timescale_service" "read_replica_multi" {
  name                = "multi-node-replica"
  read_replica_source = timescale_service.test.id
  read_replica_nodes  = 3
}

# Service with write-only password (Terraform 1.11+)
# The password is sent to the API but never stored in Terraform state.
# Increment password_wo_version to trigger a password change.
variable "db_password" {
  type      = string
  sensitive = true
}

resource "timescale_service" "secure" {
  name                = "secure-service"
  milli_cpu           = 1000
  memory_gb           = 4
  region_code         = "us-east-1"
  password_wo         = var.db_password
  password_wo_version = 1
}
