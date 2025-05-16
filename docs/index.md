# Timescale Terraform Provider
The Terraform provider for [Timescale](https://www.timescale.com/cloud).

## Requirements
- [Terraform](https://www.terraform.io/downloads.html) >= 1.0

## Quick Start

### Authorization
When you log in to your [Timescale Account](https://console.cloud.timescale.com/), click on your project name on the upper left-hand side of the page and go to the `Project settings` page.
From here, you can create client credentials for programmatic usage. Click the `Create credentials` button to generate a new public/secret key pair.

Find more information on creating Client Credentials in the [Timescale docs](https://docs.timescale.com/use-timescale/latest/security/client-credentials/#creating-client-credentials).

### Project ID

To view the project ID, click on your project name on the upper left-hand side of the page.

### Example files and usage

#### Service with HA replica and pooler

> [!NOTE]  
> The example file creates:
>  * A single instance called `tf-test` that contains:
     >    * 0.5 CPUs
     >    * 2GB of RAM
     >    * the region set to `us-west-2`
>    * an HA replica
>    * the connection pooler enabled
>  * Outputs to display the connection info for:
     >    * the primary hostname and port
     >    * the ha-replica hostname and port
     >    * the pooler hostname and port

Create a `main.tf` configuration file with the following content.
```hcl
terraform {
  required_providers {
    timescale = {
      source  = "timescale/timescale"
      version = "~> 2.0"
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
     name                      = "tf-test"
     milli_cpu                 = 500
     memory_gb                 = 2
     region_code               = "us-west-2"
     connection_pooler_enabled = true
     enable_ha_replica         = true
}

## host connection info
output "host_addr" {
     value       = timescale_service.tf-test.hostname
     description = "Service Host Address"
}

output "host_port" {
     value       = timescale_service.tf-test.port
     description = "Service Host port"
}

## ha-replica connection info
output "replica_addr" {
     value       = timescale_service.tf-test.replica_hostname
     description = "Service Replica Host Address"
}

output "replica_port" {
     value       = timescale_service.tf-test.replica_port
     description = "Service Replica Host port"
}

## pooler connection info
output "pooler_addr" {
     value       = timescale_service.tf-test.pooler_hostname
     description = "Service Pooler Host Address"
}

output "pooler_port" {
     value       = timescale_service.tf-test.pooler_port
     description = "Service Pooler Host port"
}
```

and define the `secret.tfvars` file:

```hcl
ts_project_id="WWWWWWWWWW"
ts_access_key="XXXXXXXXXXXXXXXXXXXXXXXXXX"
ts_secret_key="YYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYY"
```
> [!IMPORTANT]
> Replace the values above with the `ts_project_id`, the `ts_access_key`, and `ts_secret_key`

Now use the `terraform` cli with the `secrets.tfvars` file, for example:

```shell
terraform plan --var-file=secrets.tfvars
```

#### VPC Peering

> [!NOTE]  
> The example file creates:
>  * A VPC with name `tf-test` in `us-east-1`
>  * A peering connection

Create a `main.tf` configuration file with the following content.

```hcl
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


variable "ts_region" {
     type    = string
     default = "us-east-1"
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

resource "timescale_vpcs" "main" {
     cidr        = "10.10.0.0/16"
     name        = "tf-test"
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

### Regions
Please reference the [docs](https://docs.timescale.com/use-timescale/latest/regions/) for a list of currently supported regions.

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

