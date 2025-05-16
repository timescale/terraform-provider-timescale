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


resource "timescale_service" "test" {
  name        = ""
  milli_cpu   = 1000
  memory_gb   = 4
  region_code = ""
}

# Read replica
resource "timescale_service" "read_replica" {
  read_replica_source = timescale_service.test.id
}
