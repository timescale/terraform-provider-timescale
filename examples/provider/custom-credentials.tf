# Authenticate using client credentials. 
# They must be issued on the console for now.
# When required, they will exchange for a short-lived JWT to do the calls.
variable "ts_project_id" {
  type = string
}

variable "ts_access_key" {
  type = string
}

variable "ts_secret_key" {
  type = string
}

provider "timescale" {
  project_id = var.ts_project_id
  access_key = var.ts_access_key
  secret_key = var.ts_secret_key
}