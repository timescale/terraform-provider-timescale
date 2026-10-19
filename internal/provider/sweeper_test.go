package provider

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	tsClient "github.com/timescale/terraform-provider-timescale/internal/client"
	"log"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	resource.TestMain(m)
}

func init() {
	resource.AddTestSweepers("timescale_services", &resource.Sweeper{
		Name: "timescale_services",
		F:    sweepServices,
	})
	resource.AddTestSweepers("timescale_vpcs", &resource.Sweeper{
		Name: "timescale_vpcs",
		// Services may be attached to a test VPC, so delete them first.
		Dependencies: []string{"timescale_services"},
		F:            sweepVPCs,
	})
}

// sweepServices finds and deletes any services named with the acceptance test prefix,
// including read replicas that took the provider's default "replica-<primary>" name.
func sweepServices(_ string) error {
	c, err := createSweepClient()
	if err != nil {
		return fmt.Errorf("error creating client: %s", err)
	}

	ctx := context.Background()

	services, err := c.GetAllServices(ctx)
	if err != nil {
		return fmt.Errorf("error retrieving services: %s", err)
	}

	// Delete read replicas before the primaries they replicate from.
	for _, replicas := range []bool{true, false} {
		for _, service := range services {
			isReplica := service.ForkSpec != nil && service.ForkSpec.IsStandby
			if isReplica != replicas {
				continue
			}
			if !strings.HasPrefix(service.Name, testResourcePrefix) && !strings.HasPrefix(service.Name, "replica-"+testResourcePrefix) {
				continue
			}
			log.Printf("Destroying service %s (%s)", service.Name, service.ID)
			if _, err := c.DeleteService(ctx, service.ID); err != nil && !errors.Is(err, tsClient.ErrServiceNotFound) {
				log.Printf("Error deleting service %s (%s): %s", service.Name, service.ID, err)
			}
		}
	}

	return nil
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
