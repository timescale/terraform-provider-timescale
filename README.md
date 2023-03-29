# Timescale Cloud Terraform Provider
The Terraform provider for [Timescale Cloud](https://www.timescale.com/cloud).

## Requirements
- [Terraform](https://www.terraform.io/downloads.html) >= 1.0

## Quick Start

### Authorization
When you log in to your [Timescale Cloud Account](https://console.cloud.timescale.com/), navigate to the `Project settings` page. 
From here, you can create client credentials for programmatic usage. Click the `Create credentials` button to generate a new public/secret key pair.

### Project ID
The project ID can be found from the `Services` dashboard. In the upper right-hand side of the page, click on the three vertical dots to view the project ID. 

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

resource "timescale_service" "test" {
  # name       = ""
  # milli_cpu  = 500
  # memory_gb  = 2
  # storage_gb = 10
}
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
- 10 GB
- 25 GB - 500 GB in 25 GB increments
- 500 GB - 1TB in 100 GB increments
- 1TB - 3TB in 500 GB increments
- 3TB - 16TB in 1 TB increments

## Supported Operations
✅ Create service <br />
✅ Rename service <br />
✅ Resize service <br />
✅ Delete service <br />
✅ Import service <br />

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

### Local provider development override
To use the locally built provider, create a `~/.terraformrc` file with the following content

```text
provider_installation {

dev_overrides {
   "registry.terraform.io/providers/timescale" = "<PATH>"
}

direct {}
}
```
Change the `<Path>` variable to be the location of your `GOBIN`.

### Developing the Provider
To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `go generate`.

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources.

```shell
make testacc
```
