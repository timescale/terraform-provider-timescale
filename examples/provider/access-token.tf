# Authenticate using an access token. It is longlasting.
variable "ts_access_token" {
  type = string
}

variable "ts_project_id" {
  type = string
}

provider "timescale" {
  project_id   = var.ts_project_id
  access_token = var.ts_access_token
}