terraform {
  required_version = ">= 1.3.0"

  required_providers {
    timescale = {
      source  = "timescale/timescale"
      version = "~> 2.8"
    }
    azurerm = {
      source  = "hashicorp/azurerm"
      version = ">= 3.70.0"
    }
  }
}

# =============================================================================
# Variables
# =============================================================================

variable "ts_access_key" {
  type        = string
  description = "Timescale access key"
}

variable "ts_secret_key" {
  type        = string
  sensitive   = true
  description = "Timescale secret key"
}

variable "ts_project_id" {
  type        = string
  description = "Timescale project ID"
}

variable "azure_subscription_id" {
  type        = string
  description = "Azure subscription ID"
}

variable "azure_location" {
  type        = string
  description = "Azure region for infrastructure (VNet, VM, Private Endpoint)"
  default     = "eastus"
}

variable "timescale_region" {
  type        = string
  description = "Timescale region for the service (e.g., az-eastus, az-eastus2)"
  default     = "az-eastus"
}

variable "resource_prefix" {
  type        = string
  description = "Prefix for all resource names"
  default     = "tspl-demo"
}

# =============================================================================
# Providers
# =============================================================================

provider "timescale" {
  access_key = var.ts_access_key
  secret_key = var.ts_secret_key
  project_id = var.ts_project_id
}

provider "azurerm" {
  features {}
  subscription_id = var.azure_subscription_id
}


# =============================================================================
# Azure Infrastructure - Resource Group & Network
# =============================================================================

resource "azurerm_resource_group" "main" {
  name     = "${var.resource_prefix}-rg"
  location = var.azure_location
}

resource "azurerm_virtual_network" "main" {
  name                = "${var.resource_prefix}-vnet"
  address_space       = ["10.3.0.0/16"]
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
}

resource "azurerm_subnet" "vm" {
  name                 = "vm-subnet"
  resource_group_name  = azurerm_resource_group.main.name
  virtual_network_name = azurerm_virtual_network.main.name
  address_prefixes     = ["10.3.1.0/24"]
}

resource "azurerm_subnet" "endpoint" {
  name                              = "endpoint-subnet"
  resource_group_name               = azurerm_resource_group.main.name
  virtual_network_name              = azurerm_virtual_network.main.name
  address_prefixes                  = ["10.3.2.0/24"]
  private_endpoint_network_policies = "Disabled"
}

# =============================================================================
# Azure Infrastructure - VM for testing connectivity
# =============================================================================

resource "azurerm_public_ip" "vm" {
  name                = "${var.resource_prefix}-vm-pip"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  allocation_method   = "Static"
  sku                 = "Standard"
}

resource "azurerm_network_interface" "vm" {
  name                = "${var.resource_prefix}-vm-nic"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name

  ip_configuration {
    name                          = "ipconfig1"
    subnet_id                     = azurerm_subnet.vm.id
    private_ip_address_allocation = "Dynamic"
    public_ip_address_id          = azurerm_public_ip.vm.id
  }
}

resource "azurerm_network_security_group" "vm" {
  name                = "${var.resource_prefix}-nsg"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name

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
}

resource "azurerm_network_interface_security_group_association" "vm" {
  network_interface_id      = azurerm_network_interface.vm.id
  network_security_group_id = azurerm_network_security_group.vm.id
}

