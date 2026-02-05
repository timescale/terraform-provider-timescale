# Private Link Connection Example

This example creates a complete Azure Private Link setup with a Timescale service.

## Prerequisites

1. **Authorize your Azure subscription in Timescale Console**
   - Go to Settings → Private Link → Add Authorization
   - Enter your Azure Subscription ID
   - Note the Private Link Service alias shown after authorization

2. **Azure CLI** - logged in with `az login`

3. **Terraform** >= 1.3.0

## Setup

1. Copy and fill in the tfvars file:
   ```bash
   cp terraform.tfvars.example terraform.tfvars
   # Edit terraform.tfvars with your values
   ```

2. Build the provider locally (from repo root):
   ```bash
   cd /path/to/terraform-provider-timescale
   go build -o terraform-provider-timescale
   ```

3. Configure dev override (one-time):
   ```bash
   cat > ~/.terraformrc << 'EOF'
   provider_installation {
     dev_overrides {
       "timescale/timescale" = "/path/to/terraform-provider-timescale"
     }
     direct {}
   }
   EOF
   ```

## Run

```bash
# Initialize (downloads Azure provider)
terraform init

# Review plan
terraform plan

# Apply
terraform apply
```

## Troubleshooting

### Timeout waiting for Private Link connection

If you see this error, the resource couldn't find the connection after syncing. Check:

1. **Authorization**: Is the Azure subscription authorized in Timescale Console?
2. **Project ID**: Does the `request_message` in the Private Endpoint match your `ts_project_id`?
3. **Region**: Does the `private_link_service_alias` match the `azure_location`?

You can increase the timeout:
```hcl
resource "timescale_privatelink_connection" "main" {
  azure_connection_name = azurerm_private_endpoint.timescale.name
  region                = "az-${var.azure_location}"
  ip_address            = azurerm_private_endpoint.timescale.private_service_connection[0].private_ip_address
  timeout               = "5m"  # Increase from default 2m
}
```

## Cleanup

```bash
terraform destroy
```

## Outputs

After successful apply:

- `vm_ssh_command` - SSH into the test VM
- `connection_test_command_private_ip` - psql command to test from VM using private IP
- `connection_test_command_hostname` - psql command to test from VM using hostname
- `timescale_hostname` - Service hostname
- `private_endpoint_ip` - Private IP for the endpoint
- `private_link_connection_state` - State of the Private Link connection
