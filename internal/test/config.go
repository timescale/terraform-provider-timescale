package test

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/timescale/terraform-provider-timescale/internal/provider"
)

const (
	// ProviderConfig is a shared configuration to combine with the actual
	// test configuration so the Timescale client is properly configured.
	ProviderConfig = `
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

func TestAccPreCheck(t *testing.T) {
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

// TestAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var TestAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"timescale": providerserver.NewProtocol6WithError(provider.New("test")()),
}
