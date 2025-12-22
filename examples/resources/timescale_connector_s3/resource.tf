terraform {
  required_providers {
    timescale = {
      source  = "timescale/timescale"
      version = "~> 2.7"
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

# Create a service to attach the S3 connector to
resource "timescale_service" "example" {
  name        = "s3-connector-service"
  milli_cpu   = 1000
  memory_gb   = 4
  region_code = "us-east-1"
}

resource "timescale_connector_s3" "example" {
  service_id = timescale_service.example.id
  name       = "my-s3-connector"
  bucket     = "my-data-bucket"
  pattern    = "data/*.csv"

  credentials = {
    type = "Public"
  }

  definition = {
    type = "CSV"
    csv = {
      skip_header         = true
      auto_column_mapping = true
    }
  }

  table_identifier = {
    schema_name = "public"
    table_name  = "sensor_data"
  }

  enabled = true
}
