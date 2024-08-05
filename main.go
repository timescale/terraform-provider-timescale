package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/timescale/terraform-provider-timescale/internal/provider"
)

//go:generate terraform fmt -recursive ./examples/
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs
var (
	Version = "dev"
	debug   bool
	version bool
)

func init() {
	flag.BoolVar(
		&debug,
		"debug",
		false,
		"set to true to run the provider with support for debuggers like delve",
	)
	flag.BoolVar(
		&version,
		"version",
		false,
		"display the provider version",
	)
	flag.Parse()

	if version {
		displayVersion()
		os.Exit(0)
	}
}

func displayVersion() {
	log.Printf("Timescale provider v%s\n", Version)
}

func main() {
	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/timescale/timescale",
		Debug:   debug,
	}

	displayVersion()
	err := providerserver.Serve(context.Background(), provider.New(Version), opts)
	if err != nil {
		log.Fatal(err.Error())
	}
}
