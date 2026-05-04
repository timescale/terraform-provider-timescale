terraform {
  required_providers {
    timescale = {
      source  = "timescale/timescale"
      version = "~> 2.11"
    }
  }
}

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

# Create the target service to replicate data into
resource "timescale_service" "example" {
  name        = "postgres-connector-service"
  milli_cpu   = 1000
  memory_gb   = 4
  region_code = "us-east-1"
}

resource "timescale_connector_src_postgres" "example" {
  service_id        = timescale_service.example.id
  display_name      = "my-postgres-connector"
  name              = "my-postgres-source"
  connection_string = "postgresql://user:password@source-host:5432/mydb" # Source DB connection string

  table_sync_workers = 4
  enabled            = true

  tables = [
    {
      schema_name = "public"
      table_name  = "events"

      hypertable_spec = {
        primary_dimension = {
          column_name        = "created_at"
          partition_interval = "7d"
        }
      }
    }
  ]
}
