terraform {
  required_providers {
    timescale = {
      source  = "registry.terraform.io/providers/timescale"
      version = "~> 1.0"
    }
  }
}

variable "ts_access_key" {
  type = string
}

variable "ts_secret_key" {
  type = string
}

variable "ts_project_id" {
  type = string
}

provider "timescale" {
  access_key = var.ts_access_key
  secret_key = var.ts_secret_key
  project_id = var.ts_project_id
}

data "timescale_products" "products" {
}

output "products_list" {
  value = data.timescale_products.products
}
