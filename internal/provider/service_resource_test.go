package provider

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestServiceResource_Default_Success(t *testing.T) {
	// Test resource creation succeeds
	config := &ServiceConfig{
		Name:         "test-default",
		ResourceName: "resource",
	}
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Create default and Read testing
			{
				Config: getServiceConfig(t, config),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify the name is set.
					resource.TestCheckResourceAttrSet("timescale_service.resource", "name"),
					// Verify ID value is set in state.
					resource.TestCheckResourceAttrSet("timescale_service.resource", "id"),
					resource.TestCheckResourceAttrSet("timescale_service.resource", "password"),
					resource.TestCheckResourceAttrSet("timescale_service.resource", "hostname"),
					resource.TestCheckResourceAttrSet("timescale_service.resource", "username"),
					resource.TestCheckResourceAttrSet("timescale_service.resource", "port"),
					resource.TestCheckResourceAttr("timescale_service.resource", "milli_cpu", "500"),
					resource.TestCheckResourceAttr("timescale_service.resource", "memory_gb", "2"),
					resource.TestCheckResourceAttr("timescale_service.resource", "region_code", "us-east-1"),
					resource.TestCheckResourceAttr("timescale_service.resource", "ha_replicas", "0"),
					resource.TestCheckResourceAttr("timescale_service.resource", "sync_replicas", "0"),
					resource.TestCheckResourceAttr("timescale_service.resource", "connection_pooler_enabled", "false"),
					resource.TestCheckResourceAttr("timescale_service.resource", "environment_tag", "DEV"),
					resource.TestCheckNoResourceAttr("timescale_service.resource", "vpc_id"),
				),
			},
			// Do a compute resize
			{
				Config: getServiceConfig(t, config.WithSpec(1000, 4)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_service.resource", "milli_cpu", "1000"),
					resource.TestCheckResourceAttr("timescale_service.resource", "memory_gb", "4"),
				),
			},
			// Update service name
			{
				Config: getServiceConfig(t, config.WithName("service resource test update")),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_service.resource", "name", "service resource test update"),
				),
			},
			// Update tag
			{
				Config: getServiceConfig(t, config.WithEnvironment("PROD")),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_service.resource", "environment_tag", "PROD"),
				),
			},
			// Enable pooler
			{
				Config: getServiceConfig(t, config.WithPooler(true)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_service.resource", "connection_pooler_enabled", "true"),
					resource.TestCheckResourceAttrSet("timescale_service.resource", "pooler_hostname"),
					resource.TestCheckResourceAttrSet("timescale_service.resource", "pooler_port"),
				),
			},
			// Enable HA replica (deprecated but still maintained for backwards compatibility)
			{
				Config: getServiceConfig(t, config.WithEnableHAReplica(true).WithHAReplicasCount(1)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_service.resource", "enable_ha_replica", "true"),
					resource.TestCheckResourceAttr("timescale_service.resource", "ha_replicas", "1"),
					resource.TestCheckResourceAttrSet("timescale_service.resource", "replica_hostname"),
					resource.TestCheckResourceAttrSet("timescale_service.resource", "replica_port"),
				),
			},
			// Enable 1 replicas (1 async)
			{
				Config: getServiceConfig(t, config.WithHAReplicasAndSync(2, 1)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_service.resource", "ha_replicas", "1"),
					resource.TestCheckResourceAttr("timescale_service.resource", "sync_replicas", "0"),
				),
			},
			// Enable 2 replicas (2 async)
			{
				Config: getServiceConfig(t, config.WithHAReplicasAndSync(2, 1)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_service.resource", "ha_replicas", "2"),
					resource.TestCheckResourceAttr("timescale_service.resource", "sync_replicas", "0"),
				),
			},
			// Enable 2 replicas with 1 sync (1 sync + 1 async)
			{
				Config: getServiceConfig(t, config.WithHAReplicasAndSync(2, 1)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_service.resource", "ha_replicas", "2"),
					resource.TestCheckResourceAttr("timescale_service.resource", "sync_replicas", "1"),
				),
			},
		},
	})
}

