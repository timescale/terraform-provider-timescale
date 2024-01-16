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

type VPCConfig struct {
	ResourceName string
	Name         string
	CIDR         string
	RegionCode   string
}

func (vc *VPCConfig) WithName(s string) *VPCConfig {
	vc.Name = s
	return vc
}
func (vc *VPCConfig) WithCIDR(s string) *VPCConfig {
	vc.CIDR = s
	return vc
}
func (vc *VPCConfig) WithRegionCode(s string) *VPCConfig {
	vc.RegionCode = s
	return vc
}

func (vc *VPCConfig) String(t *testing.T) string {
	b := &strings.Builder{}
	write := func(format string, a ...any) {
		_, err := fmt.Fprintf(b, format, a...)
		require.NoError(t, err)
	}
	_, err := fmt.Fprintf(b, "\n\n resource timescale_vpcs %q { \n", vc.ResourceName)
	require.NoError(t, err)
	if vc.Name != "" {
		write("name = %q \n", vc.Name)
	}
	if vc.CIDR != "" {
		write("cidr = %q \n", vc.CIDR)
	}
	if vc.RegionCode != "" {
		write("region_code = %q \n", vc.RegionCode)
	}
	write("}")
	return b.String()
}

// getServiceConfig returns a configuration for a test step
func getVPCConfig(t *testing.T, cfgs ...*VPCConfig) string {
	res := strings.Builder{}
	res.WriteString(providerConfig)
	for _, cfg := range cfgs {
		res.WriteString(cfg.String(t))
	}
	return res.String()
}

type ServiceConfig struct {
	ResourceName      string
	Name              string
	Timeouts          Timeouts
	MilliCPU          int64
	MemoryGB          int64
	RegionCode        string
	EnableHAReplica   bool
	VpcID             int64
	ReadReplicaSource string
}

func (c *ServiceConfig) WithName(name string) *ServiceConfig {
	c.Name = name
	return c
}

func (c *ServiceConfig) WithSpec(milliCPU, memoryGB int64) *ServiceConfig {
	c.MilliCPU = milliCPU
	c.MemoryGB = memoryGB
	return c
}

func (c *ServiceConfig) WithVPC(ID int64) *ServiceConfig {
	c.VpcID = ID
	return c
}

func (c *ServiceConfig) WithHAReplica(enableHAReplica bool) *ServiceConfig {
	c.EnableHAReplica = enableHAReplica
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

// getServiceConfig returns a configuration for a test step
func getServiceConfig(t *testing.T, cfgs ...*ServiceConfig) string {
	res := strings.Builder{}
	res.WriteString(providerConfig)
	for _, cfg := range cfgs {
		res.WriteString(cfg.String(t))
	}
	return res.String()
}

// // getServiceConfig returns a configuration for a test step
// func getServiceNoProviderConfig(t *testing.T, cfgs ...*ServiceConfig) string {
// 	res := strings.Builder{}
// 	for _, cfg := range cfgs {
// 		res.WriteString(cfg.String(t))
// 	}
// 	return res.String()
// }

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
