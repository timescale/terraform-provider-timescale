terraform {
  required_providers {
    timescale = {
      source  = "timescale/timescale"
      version = "~> 2.2"
    }
  }
}

# Authenticate using client credentials.
# They are issued through the Timescale UI.
# When required, they will exchange for a short-lived JWT to do the calls.
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

resource "timescale_metric_exporter" "my_datadog_exporter" {
  name   = "Datadog Exporter from TF"
  region = "us-east-1"

  datadog = {
    api_key = "your_datadog_api_key_here"
    site    = "datadoghq.com" # or "datadoghq.eu", etc.
  }
}

resource "timescale_metric_exporter" "my_prometheus_exporter" {
  name   = "Prometheus Exporter from TF"
  region = "us-east-1"

  prometheus = {
    username = "prom_user"
    password = "a_very_secure_password"
  }
}

resource "timescale_metric_exporter" "my_cloudwatch_exporter_with_role" {
  name   = "CloudWatch Exporter via IAM Role from TF"
  region = "us-east-1"

  cloudwatch = {
    region          = "us-east-1"
    role_arn        = "arn:aws:iam::123456789012:role/MyMetricsExporterRole"
    log_group_name  = "/myapplication/metrics"
    log_stream_name = "exporter-stream-role"
    namespace       = "MyApplication/CustomMetrics"
  }
}

resource "timescale_metric_exporter" "my_cloudwatch_exporter_with_keys" {
  name   = "CloudWatch Exporter via Static Keys"
  region = "us-east-1"

  cloudwatch = {
    region          = "us-east-1"
    access_key      = "your_access_key"
    secret_key      = "your_secret_key"
    log_group_name  = "/anotherapplication/metrics"
    log_stream_name = "exporter-stream-keys"
    namespace       = "AnotherApplication/CustomMetrics"
  }
}