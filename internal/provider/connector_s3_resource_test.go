package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccConnectorS3Resource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing with CSV
			{
				Config: testAccConnectorS3ResourceConfigCSV("test-bucket", "*.csv", "test_table"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "bucket", "test-bucket"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "pattern", "*.csv"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "definition.type", "CSV"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "table_identifier.table_name", "test_table"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "table_identifier.schema_name", "public"),
					resource.TestCheckResourceAttrSet("timescale_connector_s3.test", "id"),
					resource.TestCheckResourceAttrSet("timescale_connector_s3.test", "created_at"),
				),
			},
			// Update testing
			{
				Config: testAccConnectorS3ResourceConfigCSV("test-bucket", "data/*.csv", "test_table"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "pattern", "data/*.csv"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccConnectorS3ResourceParquet(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing with Parquet
			{
				Config: testAccConnectorS3ResourceConfigParquet("test-bucket-parquet", "*.parquet", "parquet_table"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "bucket", "test-bucket-parquet"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "pattern", "*.parquet"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "definition.type", "PARQUET"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "table_identifier.table_name", "parquet_table"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "table_identifier.schema_name", "public"),
				),
			},
		},
	})
}

func testAccConnectorS3ResourceConfigCSV(bucket, pattern, tableName string) string {
	return providerConfig + fmt.Sprintf(`
resource "timescale_service" "test" {
  name = "connector-s3-test"
}

resource "timescale_connector_s3" "test" {
  service_id = timescale_service.test.id
  name       = "test-s3-connector"
  bucket     = %[1]q
  pattern    = %[2]q

  credentials = {
    type = "Public"
  }

  definition = {
    type = "CSV"
    csv = {
      skip_header         = true
      auto_column_mapping = true
    }
  }

  table_identifier = {
    schema_name = "public"
    table_name  = %[3]q
  }

  enabled = true
}
`, bucket, pattern, tableName)
}

func testAccConnectorS3ResourceConfigParquet(bucket, pattern, tableName string) string {
	return providerConfig + fmt.Sprintf(`
resource "timescale_service" "test" {
  name = "connector-s3-parquet-test"
}

resource "timescale_connector_s3" "test" {
  service_id = timescale_service.test.id
  name       = "test-s3-parquet-connector"
  bucket     = %[1]q
  pattern    = %[2]q

  credentials = {
    type = "Public"
  }

  definition = {
    type = "PARQUET"
    parquet = {
      auto_column_mapping = true
    }
  }

  table_identifier = {
    schema_name = "public"
    table_name  = %[3]q
  }

  enabled = true
}
`, bucket, pattern, tableName)
}

func TestAccConnectorS3ResourceMinimal(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing with minimal configuration
			{
				Config: testAccConnectorS3ResourceConfigMinimal("minimal-test-bucket", "*.csv", "minimal_table"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "bucket", "minimal-test-bucket"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "pattern", "*.csv"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "definition.type", "CSV"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "table_identifier.table_name", "minimal_table"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "table_identifier.schema_name", "public"),
					resource.TestCheckResourceAttrSet("timescale_connector_s3.test", "id"),
					resource.TestCheckResourceAttrSet("timescale_connector_s3.test", "created_at"),
					// Check defaults are applied
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "frequency", "@always"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "on_conflict_do_nothing", "false"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "enabled", "true"),
				),
			},
		},
	})
}

func TestAccConnectorS3ResourceFull(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing with full configuration
			{
				Config: testAccConnectorS3ResourceConfigFull("full-test-bucket", "data/*.csv", "full_table"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "name", "full-config-connector"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "bucket", "full-test-bucket"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "pattern", "data/*.csv"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "frequency", "@30minutes"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "on_conflict_do_nothing", "true"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "enabled", "false"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "credentials.type", "RoleARN"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "credentials.role_arn", "arn:aws:iam::123456789012:role/TestRole"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "definition.type", "CSV"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "definition.csv.delimiter", "|"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "definition.csv.skip_header", "false"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "definition.csv.column_names.0", "timestamp"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "definition.csv.column_names.1", "value"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "definition.csv.column_names.2", "device_id"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "table_identifier.schema_name", "public"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "table_identifier.table_name", "full_table"),
					resource.TestCheckResourceAttrSet("timescale_connector_s3.test", "id"),
					resource.TestCheckResourceAttrSet("timescale_connector_s3.test", "created_at"),
				),
			},
			// Update to enable the connector
			{
				Config: testAccConnectorS3ResourceConfigFullEnabled("full-test-bucket", "data/*.csv", "full_table"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "enabled", "true"),
				),
			},
		},
	})
}

