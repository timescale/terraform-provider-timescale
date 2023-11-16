package provider

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/stretchr/testify/require"
)

const (
	// providerConfig is a shared configuration to combine with the actual
	// test configuration so the Timescale client is properly configured.
	providerConfig = `
variable "ts_access_key" {
	type = string
}
	
variable "ts_secret_key" {
	type = string
}
	
variable "ts_project_id" {
	type = string
}
	
provider "timescale" {
	access_key = var.ts_access_key
	secret_key = var.ts_secret_key
	project_id = var.ts_project_id
}	  
`
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"timescale": providerserver.NewProtocol6WithError(New("test")()),
}

type Config struct {
	ResourceName    string
	Name            string
	Timeouts        Timeouts
	MilliCPU        int64
	MemoryGB        int64
	RegionCode      string
	EnableHAReplica bool
	VpcID           int64
}

func (c *Config) WithName(name string) *Config {
	c.Name = name
	return c
}

func (c *Config) WithSpec(milliCPU, memoryGB int64) *Config {
	c.MilliCPU = milliCPU
	c.MemoryGB = memoryGB
	return c
}

func (c *Config) WithVPC(ID int64) *Config {
	c.VpcID = ID
	return c
}

func (c *Config) WithHAReplica(enableHAReplica bool) *Config {
	c.EnableHAReplica = enableHAReplica
	return c
}

func (c *Config) String(t *testing.T) string {
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
	if c.EnableHAReplica {
		write("enable_ha_replica = %t \n", c.EnableHAReplica)
	}
	if c.RegionCode != "" {
		write("region_code = %q \n", c.RegionCode)
	}
	if c.VpcID != 0 {
		write("vpc_id = %d \n", c.VpcID)
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

func (c *Config) setDefaults() {
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

// getConfig returns a configuration for a test step
func getConfig(t *testing.T, cfgs ...*Config) string {
	res := strings.Builder{}
	res.WriteString(providerConfig)
	for _, cfg := range cfgs {
		res.WriteString(cfg.String(t))
	}
	return res.String()
}

type Timeouts struct {
	Create string
}

func testAccPreCheck(t *testing.T) {
	_, ok := os.LookupEnv("TF_VAR_ts_access_key")
	if !ok {
		t.Fatal("environment variable TF_VAR_ts_access_key not set")
	}
	_, ok = os.LookupEnv("TF_VAR_ts_secret_key")
	if !ok {
		t.Fatal("environment variable TF_VAR_ts_secret_key not set")
	}
	_, ok = os.LookupEnv("TF_VAR_ts_project_id")
	if !ok {
		t.Fatal("environment variable TF_VAR_ts_project_id not set")
	}
	_, ok = os.LookupEnv("TIMESCALE_DEV_URL")
	if !ok {
		t.Fatal("environment variable TIMESCALE_DEV_URL not set")
	}
}
