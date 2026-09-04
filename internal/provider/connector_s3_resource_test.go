package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// The S3 connector API validates s3:ListBucket and s3:GetObject when the connector is
// created, so these tests need a bucket that really exists and a role the connectors
// account can assume. Both are provisioned by the connectors-canaries prod stack
// (terraform/modules/connector_s3), whose role trusts any service under this repo's CI
// project. Nothing here writes to the bucket; only the create-time permission check runs.
const (
	testS3Bucket  = "connector-s3-canary-prod"
	testS3RoleARN = "arn:aws:iam::142548018081:role/connector-s3-canary-terraform-provider-customer-role-prod"
)

func TestAccConnectorS3Resource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing with CSV
			{
				Config: testAccConnectorS3ResourceConfigCSV(testS3Bucket, "*.csv", "test_table"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "bucket", testS3Bucket),
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
				Config: testAccConnectorS3ResourceConfigCSV(testS3Bucket, "data/*.csv", "test_table"),
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
				Config: testAccConnectorS3ResourceConfigParquet(testS3Bucket, "*.parquet", "parquet_table"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "bucket", testS3Bucket),
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
    type     = "RoleARN"
    role_arn = %[4]q
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
`, bucket, pattern, tableName, testS3RoleARN)
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
    type     = "RoleARN"
    role_arn = %[4]q
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
`, bucket, pattern, tableName, testS3RoleARN)
}

func TestAccConnectorS3ResourceMinimal(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing with minimal configuration
			{
				Config: testAccConnectorS3ResourceConfigMinimal(testS3Bucket, "*.csv", "minimal_table"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "bucket", testS3Bucket),
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
				Config: testAccConnectorS3ResourceConfigFull(testS3Bucket, "data/*.csv", "full_table"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "name", "full-config-connector"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "bucket", testS3Bucket),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "pattern", "data/*.csv"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "frequency", "@30minutes"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "on_conflict_do_nothing", "true"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "enabled", "false"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "credentials.type", "RoleARN"),
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "credentials.role_arn", testS3RoleARN),
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
				Config: testAccConnectorS3ResourceConfigFullEnabled(testS3Bucket, "data/*.csv", "full_table"),
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
    type     = "RoleARN"
    role_arn = %[4]q
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
`, bucket, pattern, tableName, testS3RoleARN)
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
    role_arn = %[4]q
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
`, bucket, pattern, tableName, testS3RoleARN)
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
    role_arn = %[4]q
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
`, bucket, pattern, tableName, testS3RoleARN)
}

func TestAccConnectorS3ResourceColumnMapping(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConnectorS3ResourceConfigColumnMapping(testS3Bucket, "data/*.csv", "mapped_table"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "bucket", testS3Bucket),
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
    type     = "RoleARN"
    role_arn = %[4]q
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
`, bucket, pattern, tableName, testS3RoleARN)
}

func TestAccConnectorS3ResourceImport(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConnectorS3ResourceConfigCSV(testS3Bucket, "*.csv", "import_table"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_connector_s3.test", "bucket", testS3Bucket),
					resource.TestCheckResourceAttrSet("timescale_connector_s3.test", "id"),
					resource.TestCheckResourceAttrSet("timescale_connector_s3.test", "service_id"),
				),
			},
			{
				ResourceName:      "timescale_connector_s3.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"updated_at",
				},
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					rs, ok := state.RootModule().Resources["timescale_connector_s3.test"]
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