func testAccConnectorS3ResourceConfigMinimal(bucket, pattern, tableName string) string {
	return providerConfig + fmt.Sprintf(`
resource "timescale_service" "test" {
  name = "connector-s3-minimal-test"
}

resource "timescale_connector_s3" "test" {
  service_id = timescale_service.test.id
  name       = "minimal-connector"
  bucket     = %[1]q
  pattern    = %[2]q

  credentials = {
    type = "Public"
  }

  definition = {
    type = "CSV"
    csv = {
      skip_header         = true
      auto_column_mapping = true
    }
  }

  table_identifier = {
    schema_name = "public"
    table_name  = %[3]q
  }
}
`, bucket, pattern, tableName)
}

func testAccConnectorS3ResourceConfigFull(bucket, pattern, tableName string) string {
	return providerConfig + fmt.Sprintf(`
resource "timescale_service" "test" {
  name = "connector-s3-full-test"
}

resource "timescale_connector_s3" "test" {
  service_id              = timescale_service.test.id
  name                    = "full-config-connector"
  bucket                  = %[1]q
  pattern                 = %[2]q
  frequency               = "@30minutes"
  on_conflict_do_nothing  = true
  enabled                 = false

  credentials = {
    type     = "RoleARN"
    role_arn = "arn:aws:iam::123456789012:role/TestRole"
  }

  definition = {
    type = "CSV"
    csv = {
      delimiter    = "|"
      skip_header  = false
      column_names = ["timestamp", "value", "device_id"]
    }
  }

  table_identifier = {
    schema_name = "public"
    table_name  = %[3]q
  }
}
`, bucket, pattern, tableName)
}

func testAccConnectorS3ResourceConfigFullEnabled(bucket, pattern, tableName string) string {
	return providerConfig + fmt.Sprintf(`
resource "timescale_service" "test" {
  name = "connector-s3-full-test"
}

resource "timescale_connector_s3" "test" {
  service_id              = timescale_service.test.id
  name                    = "full-config-connector"
  bucket                  = %[1]q
  pattern                 = %[2]q
  frequency               = "@30minutes"
  on_conflict_do_nothing  = true
  enabled                 = true

  credentials = {
    type     = "RoleARN"
    role_arn = "arn:aws:iam::123456789012:role/TestRole"
  }

  definition = {
    type = "CSV"
    csv = {
      delimiter    = "|"
      skip_header  = false
      column_names = ["timestamp", "value", "device_id"]
    }
  }

  table_identifier = {
    schema_name = "public"
    table_name  = %[3]q
  }
}
`, bucket, pattern, tableName)
}

func TestAccConnectorS3ResourceColumnMapping(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConnectorS3ResourceConfigColumnMapping("mapping-test-bucket", "data/*.csv", "mapped_table"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "bucket", "mapping-test-bucket"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "pattern", "data/*.csv"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "definition.type", "CSV"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "definition.csv.skip_header", "true"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "definition.csv.column_mappings.0.source", "ts"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "definition.csv.column_mappings.0.destination", "timestamp"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "definition.csv.column_mappings.1.source", "val"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "definition.csv.column_mappings.1.destination", "value"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "definition.csv.column_mappings.2.source", "dev_id"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "definition.csv.column_mappings.2.destination", "device_id"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "table_identifier.table_name", "mapped_table"),
				),
			},
		},
	})
}

func testAccConnectorS3ResourceConfigColumnMapping(bucket, pattern, tableName string) string {
	return providerConfig + fmt.Sprintf(`
resource "timescale_service" "test" {
  name = "connector-s3-mapping-test"
}

resource "timescale_connector_s3" "test" {
  service_id = timescale_service.test.id
  name       = "column-mapping-connector"
  bucket     = %[1]q
  pattern    = %[2]q

  credentials = {
    type = "Public"
  }

  definition = {
    type = "CSV"
    csv = {
      skip_header = true
      column_mappings = [
        {
          source      = "ts"
          destination = "timestamp"
        },
        {
          source      = "val"
          destination = "value"
        },
        {
          source      = "dev_id"
          destination = "device_id"
        }
      ]
    }
  }

  table_identifier = {
    schema_name = "public"
    table_name  = %[3]q
  }

  enabled = true
}
`, bucket, pattern, tableName)
}

func TestAccConnectorS3ResourceImport(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConnectorS3ResourceConfigCSV("import-test-bucket", "*.csv", "import_table"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "bucket", "import-test-bucket"),
					resource.TestCheckResourceAttrSet("timescale_connector_s3.test", "id"),
					resource.TestCheckResourceAttrSet("timescale_connector_s3.test", "service_id"),
				),
			},
			{
				ResourceName:      "timescale_connector_s3.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *resource.State) (string, error) {
					rs, ok := s.RootModule().Resources["timescale_connector_s3.test"]
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