func TestServiceResource_Read_Replica(t *testing.T) {
	t.Skipf("skip until fix")
	const (
		primaryName = "primary"
		extraName   = "extra"
		replicaName = "read_replica"
		primaryFQID = "timescale_service." + primaryName
		extraFQID   = "timescale_service." + extraName
		replicaFQID = "timescale_service." + replicaName
	)
	var (
		primaryConfig = &ServiceConfig{
			ResourceName: primaryName,
			Name:         "service resource test init",
		}
		extraConfig = &ServiceConfig{
			ResourceName: extraName,
		}
		replicaConfig = &ServiceConfig{
			ResourceName:      replicaName,
			ReadReplicaSource: primaryFQID + ".id",
			MilliCPU:          500,
			MemoryGB:          2,
		}
		extraReplicaConfig = &ServiceConfig{
			ResourceName:      replicaName + "_2",
			ReadReplicaSource: primaryFQID + ".id",
		}
	)
	// Test creating a service with a read replica
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: getServiceConfig(t, primaryConfig, replicaConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify service attributes
					resource.TestCheckResourceAttr(primaryFQID, "name", "service resource test init"),
					resource.TestCheckResourceAttrSet(primaryFQID, "id"),
					resource.TestCheckResourceAttrSet(primaryFQID, "password"),
					resource.TestCheckResourceAttrSet(primaryFQID, "hostname"),
					resource.TestCheckResourceAttrSet(primaryFQID, "username"),
					resource.TestCheckResourceAttrSet(primaryFQID, "port"),
					resource.TestCheckResourceAttr(primaryFQID, "milli_cpu", "500"),
					resource.TestCheckResourceAttr(primaryFQID, "memory_gb", "2"),
					resource.TestCheckResourceAttr(primaryFQID, "region_code", "us-east-1"),
					resource.TestCheckResourceAttr(primaryFQID, "ha_replicas", "0"),
					resource.TestCheckResourceAttr(primaryFQID, "sync_replicas", "0"),
					resource.TestCheckNoResourceAttr(primaryFQID, "vpc_id"),

					// Verify read replica attributes
					resource.TestCheckResourceAttr(replicaFQID, "name", "replica-service resource test init"),
					resource.TestCheckResourceAttrSet(replicaFQID, "id"),
					resource.TestCheckResourceAttrSet(replicaFQID, "password"),
					resource.TestCheckResourceAttrSet(replicaFQID, "hostname"),
					resource.TestCheckResourceAttrSet(replicaFQID, "username"),
					resource.TestCheckResourceAttrSet(replicaFQID, "port"),
					resource.TestCheckResourceAttr(replicaFQID, "milli_cpu", "500"),
					resource.TestCheckResourceAttr(replicaFQID, "memory_gb", "2"),
					resource.TestCheckResourceAttr(replicaFQID, "region_code", "us-east-1"),
					resource.TestCheckResourceAttr(replicaFQID, "ha_replicas", "0"),
					resource.TestCheckResourceAttr(replicaFQID, "sync_replicas", "0"),
					resource.TestCheckResourceAttrSet(replicaFQID, "read_replica_source"),
					resource.TestCheckNoResourceAttr(replicaFQID, "vpc_id"),
				),
			},
			// Update replica name
			{
				Config: getServiceConfig(t, primaryConfig, replicaConfig.WithName("replica")),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(replicaFQID, "name", "replica"),
				),
			},
			// Test creating a second read replica
			{
				Config: getServiceConfig(t, primaryConfig, replicaConfig, extraReplicaConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(replicaFQID, "name", "replica"),
					resource.TestCheckResourceAttr(extraFQID, "name", extraReplicaConfig.Name),
				),
			},
			// Do a compute resize
			{
				Config: getServiceConfig(t, primaryConfig, replicaConfig.WithSpec(1000, 4)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(replicaFQID, "milli_cpu", "1000"),
					resource.TestCheckResourceAttr(replicaFQID, "memory_gb", "4"),
				),
			},
			// Check adding HA returns an error
			{
				Config:      getServiceConfig(t, primaryConfig, replicaConfig.WithEnableHAReplica(true)),
				ExpectError: regexp.MustCompile(errReplicaWithHA),
			},
			// Check removing read_replica_source returns an error
			{
				Config:      getServiceConfig(t, primaryConfig, replicaConfig.WithEnableHAReplica(false).WithReadReplica("")),
				ExpectError: regexp.MustCompile(errUpdateReplicaSource),
			},
			// Check changing read_replica_source returns an error
			{
				Config:      getServiceConfig(t, primaryConfig, extraConfig, replicaConfig.WithReadReplica(extraFQID+".id")),
				ExpectError: regexp.MustCompile(errUpdateReplicaSource),
			},
			// Check enabling read_replica_source returns an error
			{
				Config:      getServiceConfig(t, primaryConfig.WithReadReplica(extraFQID+".id"), extraConfig, replicaConfig.WithReadReplica(primaryFQID+".id")),
				ExpectError: regexp.MustCompile(errUpdateReplicaSource),
			},
			// Test creating a read replica from a read replica returns an error
			{
				Config:      getServiceConfig(t, primaryConfig, replicaConfig, extraReplicaConfig.WithReadReplica(replicaFQID+".id")),
				ExpectError: regexp.MustCompile(errReplicaFromFork),
			},
			// Remove Replica
			{
				Config: getServiceConfig(t, primaryConfig),
				Check: func(state *terraform.State) error {
					resources := state.RootModule().Resources
					if _, ok := resources[replicaFQID]; ok {
						return errors.New("expected replica to be deleted")
					}
					return nil
				},
			},
		},
	})
}

