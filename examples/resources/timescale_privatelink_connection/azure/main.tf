# Minimal end-to-end Azure Private Link setup:
#
#   1. a small Tiger Cloud service
#   2. a private endpoint, claimed as a private link connection
#   3. a small VM to test from
#   4. a `terraform output` command that runs the test
#
# Tiger Cloud does not publish DNS for private link. The customer owns the
# records, so this example also creates a private DNS zone pointing the service
# hostname at the endpoint. That is what keeps sslmode=verify-full working: the
# client asks for the same hostname the public endpoint uses, and the existing
# certificate covers it.

terraform {
  required_version = ">= 1.3.0"

  required_providers {
    timescale = {
      source  = "timescale/timescale"
      version = "~> 2.0"
    }
    azurerm = {
      source  = "hashicorp/azurerm"
      version = ">= 4.0"
    }
  }
}

variable "ts_access_key" {
  description = "Tiger Cloud client credentials access key."
  type        = string
}

variable "ts_secret_key" {
  description = "Tiger Cloud client credentials secret key."
  type        = string
  sensitive   = true
}

variable "ts_project_id" {
  description = "Tiger Cloud project ID."
  type        = string
}

variable "subscription_id" {
  description = "Azure subscription hosting the test resources."
  type        = string
}

variable "timescale_region" {
  description = "Tiger Cloud region code, e.g. az-eastus2."
  type        = string
  default     = "az-eastus2"
}

variable "azure_location" {
  description = "Azure region for the test resources. The endpoint NIC lives here; it may target a Private Link Service in another region."
  type        = string
  default     = "eastus2"
}

variable "name_prefix" {
  description = "Prefix for every resource name."
  type        = string
  default     = "tf-privatelink-example"
}

variable "vnet_cidr" {
  description = "CIDR of the test VNet. Change it if it overlaps an existing network in the subscription."
  type        = string
  default     = "10.42.0.0/16"
}

variable "admin_username" {
  description = "Admin user created on the test VM."
  type        = string
  default     = "azureuser"
}

variable "ssh_public_key_path" {
  description = "Public key authorized on the test VM."
  type        = string
  default     = "~/.ssh/id_ed25519.pub"
}

provider "timescale" {
  access_key = var.ts_access_key
  secret_key = var.ts_secret_key
  project_id = var.ts_project_id
}

provider "azurerm" {
  features {}
  subscription_id = var.subscription_id
}

# ============================================================================
# 1. Tiger Cloud service
# ============================================================================

resource "timescale_service" "example" {
  name        = var.name_prefix
  region_code = var.timescale_region
  milli_cpu   = 500
  memory_gb   = 2

  private_endpoint_connection_ids = [timescale_privatelink_connection.example.connection_id]

  timeouts = {
    create = "30m"
  }
}

# ============================================================================
# 2. Private endpoint, claimed as a private link connection
# ============================================================================

# The Private Link Service alias to point the private endpoint at.
data "timescale_privatelink_region" "example" {
  region = var.timescale_region
}

resource "azurerm_resource_group" "example" {
  name     = "${var.name_prefix}-rg"
  location = var.azure_location
}

resource "azurerm_virtual_network" "example" {
  name                = "${var.name_prefix}-vnet"
  address_space       = [var.vnet_cidr]
  location            = azurerm_resource_group.example.location
  resource_group_name = azurerm_resource_group.example.name
}

# A private endpoint can only land in a subnet with endpoint network policies
# disabled.
resource "azurerm_subnet" "example" {
  name                              = "${var.name_prefix}-subnet"
  resource_group_name               = azurerm_resource_group.example.name
  virtual_network_name              = azurerm_virtual_network.example.name
  address_prefixes                  = [cidrsubnet(var.vnet_cidr, 8, 1)]
  private_endpoint_network_policies = "Disabled"
}

resource "azurerm_private_endpoint" "example" {
  name                = "${var.name_prefix}-pe"
  location            = azurerm_resource_group.example.location
  resource_group_name = azurerm_resource_group.example.name
  subnet_id           = azurerm_subnet.example.id

  private_service_connection {
    name                              = "${var.name_prefix}-psc"
    private_connection_resource_alias = data.timescale_privatelink_region.example.service_name
    is_manual_connection              = true
    request_message                   = "Tiger Cloud private link example"
  }
}

# The Azure claim identifier is the private endpoint's resourceGuid, which the
# azurerm resource does not expose (hashicorp/terraform-provider-azurerm#17011).
# It does appear in the approval message Tiger Cloud writes back on the
# connection, which this data source exposes as request_response. The provider
# extracts the GUID from it, so the message can be passed through as-is.
data "azurerm_private_endpoint_connection" "example" {
  name                = azurerm_private_endpoint.example.name
  resource_group_name = azurerm_resource_group.example.name
}

