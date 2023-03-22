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
      source  = "registry.terraform.io/providers/timescale"
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


## Local Provider Usage and Development
#### Requirements
- [Go](https://go.dev) >= v1.18

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
