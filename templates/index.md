# Timescale Terraform Provider
The Terraform provider for [Timescale](https://www.timescale.com/cloud).

## Requirements
- [Terraform](https://www.terraform.io/downloads.html) >= 1.0

## Quick Start

### Authorization
When you log in to your [Timescale Account](https://console.cloud.timescale.com/), navigate to the `Project settings` page.
From here, you can create client credentials for programmatic usage. Click the `Create credentials` button to generate a new public/secret key pair.

### Project ID
The project ID can be found on the `Project settings` page.

Create a `main.tf` configuration file with the following content.
```hcl
terraform {
  required_providers {
    timescale = {
      source  = "timescale/timescale"
      version = "x.y.z"
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

resource "timescale_service" "test" {
  # name       = ""
  # milli_cpu  = 500
  # memory_gb  = 2
  # region_code = "us-east-1"
  # enable_ha_replica = false
  # timeouts = {
  #   create = "30m"
  # }
}
```

### VPC Peering

Since v1.9.0 it is possible to peer Timescale VPCs to AWS VPCs using terraform.

Below is a minimal working example:

```hcl
terraform {
  required_providers {
    timescale = {
      source  = "timescale/timescale"
      version = "~> 1.13.1"
    }
  }
}

provider "timescale" {
  project_id = var.ts_project_id
  access_key = var.ts_access_key
  secret_key = var.ts_secret_key
}

# If you have multiple regions, you’ll need to use multiple `provider` configurations.
provider "aws" {
  region = var.aws_region
}

variable "aws_account_id" {
  type = string
}

variable "aws_region" {
  type    = string
  default = "us-east-1"
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

variable "ts_region" {
  type    = string
  default = "us-east-1"
}

resource "timescale_vpcs" "main" {
  cidr        = "10.10.0.0/16"
  name        = "vpc_name"
  region_code = var.ts_region
}

resource "aws_vpc" "main" {
  cidr_block = "10.0.1.0/24"
}

# Requester's side of the peering connection.
resource "timescale_peering_connection" "peer" {
  peer_account_id  = var.aws_account_id
  peer_region_code = var.aws_region
  peer_vpc_id      = aws_vpc.main.id
  timescale_vpc_id = timescale_vpcs.main.id
}

# Accepter's side of the peering connection.
resource "aws_vpc_peering_connection_accepter" "peer" {
  vpc_peering_connection_id = timescale_peering_connection.peer.provisioned_id
  auto_accept               = true

  depends_on = [timescale_peering_connection.peer]
}
```

Note that this configuration may fail on first apply, as the value of
`timescale_peering_connection.peer.provisioned_id` (starting with `pcx-`) may
not be immediately available. This typically happens due to the asynchronous
nature of the VPC peering request and its acceptance process. In this case, a
second `terraform apply` can be run to ensure everything is applied.

## Supported Service Configurations
### Compute
- 500m CPU / 2 GB Memory
- 1000m CPU / 4 GB Memory
- 2000m CPU / 8 GB Memory
- 4000m CPU / 16 GB Memory
- 8000m CPU / 32 GB Memory
- 16000m CPU / 64 GB Memory
- 32000m CPU / 128 GB Memory

### Storage
Since June 2023, you no longer need to allocate a fixed storage volume or worry about managing your disk size, and you'll be billed only for the storage you actually use.
See more info in our [blogpost](https://www.timescale.com/blog/savings-unlocked-why-we-switched-to-a-pay-for-what-you-store-database-storage-model/)

## Supported Operations
✅ Create service <br />
✅ Rename service <br />
✅ Resize service <br />
✅ Pause/resume service <br />
✅ Delete service <br />
✅ Import service <br />
✅ Enable High Availability replicas <br />
✅ Enable read replicas <br />
✅ VPC peering <br />
✅ Connection pooling <br />

## Billing
Services are currently billed for hourly usage. If a service is running for less than an hour,
it will still be charged for the full hour of usage.