# Tiger Cloud's Private Link Service accepts every connection but leaves it
# unowned. Claiming it assigns it to this project.
resource "timescale_privatelink_connection" "example" {
  claim_identifier = data.azurerm_private_endpoint_connection.example.private_service_connection[0].request_response
  name             = var.name_prefix
}

# ============================================================================
# Private DNS: the customer's own zone, pointing the service hostname at the
# endpoint. A plain A record, because an Azure private endpoint has one IP for
# its lifetime and Microsoft handles zone resilience below it.
# ============================================================================

locals {
  hostname_labels = split(".", timescale_service.example.hostname)

  # Split the service hostname into the zone and the record name within it, so
  # every service in this project resolves under one zone:
  #
  #   hostname = "abc123xyz.myproject.db.az.tigerdata.com"
  #   zone     =           "myproject.db.az.tigerdata.com"  (service_domain)
  #   record   = "abc123xyz"                                (service_label)
  #
  # A wildcard record is not possible: Tiger Cloud issues per-service
  # certificates, so only exact hostnames validate under verify-full. Add one
  # record per service.
  service_domain = join(".", slice(local.hostname_labels, 1, length(local.hostname_labels)))
  service_label  = local.hostname_labels[0]
}

resource "azurerm_private_dns_zone" "example" {
  name                = local.service_domain
  resource_group_name = azurerm_resource_group.example.name
}

# A zone only answers for VNets it is linked to.
resource "azurerm_private_dns_zone_virtual_network_link" "example" {
  name                 = "${var.name_prefix}-link"
  private_dns_zone_id  = azurerm_private_dns_zone.example.id
  virtual_network_id   = azurerm_virtual_network.example.id
  registration_enabled = false
}

resource "azurerm_private_dns_a_record" "example" {
  name                = local.service_label
  private_dns_zone_id = azurerm_private_dns_zone.example.id
  ttl                 = 300
  records             = [azurerm_private_endpoint.example.private_service_connection[0].private_ip_address]
}

# ============================================================================
# 3. Test VM
# ============================================================================

resource "azurerm_public_ip" "example" {
  name                = "${var.name_prefix}-pip"
  location            = azurerm_resource_group.example.location
  resource_group_name = azurerm_resource_group.example.name
  allocation_method   = "Static"
  sku                 = "Standard"
}

resource "azurerm_network_security_group" "example" {
  name                = "${var.name_prefix}-nsg"
  location            = azurerm_resource_group.example.location
  resource_group_name = azurerm_resource_group.example.name

  security_rule {
    name                       = "SSH"
    priority                   = 1001
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "22"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }

  # There are no reserved ports here: every binding draws from the backend's
  # pool, so the endpoint answers somewhere in 50000-51000 rather than on 5432.
  # The default AllowVnetOutBound already permits this; the rule is explicit so
  # the range is visible and egress can be tightened without breaking it.
  security_rule {
    name                       = "PostgresPrivateLink"
    priority                   = 1002
    direction                  = "Outbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "50000-51000"
    source_address_prefix      = "*"
    destination_address_prefix = "VirtualNetwork"
  }
}

resource "azurerm_network_interface" "example" {
  name                = "${var.name_prefix}-nic"
  location            = azurerm_resource_group.example.location
  resource_group_name = azurerm_resource_group.example.name

  ip_configuration {
    name                          = "internal"
    subnet_id                     = azurerm_subnet.example.id
    private_ip_address_allocation = "Dynamic"
    public_ip_address_id          = azurerm_public_ip.example.id
  }
}

resource "azurerm_network_interface_security_group_association" "example" {
  network_interface_id      = azurerm_network_interface.example.id
  network_security_group_id = azurerm_network_security_group.example.id
}

resource "azurerm_linux_virtual_machine" "example" {
  name                  = "${var.name_prefix}-vm"
  resource_group_name   = azurerm_resource_group.example.name
  location              = azurerm_resource_group.example.location
  size                  = "Standard_B1s"
  admin_username        = var.admin_username
  network_interface_ids = [azurerm_network_interface.example.id]

  custom_data = base64encode(<<-EOT
    #!/bin/bash
    export DEBIAN_FRONTEND=noninteractive
    apt-get update
    apt-get install -y postgresql-client dnsutils
  EOT
  )

  admin_ssh_key {
    username   = var.admin_username
    public_key = file(pathexpand(var.ssh_public_key_path))
  }

  os_disk {
    caching              = "ReadWrite"
    storage_account_type = "Standard_LRS"
  }

  source_image_reference {
    publisher = "Canonical"
    offer     = "ubuntu-24_04-lts"
    sku       = "server"
    version   = "latest"
  }
}

