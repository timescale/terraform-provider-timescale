# Look up a Private Link connection this project has claimed.
#
# Only claimed connections are visible. An unclaimed connection belongs to no
# project and cannot be looked up — claim it with the
# timescale_privatelink_connection resource instead.
data "timescale_privatelink_connection" "existing" {
  connection_id = "549b2b12-c82c-4005-a7f7-175f607f086c"
}

# Attach a service to a connection claimed elsewhere, for example in another
# Terraform workspace.
resource "timescale_service" "example" {
  name        = "example"
  region_code = data.timescale_privatelink_connection.existing.region
  milli_cpu   = 500
  memory_gb   = 2

  private_endpoint_connection_ids = [
    data.timescale_privatelink_connection.existing.connection_id,
  ]
}

output "connection_state" {
  value = data.timescale_privatelink_connection.existing.state
}
