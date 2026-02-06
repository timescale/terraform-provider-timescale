# Look up an existing Private Link connection by connection ID
data "timescale_privatelink_connection" "by_id" {
  connection_id = "conn-123"
}

# Or look up by Azure connection name and region
data "timescale_privatelink_connection" "by_name" {
  azure_connection_name = "my-private-endpoint"
  region                = "az-eastus2"
}

output "connection_ip" {
  value = data.timescale_privatelink_connection.by_id.ip_address
}
