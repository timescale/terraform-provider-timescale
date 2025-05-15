# Timescale Terraform Provider
The Terraform provider for [Timescale](https://www.timescale.com/cloud).

## Requirements
- [Terraform](https://www.terraform.io/downloads.html) >= 1.0

## Quick Start
Check provider [documentation](docs/index.md#quick-start)

## Local Provider Usage and Development
### Requirements
- [Go](https://go.dev) >= v1.24

### Building The Provider
1. Clone the repository
1. Enter the repository directory
1. Run `make` to fmt/lint/install/generate:
```shell
make
```

### Update documentation

`docs` folder content is automatically generated. Please **do not modify these files manually**.

Update `./templates/index.md` and just run `make`:
```shell
make
```

`data-sources` and `resources` doc files are generated from the actual provider go code (Schema definitions, descriptions, etc.).


### Local provider development override

By default, Terraform cli will search providers in the official registry (registry.terraform.io).
The following steps will tell Terraform to look for this specific provider in the local computer.
Remember to remove this configuration when finished.

> Change the `$HOME/go/bin` variable to be the location of your `GOBIN` if necessary.
>
> When using the local provider, is not necessary to run `terraform init`.

To use the local provider, create a `~/.terraformrc` file with the following content:

```hcl
provider_installation {
  dev_overrides {
      "registry.terraform.io/providers/timescale" = "$HOME/go/bin"
  }

  direct {}
}
```

From now on, `terraform plan` and `terraform apply` will interact with the locally installed provider.

Remember to run `make` again whenever the provider code is changed.

### Running the acceptance tests

> [WARNING]
> Acceptance tests create real resources.


Run `make` to install the last version of the provider.

Set all the required environment variables to allow the tests to run:

```shell
export TF_VAR_ts_project_id=<project_id>
export TF_VAR_ts_access_key=<access_key>
export TF_VAR_ts_secret_key=<secret_key>
export TIMESCALE_DEV_URL=<api_url> # Optional: to use different environment
```

Use `make testacc` to run the full acceptance tests suite. This can take up to 20 minutes as several services are created. 

**Please do not abort the execution to prevent dangling resources.**
```shell
make testacc
```

### Dangling resources and sweepers

Acceptance tests usually destroy all created assets, but failures or execution abortions can leave dangling resources.

We use [sweepers](https://developer.hashicorp.com/terraform/plugin/testing/acceptance-tests/sweepers) to cleanup all possible dangling resources before running the acceptance tests.

```
make testacc SWEEP=timescale_vpcs
```

