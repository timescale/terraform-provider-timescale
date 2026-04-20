# Look up an existing Private Link authorization by subscription ID
data "timescale_privatelink_authorization" "existing" {
  subscription_id = "00000000-0000-0000-0000-000000000000"
}

output "authorization_name" {
  value = data.timescale_privatelink_authorization.existing.name
}
