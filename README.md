# Timescale Terraform Provider
The Terraform provider for [Timescale](https://www.timescale.com/cloud).

Does not work for Managed Service for TimescaleDB.

## Requirements
- [Terraform](https://www.terraform.io/downloads.html) >= 1.0

## Quick Start

### Authorization
When you log in to your [Timescale Account](https://console.cloud.timescale.com/), navigate to the `Project settings` page. 
From here, you can create client credentials for programmatic usage. Click the `Create credentials` button to generate a new public/secret key pair.

Find more information on creating Client Credentials in the [Timescale docs](https://docs.timescale.com/use-timescale/latest/security/client-credentials/#creating-client-credentials).

### Example file and usage

The project ID can be found from the `Services` dashboard. In the upper right-hand side of the page, click on the three vertical dots to view the project ID. 


> [!NOTE]  
> The example file creates:
>  * A single instance called `tf-test`
>  * Outputs to display the connection info for:
>    * the primary hostname and port
>    * the ha-replica hostname and port
>    * the pooler hostname and port

Into a new folder, create the `main.tf` file:

```hcl
terraform {
  required_providers {
    timescale = {
      source  = "registry.terraform.io/providers/timescale"
      version = "~> 1.0"
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
  type = string
}

resource "timescale_service" "tf-test" {
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
> Replace the values above with the the `ts_project_id`, the `ts_access_key`, and `ts_secret_key`

Now use the `terraform` cli with the `secrets.tfvars` file, for example:

```shell
terraform plan --var-file=secrets.tfvars
```

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

## Local Provider Usage and Development
#### Requirements
- [Go](https://go.dev) >= v1.20

#### Building The Provider
1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:

```shell
go install .
```

#### Generating the provider documentation

Doc is generated from `./templates` files and in-file schema definitions.

```shell
go generate
```


## Developing the Provider

### Local provider development override

> [!IMPORTANT]
> Change the `$HOME/go/bin` variable to be the location of your `GOBIN` if necessary.
>
> When using the local provider, is not necessary to run `terraform init`.

To use the locally built provider, create a `~/.terraformrc` file with the following content:

```hcl
provider_installation {
  dev_overrides {
      "registry.terraform.io/providers/timescale" = "$HOME/go/bin"
  }

  direct {}
}
```

### Runing the acceptance tests

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `go generate`.

In order to run the full suite of Acceptance tests, run `make testacc`.

> [!WARNING]
> Acceptance tests create real resources.

```shell
make testacc
```
