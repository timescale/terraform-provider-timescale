package provider

import (
	"log"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
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

var (
	peerAccountID string
	peerVPCID     string
	peerTGWID     string
	peerRegion    string
)

func init() {
	var ok bool
	peerAccountID, ok = os.LookupEnv("PEER_ACCOUNT_ID")
	if !ok {
		log.Fatal("environment variable PEER_ACCOUNT_ID not set")
	}
	peerVPCID, ok = os.LookupEnv("PEER_VPC_ID")
	if !ok {
		log.Fatal("environment variable PEER_VPC_ID not set")
	}
	peerTGWID, ok = os.LookupEnv("PEER_TGW_ID")
	if !ok {
		log.Fatal("environment variable PEER_TGW_ID not set")
	}
	peerRegion, ok = os.LookupEnv("PEER_REGION")
	if !ok {
		log.Fatal("environment variable PEER_REGION not set")
	}
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
}
