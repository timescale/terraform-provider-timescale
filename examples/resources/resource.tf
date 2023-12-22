resource "timescale_service" "test" {
  # name       = ""
  # milli_cpu  = 1000
  # memory_gb  = 4
  # region_code = ""
}

# Read replica
resource "timescale_service" "read_replica" {
  read_replica_source = timescale_service.test.id
}