func TestServiceResource_HA_Validation(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: getServiceConfig(t, (&ServiceConfig{
					Name:         "test-ha-validation",
					ResourceName: "resource",
				}).WithHAReplicasAndSync(1, 1)),
				ExpectError: regexp.MustCompile("sync_replicas can only be 1 when ha_replicas = 2"),
			},
			{
				Config: getServiceConfig(t, (&ServiceConfig{
					Name:         "test-ha-conflict",
					ResourceName: "resource",
					//EnableHAReplica: false,
				}).WithHAReplicasCount(1).WithEnableHAReplica(false)),
				ExpectError: regexp.MustCompile("cannot set enable_ha_replica as false together with ha_replicas > 0"),
			},
			{
				Config: getServiceConfig(t, (&ServiceConfig{
					Name:         "test-ha-conflict-2",
					ResourceName: "resource",
				}).WithHAReplicasCount(0).WithEnableHAReplica(true)),
				ExpectError: regexp.MustCompile("cannot set enable_ha_replica as true together with ha_replicas = 0"),
			},
		},
	})
}

func TestServiceResource_With_Exporter(t *testing.T) {
	const serviceConfig = `
resource "timescale_metric_exporter" "test_datadog" {
  name   = "test-datadog"
  region = "us-east-1"
  datadog = {
    api_key = "test"
    site    = "datadoghq.com"
  }
}
resource "timescale_service" "resource" {
  name               = "test-metric-exporter"
  milli_cpu          = 1000
  memory_gb          = 4
  region_code        = "us-east-1"
  metric_exporter_id = timescale_metric_exporter.test_datadog.id
}
`
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: providerConfig + serviceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("timescale_service.resource", "id"),
					resource.TestCheckResourceAttrSet("timescale_service.resource", "metric_exporter_id"),
				),
			},
		},
	})
}

func TestServiceResource_Timeout(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: newServiceConfig(ServiceConfig{
					Name: "test-service-timeout",
					Timeouts: Timeouts{
						Create: "1s",
					},
				}),
				ExpectError: regexp.MustCompile(ErrCreateTimeout),
			},
		},
	})
}

