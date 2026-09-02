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
	resource.AddTestSweepers("timescale_privatelink_connection", &resource.Sweeper{
		Name: "timescale_privatelink_connection",
		F:    sweepPrivateLinkConnections,
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

func sweepPrivateLinkConnections(_ string) error {
	log.Printf("Sweeper starting for Private Link connections...")
	c, err := createSweepClient()
	if err != nil {
		return fmt.Errorf("error creating client: %s", err)
	}

	ctx := context.Background()

	connections, err := c.ListPrivateLinkConnections(ctx, "")
	if err != nil {
		return fmt.Errorf("error retrieving Private Link connections: %s", err)
	}

	for _, conn := range connections {
		if !strings.HasPrefix(conn.Name, "test-") {
			continue
		}
		log.Printf("Rejecting Private Link connection %s (%s)", conn.Name, conn.ConnectionID)
		if err := c.DeletePrivateLinkConnection(ctx, conn.ConnectionID); err != nil {
			log.Printf("Error rejecting Private Link connection %s: %s", conn.Name, err)
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
