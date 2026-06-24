package provider

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	tsClient "github.com/timescale/terraform-provider-timescale/internal/client"
)

func TestMain(m *testing.M) {
	resource.TestMain(m)
}

func init() {
	resource.AddTestSweepers("timescale_vpcs", &resource.Sweeper{
		Name: "timescale_vpcs",
		F:    sweepVPCs,
	})
	resource.AddTestSweepers("timescale_privatelink_authorization", &resource.Sweeper{
		Name: "timescale_privatelink_authorization",
		F:    sweepPrivateLinkAuthorizations,
	})
}

// sweepVPCs finds and deletes any VPCs with the test prefix.
func sweepVPCs(_ string) error {
	log.Printf("Sweeper starting...")
	c, err := createSweepClient()
	if err != nil {
		return fmt.Errorf("error creating client: %s", err)
	}

	ctx := context.Background()

	vpcs, err := c.GetVPCs(ctx)
	if err != nil {
		return fmt.Errorf("error retrieving VPCs: %s", err)
	}

	// Filter and delete test VPCs
	for _, vpc := range vpcs {
		if strings.HasPrefix(vpc.Name, "test-vpc-") {
			log.Printf("Destroying VPC %s (%s)", vpc.Name, vpc.ID)
			vpcID, err := strconv.ParseInt(vpc.ID, 10, 64)
			if err != nil {
				log.Printf("Error parsing VPC ID %s: %s", vpc.ID, err)
				continue
			}

			if err := c.DeleteVPC(ctx, vpcID); err != nil {
				log.Printf("Error deleting VPC %s (%s): %s", vpc.Name, vpc.ID, err)
			}
		}
	}

	return nil
}

func sweepPrivateLinkAuthorizations(_ string) error {
	log.Printf("Sweeper starting for Private Link authorizations...")
	c, err := createSweepClient()
	if err != nil {
		return fmt.Errorf("error creating client: %s", err)
	}

	ctx := context.Background()

	authorizations, err := c.ListPrivateLinkAuthorizations(ctx)
	if err != nil {
		return fmt.Errorf("error retrieving Private Link authorizations: %s", err)
	}

	for _, auth := range authorizations {
		if strings.HasPrefix(auth.Name, "test-") || strings.HasPrefix(auth.Name, "Terraform managed") {
			log.Printf("Destroying Private Link authorization %s (principal=%s, provider=%s)", auth.Name, auth.PrincipalID, auth.CloudProvider)
			if err := c.DeletePrivateLinkAuthorization(ctx, auth.PrincipalID, auth.CloudProvider); err != nil {
				log.Printf("Error deleting Private Link authorization %s: %s", auth.Name, err)
			}
		}
	}

	return nil
}

// createSweepClient creates an API client for sweeper functions.
func createSweepClient() (*tsClient.Client, error) {
	accessKey, ok := os.LookupEnv("TF_VAR_ts_access_key")
	if !ok {
		return nil, errors.New("environment variable TF_VAR_ts_access_key not set")
	}
	secretKey, ok := os.LookupEnv("TF_VAR_ts_secret_key")
	if !ok {
		return nil, errors.New("environment variable TF_VAR_ts_secret_key not set")
	}
	projectID, ok := os.LookupEnv("TF_VAR_ts_project_id")
	if !ok {
		return nil, errors.New("environment variable TF_VAR_ts_project_id not set")
	}

	client := tsClient.NewClient("", projectID, "", "")
	err := tsClient.JWTFromCC(client, accessKey, secretKey)
	if err != nil {
		return nil, err
	}

	return client, nil
}
