resource "timescale_service" "db" {
  name = "my-service"
}

resource "timescale_ip_allow_list" "office" {
  description = "Office network"
  cidr_blocks = ["203.0.113.0/24", "198.51.100.7/32"]
  service_ids = [timescale_service.db.id]
}