func TestServiceResource_CustomConf(t *testing.T) {
	// Test resource creation succeeds and update is not allowed
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Invalid conf millicpu & memory invalid ratio
			{
				Config: newServiceCustomConfig("invalid", ServiceConfig{
					Name:     "test-service-conf",
					MilliCPU: 2000,
					MemoryGB: 2,
				}),
				ExpectError: regexp.MustCompile(ErrInvalidAttribute),
			},
			// Invalid conf storage invalid value
			{
				Config: newServiceCustomConfig("invalid", ServiceConfig{
					Name:     "test-service-conf",
					MilliCPU: 500,
					MemoryGB: 3,
				}),
				ExpectError: regexp.MustCompile(ErrInvalidAttribute),
			},
			// Invalid conf storage invalid region
			{
				Config: newServiceCustomConfig("invalid", ServiceConfig{
					RegionCode: "test-invalid-region",
				}),
				ExpectError: regexp.MustCompile(ErrInvalidAttribute),
			},
			// Create with custom conf and region
			{
				Config: newServiceCustomConfig("custom", ServiceConfig{
					Name:       "test-service-conf",
					RegionCode: "eu-central-1",
					MilliCPU:   1000,
					MemoryGB:   4,
					Password:   "test123456789",
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("timescale_service.custom", "name", "test-service-conf"),
					resource.TestCheckResourceAttr("timescale_service.custom", "password", "test123456789"),
					resource.TestCheckResourceAttr("timescale_service.custom", "region_code", "eu-central-1"),
					resource.TestCheckNoResourceAttr("timescale_service.custom", "vpc_id"),
				),
			},
		},
	})
}

func TestServiceResource_Import(t *testing.T) {
	config := newServiceConfig(ServiceConfig{Name: "test-import"})
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			// Create the service to import
			{
				Config: config,
			},
			// Import the resource. This step compares the resource attributes for "test" defined above with the imported resource
			// "test_import" defined in the config for this step. This check is done by specifying the ImportStateVerify configuration option.
			{
				Check: func(_ *terraform.State) error {
					time.Sleep(10 * time.Second)
					return nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					for name, r := range resources {
						if name == "timescale_service.resource" {
							return r.Primary.ID, nil
						}
					}
					return "", errors.New("import ID not found")
				},
				ResourceName: "timescale_service.resource_import",
				Config: config + ` 
				resource "timescale_service" "resource_import" {}
				`,
			},
			// Import the resource. This step compares the replica resource attributes for "test" defined above with the imported resource
			// "test_import" defined in the config for this step. This check is done by specifying the ImportStateVerify configuration option.
			{
				Check: func(_ *terraform.State) error {
					time.Sleep(10 * time.Second)
					return nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					for name, r := range resources {
						if name == "timescale_service.resource_replica.0" {
							return r.Primary.ID, nil
						}
					}
					return "", errors.New("import ID for replica not found")
				},
				ResourceName: "timescale_service.resource_replica_import[0]",
				Config: config + `
				resource "timescale_service" "resource_replica_import" {
				  count = 1
				}
				`,
			},
		},
	})
}

func newServiceConfig(config ServiceConfig) string {
	if config.Timeouts.Create == "" {
		config.Timeouts.Create = "15m"
	}
	return providerConfig + fmt.Sprintf(`
				resource "timescale_service" "resource" {
					name = %q
					timeouts = {
						create = %q
					}
				}
				resource "timescale_service" "resource_replica" {
				  count = 1
				  read_replica_source = timescale_service.resource.id
				  name                = "%s replica"
					timeouts = {
						create = %q
					}
				}`, config.Name, config.Timeouts.Create, config.Name, config.Timeouts.Create)
}

func newServiceCustomConfig(resourceName string, config ServiceConfig) string {
	if config.Timeouts.Create == "" {
		config.Timeouts.Create = "30m"
	}

	passwordLine := ""
	if config.Password != "" {
		passwordLine = fmt.Sprintf("\n\t\tpassword = %q", config.Password)
	}

	return providerConfig + fmt.Sprintf(`
		resource "timescale_service" "%s" {
			name = %q
			timeouts = {
				create = %q
			}
			milli_cpu  = %d
			memory_gb  = %d
			region_code = %q%s
		}`, resourceName, config.Name, config.Timeouts.Create, config.MilliCPU, config.MemoryGB, config.RegionCode, passwordLine)
}

type ServiceConfig struct {
	ResourceName      string
	Name              string
	Timeouts          Timeouts
	MilliCPU          int64
	MemoryGB          int64
	RegionCode        string
	EnableHAReplica   *bool
	HAReplicas        *int64
	SyncReplicas      *int64
	VpcID             int64
	ReadReplicaSource string
	Pooler            bool
	Environment       string
	Password          string
	MetricExporterID  string
	LogExporterID     string
}

type Timeouts struct {
	Create string
}

