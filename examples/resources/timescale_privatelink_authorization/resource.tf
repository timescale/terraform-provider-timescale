# Authorize an Azure subscription to connect via Private Link
resource "timescale_privatelink_authorization" "example" {
  subscription_id = "00000000-0000-0000-0000-000000000000"
  name            = "My Azure Subscription"
}
