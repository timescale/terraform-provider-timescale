# Claim a Private Link connection for this project.
#
# Tiger Cloud's endpoint service is open: anyone can create a private endpoint
# against it, and every connection is accepted but left unowned. Claiming it
# with an identifier read from your own cloud resource establishes ownership —
# authentication to the project is the proof.
#
# Create the cloud-side endpoint yourself, then claim it here. See the aws/ and
# azure/ subdirectories for complete, runnable setups.

# AWS: the claim identifier is the VPC endpoint ID.
resource "timescale_privatelink_connection" "aws" {
  claim_identifier = aws_vpc_endpoint.example.id
  name             = "production"
}

# Azure: the claim identifier is the private endpoint's resourceGuid, which
# azurerm does not expose (hashicorp/terraform-provider-azurerm#17011). Read it
# from the Azure API with the azapi provider.
data "azapi_resource" "example_pe" {
  type                   = "Microsoft.Network/privateEndpoints@2024-05-01"
  resource_id            = azurerm_private_endpoint.example.id
  response_export_values = ["properties.resourceGuid"]
}

resource "timescale_privatelink_connection" "azure" {
  claim_identifier = data.azapi_resource.example_pe.output.properties.resourceGuid
  name             = "production"
}

# Attach services to the claimed connection.
resource "timescale_service" "example" {
  name        = "example"
  region_code = "us-east-1"
  milli_cpu   = 500
  memory_gb   = 2

  private_endpoint_connection_ids = [timescale_privatelink_connection.aws.connection_id]
}
