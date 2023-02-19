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

resource "timescale_service" "test" {
  name = "kim"
}