func (c *ServiceConfig) WithName(name string) *ServiceConfig {
	c.Name = name
	return c
}

func (c *ServiceConfig) WithMetricExporterID(id string) *ServiceConfig {
	c.MetricExporterID = id
	return c
}

func (c *ServiceConfig) WithLogExporterID(id string) *ServiceConfig {
	c.LogExporterID = id
	return c
}

func (c *ServiceConfig) WithEnvironment(name string) *ServiceConfig {
	c.Environment = name
	return c
}

func (c *ServiceConfig) WithSpec(milliCPU, memoryGB int64) *ServiceConfig {
	c.MilliCPU = milliCPU
	c.MemoryGB = memoryGB
	return c
}

func (c *ServiceConfig) WithVPC(id int64) *ServiceConfig {
	c.VpcID = id
	return c
}

func (c *ServiceConfig) WithEnableHAReplica(enableHAReplica bool) *ServiceConfig {
	c.EnableHAReplica = &enableHAReplica
	return c
}

func (c *ServiceConfig) WithHAReplicasAndSync(haReplicas, syncReplicas int64) *ServiceConfig {
	c.HAReplicas = &haReplicas
	c.SyncReplicas = &syncReplicas
	return c
}

func (c *ServiceConfig) WithHAReplicasCount(haReplicas int64) *ServiceConfig {
	c.HAReplicas = &haReplicas
	return c
}

func (c *ServiceConfig) WithSyncReplicas(syncReplicas int64) *ServiceConfig {
	c.SyncReplicas = &syncReplicas
	return c
}
func (c *ServiceConfig) WithPooler(pooler bool) *ServiceConfig {
	c.Pooler = pooler
	return c
}

func (c *ServiceConfig) WithReadReplica(source string) *ServiceConfig {
	c.ReadReplicaSource = source
	return c
}

func (c *ServiceConfig) String(t *testing.T) string {
	c.setDefaults()
	b := &strings.Builder{}
	write := func(format string, a ...any) {
		_, err := fmt.Fprintf(b, format, a...)
		require.NoError(t, err)
	}
	_, err := fmt.Fprintf(b, "\n\n resource timescale_service %q { \n", c.ResourceName)
	require.NoError(t, err)
	if c.Name != "" {
		write("name = %q \n", c.Name)
	}
	if c.ReadReplicaSource != "" {
		write("read_replica_source = %s \n", c.ReadReplicaSource)
	}
	if c.EnableHAReplica != nil {
		write("enable_ha_replica = %t \n", *c.EnableHAReplica)
	}
	if c.HAReplicas != nil {
		write("ha_replicas = %d \n", *c.HAReplicas)
	}
	if c.SyncReplicas != nil {
		write("sync_replicas = %d \n", *c.SyncReplicas)
	}
	if c.Pooler {
		write("connection_pooler_enabled = %t \n", c.Pooler)
	}
	if c.Environment != "" {
		write("environment_tag = %q \n", c.Environment)
	}
	if c.RegionCode != "" {
		write("region_code = %q \n", c.RegionCode)
	}
	if c.VpcID != 0 {
		write("vpc_id = %d \n", c.VpcID)
	}
	if c.MetricExporterID != "" {
		write("metric_exporter_id = %s \n", c.MetricExporterID)
	}
	if c.LogExporterID != "" {
		write("log_exporter_id = %s \n", c.LogExporterID)
	}
	write(`
			milli_cpu  = %d
			memory_gb  = %d
			timeouts = {
				create = %q
			}`+"\n",
		c.MilliCPU, c.MemoryGB, c.Timeouts.Create)
	write("}")
	return b.String()
}

func (c *ServiceConfig) setDefaults() {
	if c.MilliCPU == 0 {
		c.MilliCPU = 500
	}
	if c.MemoryGB == 0 {
		c.MemoryGB = 2
	}
	if c.Timeouts.Create == "" {
		c.Timeouts.Create = "10m"
	}
}

// getServiceConfig returns a configuration for a test step.
func getServiceConfig(t *testing.T, cfgs ...*ServiceConfig) string {
	res := strings.Builder{}
	res.WriteString(providerConfig)
	for _, cfg := range cfgs {
		res.WriteString(cfg.String(t))
	}
	return res.String()
}