# ============================================================================
# 4. Test the connection
# ============================================================================

# sslrootcert=system trusts the OS CA bundle. Tiger Cloud certificates come from
# a public CA, so verify-full validates without downloading a root certificate —
# and it validates the same hostname the public endpoint uses, which is the whole
# point of pointing your own DNS at the private endpoint.
output "test_command" {
  description = "SSHes into the test VM and runs SELECT 1 over private link."
  value = join(" ", [
    "ssh -o StrictHostKeyChecking=no",
    "-i ${trimsuffix(pathexpand(var.ssh_public_key_path), ".pub")}",
    "${var.admin_username}@${azurerm_public_ip.example.ip_address}",
    "\"PGPASSWORD='${timescale_service.example.password}'",
    "psql",
    "'host=${timescale_service.example.hostname}",
    "port=${timescale_service.example.port}",
    "user=${timescale_service.example.username}",
    "dbname=tsdb",
    "sslmode=verify-full",
    "sslrootcert=system'",
    "-c 'SELECT 1'\"",
  ])
  sensitive = true
}

output "dns_check_command" {
  description = "Confirms the service hostname resolves to the endpoint inside the VNet."
  value       = "ssh ${var.admin_username}@${azurerm_public_ip.example.ip_address} 'dig +short ${timescale_service.example.hostname}'"
}

output "service_hostname" {
  description = "Hostname to connect to. Same as the public endpoint; only the resolution and port differ."
  value       = timescale_service.example.hostname
}

output "service_port" {
  description = "Private link port for this service, allocated per binding."
  value       = timescale_service.example.port
}

output "claim_identifier" {
  description = "The approval message passed to the provider. The resourceGuid inside it is what identifies the connection."
  value       = data.azurerm_private_endpoint_connection.example.private_service_connection[0].request_response
}

output "connection_id" {
  description = "The claimed private link connection."
  value       = timescale_privatelink_connection.example.connection_id
}

# ============================================================================
# Everything Azure exposes about the private endpoint
#
# Here to make the claim-identifier problem visible. Notice what is missing:
# no field on the azurerm resource carries the endpoint's resourceGuid, which
# is what the claim needs. The GUID that appears inside `id` is the
# subscription ID, not the endpoint's.
#
# Read it all with:  terraform output azure_private_endpoint
# ============================================================================

output "azure_private_endpoint" {
  description = "Every attribute the azurerm_private_endpoint resource exposes."
  value = {
    id                            = azurerm_private_endpoint.example.id
    name                          = azurerm_private_endpoint.example.name
    location                      = azurerm_private_endpoint.example.location
    resource_group_name           = azurerm_private_endpoint.example.resource_group_name
    subnet_id                     = azurerm_private_endpoint.example.subnet_id
    tags                          = azurerm_private_endpoint.example.tags
    custom_network_interface_name = azurerm_private_endpoint.example.custom_network_interface_name
    network_interface             = azurerm_private_endpoint.example.network_interface
    custom_dns_configs            = azurerm_private_endpoint.example.custom_dns_configs
    private_dns_zone_configs      = azurerm_private_endpoint.example.private_dns_zone_configs
    private_service_connection    = azurerm_private_endpoint.example.private_service_connection
  }
}

output "azure_private_endpoint_connection" {
  description = "Every attribute the azurerm_private_endpoint_connection data source exposes. private_service_connection[0].request_response carries Tiger Cloud's approval message, which contains the claim identifier."
  value = {
    id                         = data.azurerm_private_endpoint_connection.example.id
    name                       = data.azurerm_private_endpoint_connection.example.name
    location                   = data.azurerm_private_endpoint_connection.example.location
    resource_group_name        = data.azurerm_private_endpoint_connection.example.resource_group_name
    network_interface          = data.azurerm_private_endpoint_connection.example.network_interface
    private_service_connection = data.azurerm_private_endpoint_connection.example.private_service_connection
  }
}

# The endpoint's resourceGuid is reachable without azurerm exposing it, via the
# raw ARM API. Uncomment along with the azapi provider if you would rather pass
# the bare GUID than the approval message.
#
# data "azapi_resource" "example_pe" {
#   type                   = "Microsoft.Network/privateEndpoints@2024-05-01"
#   resource_id            = azurerm_private_endpoint.example.id
#   response_export_values = ["*"]
# }