resource "azurerm_linux_virtual_machine" "vm" {
  name                = "${var.resource_prefix}-vm"
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location
  size                = "Standard_B1s"
  admin_username      = "adminuser"

  network_interface_ids = [
    azurerm_network_interface.vm.id,
  ]

  admin_ssh_key {
    username   = "adminuser"
    public_key = file("~/.ssh/id_rsa.pub")
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

  custom_data = base64encode(<<-EOF
    #!/bin/bash
    set -e
    export DEBIAN_FRONTEND=noninteractive
    apt-get update
    apt-get install -y postgresql-client netcat-openbsd curl
    psql --version
  EOF
  )

  tags = {
    Environment = "Demo"
    ManagedBy   = "Terraform"
  }
}

# =============================================================================
# Timescale Private Link Setup
# =============================================================================

# Step 1: Authorize the Azure subscription
resource "timescale_privatelink_authorization" "main" {
  subscription_id = var.azure_subscription_id
  name            = "Terraform managed - ${var.resource_prefix}"
}

# Step 2: Get the Private Link Service alias for the region
data "timescale_privatelink_available_regions" "all" {}

locals {
  private_link_service_alias = data.timescale_privatelink_available_regions.all.regions[var.timescale_region].private_link_service_alias
}

# =============================================================================
# Azure Private Endpoint
# =============================================================================

resource "azurerm_private_endpoint" "timescale" {
  name                = "${var.resource_prefix}-pe"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  subnet_id           = azurerm_subnet.endpoint.id

  private_service_connection {
    name                              = "${var.resource_prefix}-psc"
    private_connection_resource_alias = local.private_link_service_alias
    is_manual_connection              = true
    request_message                   = var.ts_project_id
  }

  tags = {
    Environment = "Demo"
    ManagedBy   = "Terraform"
  }

  depends_on = [timescale_privatelink_authorization.main]
}

# Data source to get the connection status (not available on the resource itself)
data "azurerm_private_endpoint_connection" "timescale" {
  name                = azurerm_private_endpoint.timescale.name
  resource_group_name = azurerm_resource_group.main.name
}

# =============================================================================
# Timescale Private Link Connection
# =============================================================================

# This resource syncs and waits for the Azure connection to appear,
# then configures it with the IP address from the Azure Private Endpoint
resource "timescale_privatelink_connection" "main" {
  azure_connection_name = azurerm_private_endpoint.timescale.name
  region                = var.timescale_region
  ip_address            = azurerm_private_endpoint.timescale.private_service_connection[0].private_ip_address
  name                  = "Managed by Terraform"

  depends_on = [azurerm_private_endpoint.timescale]

  lifecycle {
    ## This is needed because we need to attach to the new connection before we can destroy the old one.action_trigger 
    ## only useful in the scenario when we move between connections
    create_before_destroy = true
  }
}

# =============================================================================
# Timescale Service with Private Link
# =============================================================================

resource "timescale_service" "main" {
  name        = "${var.resource_prefix}-db"
  milli_cpu   = 500
  memory_gb   = 2
  region_code = var.timescale_region

  private_endpoint_connection_id = timescale_privatelink_connection.main.connection_id
}

# =============================================================================
# Outputs
# =============================================================================

output "vm_public_ip" {
  description = "Public IP of VM for SSH access"
  value       = azurerm_public_ip.vm.ip_address
}

output "vm_ssh_command" {
  description = "SSH command to connect to VM"
  value       = "ssh adminuser@${azurerm_public_ip.vm.ip_address}"
}

output "private_endpoint_ip" {
  description = "Private IP of the Private Endpoint"
  value       = azurerm_private_endpoint.timescale.private_service_connection[0].private_ip_address
}

output "timescale_hostname" {
  description = "Timescale service hostname"
  value       = timescale_service.main.hostname
}

output "timescale_port" {
  description = "Timescale service port"
  value       = timescale_service.main.port
}

output "timescale_username" {
  description = "Timescale service username"
  value       = timescale_service.main.username
}

output "connection_test_command_private_ip" {
  description = "Command to test connection from VM using private IP (run after SSH)"
  value       = "PGPASSWORD='${timescale_service.main.password}' psql -h ${azurerm_private_endpoint.timescale.private_service_connection[0].private_ip_address} -p ${timescale_service.main.port} -U ${timescale_service.main.username} -d tsdb"
  sensitive   = true
}

output "connection_test_command_hostname" {
  description = "Command to test connection using service hostname (run after SSH)"
  value       = "PGPASSWORD='${timescale_service.main.password}' psql -h ${timescale_service.main.hostname} -p ${timescale_service.main.port} -U ${timescale_service.main.username} -d tsdb"
  sensitive   = true
}

output "private_link_connection_state" {
  description = "State of the Private Link connection"
  value       = timescale_privatelink_connection.main.state
}

output "azure_private_endpoint_status" {
  description = "Azure Private Endpoint connection status (Pending, Approved, Rejected, Disconnected)"
  value       = data.azurerm_private_endpoint_connection.timescale.private_service_connection[0].status
}

output "azure_private_endpoint_message" {
  description = "Azure Private Endpoint connection request/response message"
  value       = data.azurerm_private_endpoint_connection.timescale.private_service_connection[0].request_response
}

output "private_link_connection_id" {
  description = "Connection ID for use with timescale_service"
  value       = timescale_privatelink_connection.main.connection_id
}

output "service_id" {
  description = "Timescale service ID"
  value       = timescale_service.main.id
}

output "private_link_service_alias" {
  description = "The Private Link Service alias used"
  value       = local.private_link_service_alias
}

output "ssh_select_1" {
  description = "SSH command to execute SELECT 1 on the database via private link"
  value       = "ssh adminuser@${azurerm_public_ip.vm.ip_address} \"PGPASSWORD='${timescale_service.main.password}' psql -h ${timescale_service.main.hostname} -p ${timescale_service.main.port} -U ${timescale_service.main.username} -d tsdb -c 'SELECT 1'\""
  sensitive   = true
}
