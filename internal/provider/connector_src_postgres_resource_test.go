package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func testAccPreCheckPgSrcConnector(t *testing.T) {
	t.Helper()
	testAccPreCheck(t)

	if _, ok := os.LookupEnv("TF_VAR_pg_source_connection_string"); !ok {
		t.Skip("TF_VAR_pg_source_connection_string not set, skipping pgsrc connector test")
	}
}

func TestAccConnectorSrcPostgresResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckPgSrcConnector(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccConnectorSrcPostgresConfig("pg-connector-test", "pg-source-cfg", true, 4),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_connector_src_postgres.test", "display_name", "pg-connector-test"),
					resource.TestCheckResourceAttr("timescale_connector_src_postgres.test", "name", "pg-source-cfg"),
					resource.TestCheckResourceAttr("timescale_connector_src_postgres.test", "enabled", "true"),
					resource.TestCheckResourceAttr("timescale_connector_src_postgres.test", "table_sync_workers", "4"),
					resource.TestCheckResourceAttrSet("timescale_connector_src_postgres.test", "id"),
					resource.TestCheckResourceAttrSet("timescale_connector_src_postgres.test", "source_id"),
					resource.TestCheckResourceAttrSet("timescale_connector_src_postgres.test", "created_at"),
					resource.TestCheckResourceAttrSet("timescale_connector_src_postgres.test", "status"),
				),
			},
			// Update display_name and table_sync_workers
			{
				Config: testAccConnectorSrcPostgresConfig("pg-connector-renamed", "pg-source-cfg", true, 2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_connector_src_postgres.test", "display_name", "pg-connector-renamed"),
					resource.TestCheckResourceAttr("timescale_connector_src_postgres.test", "table_sync_workers", "2"),
				),
			},
			// Disable
			{
				Config: testAccConnectorSrcPostgresConfig("pg-connector-renamed", "pg-source-cfg", false, 2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_connector_src_postgres.test", "enabled", "false"),
				),
			},
			// Delete automatically occurs in TestCase
		},
	})
}

func TestAccConnectorSrcPostgresResource_tables(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckPgSrcConnector(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with tables
			{
				Config: testAccConnectorSrcPostgresConfigWithTables(
					"pg-connector-tables", "pg-source-tables",
					[]testTable{{schema: "public", table: "sensor_readings"}, {schema: "public", table: "user_events"}},
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_connector_src_postgres.test", "tables.#", "2"),
				),
			},
			// Add a table
			{
				Config: testAccConnectorSrcPostgresConfigWithTables(
					"pg-connector-tables", "pg-source-tables",
					[]testTable{
						{schema: "public", table: "sensor_readings"},
						{schema: "public", table: "user_events"},
						{schema: "public", table: "order_transactions"},
					},
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_connector_src_postgres.test", "tables.#", "3"),
				),
			},
			// Drop a table
			{
				Config: testAccConnectorSrcPostgresConfigWithTables(
					"pg-connector-tables", "pg-source-tables",
					[]testTable{{schema: "public", table: "sensor_readings"}, {schema: "public", table: "order_transactions"}},
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_connector_src_postgres.test", "tables.#", "2"),
				),
			},
		},
	})
}

func TestAccConnectorSrcPostgresResource_import(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckPgSrcConnector(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConnectorSrcPostgresConfig("pg-connector-import", "pg-source-import", true, 4),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("timescale_connector_src_postgres.test", "id"),
					resource.TestCheckResourceAttrSet("timescale_connector_src_postgres.test", "service_id"),
				),
			},
			{
				ResourceName:      "timescale_connector_src_postgres.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"connection_string",
				},
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					rs, ok := state.RootModule().Resources["timescale_connector_src_postgres.test"]
					if !ok {
						return "", fmt.Errorf("resource not found")
					}
					serviceID := rs.Primary.Attributes["service_id"]
					connectorID := rs.Primary.Attributes["id"]
					return fmt.Sprintf("%s:%s", serviceID, connectorID), nil
				},
			},
		},
	})
}

// --- Config helpers ---

type testTable struct {
	schema string
	table  string
}

func testAccConnectorSrcPostgresConfig(displayName, name string, enabled bool, workers int) string {
	return providerConfig + fmt.Sprintf(`
variable "pg_source_connection_string" {
  type      = string
  sensitive = true
}

resource "timescale_service" "test" {
  name = "connector-pgsrc-test"
}

resource "timescale_connector_src_postgres" "test" {
  service_id         = timescale_service.test.id
  display_name       = %[1]q
  name               = %[2]q
  connection_string  = var.pg_source_connection_string
  enabled            = %[3]t
  table_sync_workers = %[4]d
}
`, displayName, name, enabled, workers)
}

func testAccConnectorSrcPostgresConfigWithTables(displayName, name string, tables []testTable) string {
	tablesHCL := ""
	for _, t := range tables {
		tablesHCL += fmt.Sprintf(`
  tables {
    schema_name = %q
    table_name  = %q
  }
`, t.schema, t.table)
	}

	return providerConfig + fmt.Sprintf(`
variable "pg_source_connection_string" {
  type      = string
  sensitive = true
}

resource "timescale_service" "test" {
  name = "connector-pgsrc-tables-test"
}

resource "timescale_connector_src_postgres" "test" {
  service_id         = timescale_service.test.id
  display_name       = %[1]q
  name               = %[2]q
  connection_string  = var.pg_source_connection_string
  enabled            = true
  table_sync_workers = 4
%[3]s}
`, displayName, name, tablesHCL)
}
