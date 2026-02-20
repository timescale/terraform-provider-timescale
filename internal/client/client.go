package client

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	//go:embed queries/create_service.graphql
	CreateServiceMutation string
	//go:embed queries/rename_service.graphql
	RenameServiceMutation string
	//go:embed queries/resize_instance.graphql
	ResizeInstanceMutation string
	//go:embed queries/delete_service.graphql
	DeleteServiceMutation string
	//go:embed queries/toggle_service.graphql
	ToggleServiceMutation string
	//go:embed queries/toggle_connection_pooler.graphql
	ToggleConnectionPoolerMutation string
	//go:embed queries/set_env_tag.graphql
	SetEnvironmentTagMutation string
	//go:embed queries/get_all_services.graphql
	GetAllServicesQuery string
	//go:embed queries/get_service.graphql
	GetServiceQuery string
	//go:embed queries/products.graphql
	ProductsQuery string
	//go:embed queries/jwt_cc.graphql
	JWTFromCCQuery string
	//go:embed queries/set_replica_count.graphql
	SetReplicaCountMutation string
	//go:embed queries/change_service_password.graphql
	ResetServicePassword string

	// VCPs
	//go:embed queries/vpcs.graphql
	GetVPCsQuery string
	//go:embed queries/vpc_by_name.graphql
	GetVPCByNameQuery string
	//go:embed queries/vpc_by_id.graphql
	GetVPCByIDQuery string
	//go:embed queries/attach_service_to_vpc.graphql
	AttachServiceToVPCMutation string
	//go:embed queries/detach_service_from_vpc.graphql
	DetachServiceFromVPCMutation string
	//go:embed queries/create_vpc.graphql
	CreateVPCMutation string
	//go:embed queries/delete_vpc.graphql
	DeleteVPCMutation string
	//go:embed queries/rename_vpc.graphql
	RenameVPCMutation string
	//go:embed queries/open_peer_request.graphql
	OpenPeerRequestMutation string
	//go:embed queries/delete_peer_request.graphql
	DeletePeeringConnectionMutation string
	//go:embed queries/update_peering_connection_cidrs.graphql
	UpdatePeeringConnectionCIDRsMutation string

	// Metric Exporters
	//go:embed queries/create_metric_exporter.graphql
	CreateMetricExporterMutation string
	//go:embed queries/get_all_metric_exporters.graphql
	GetAllMetricExportersQuery string
	//go:embed queries/delete_metric_exporter.graphql
	DeleteMetricExporterMutation string
	//go:embed queries/update_metric_exporter.graphql
	UpdateMetricExporterMutation string

	// Generic Exporters (logs)
	//go:embed queries/create_generic_exporter.graphql
	CreateGenericExporterMutation string
	//go:embed queries/get_all_generic_exporters.graphql
	GetAllGenericExportersQuery string
	//go:embed queries/delete_generic_exporter.graphql
	DeleteGenericExporterMutation string
	//go:embed queries/update_generic_exporter.graphql
	UpdateGenericExporterMutation string

	// Exporters attachment
	//go:embed queries/attach_metric_exporter.graphql
	AttachMetricExporterMutation string
	//go:embed queries/detach_metric_exporter.graphql
	DetachMetricExporterMutation string
	//go:embed queries/attach_generic_exporter.graphql
	AttachGenericExporterMutation string
	//go:embed queries/detach_generic_exporter.graphql
	DetachGenericExporterMutation string

	// S3 Connectors
	//go:embed queries/connector_s3_create.graphql
	CreateS3ConnectorMutation string
	//go:embed queries/connector_s3_update.graphql
	UpdateS3ConnectorMutation string
	//go:embed queries/connector_s3_get.graphql
	GetS3ConnectorQuery string
	//go:embed queries/connector_s3_delete.graphql
	DeleteS3ConnectorMutation string

	// Private Link
	//go:embed queries/list_private_link_bindings.graphql
	ListPrivateLinkBindingsQuery string
	//go:embed queries/attach_service_to_private_link.graphql
	AttachServiceToPrivateLinkConnectionMutation string
	//go:embed queries/detach_service_from_private_link.graphql
	DetachServiceFromPrivateLinkConnectionMutation string
	//go:embed queries/list_private_link_connections.graphql
	ListPrivateLinkConnectionsQuery string
	//go:embed queries/sync_private_link_connections.graphql
	SyncPrivateLinkConnectionsMutation string
	//go:embed queries/update_private_link_connection.graphql
	UpdatePrivateLinkConnectionMutation string
	//go:embed queries/list_private_link_available_regions.graphql
	ListPrivateLinkAvailableRegionsQuery string
	//go:embed queries/delete_private_link_connection.graphql
	DeletePrivateLinkConnectionMutation string
	//go:embed queries/list_private_link_authorizations.graphql
	ListPrivateLinkAuthorizationsQuery string
	//go:embed queries/create_private_link_authorization.graphql
	CreatePrivateLinkAuthorizationMutation string
	//go:embed queries/update_private_link_authorization.graphql
	UpdatePrivateLinkAuthorizationMutation string
	//go:embed queries/delete_private_link_authorization.graphql
	DeletePrivateLinkAuthorizationMutation string
)

type Client struct {
	httpClient       *http.Client
	retryClient      *retryablehttp.Client
	token            string
	projectID        string
	url              string
	version          string
	terraformVersion string
}

type Response[T any] struct {
	Data   *T       `json:"data"`
	Errors []*Error `json:"errors"`
}

type Error struct {
	Message string `json:"message"`
}

func NewClient(token, projectID, version, terraformVersion string) *Client {
	url := getURL()

	// Configure retryable HTTP client
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = getEnvInt("TIMESCALE_MAX_RETRIES", 5)
	retryClient.RetryWaitMin = time.Duration(getEnvInt("TIMESCALE_RETRY_WAIT_MIN_SEC", 1)) * time.Second
	retryClient.RetryWaitMax = time.Duration(getEnvInt("TIMESCALE_RETRY_WAIT_MAX_SEC", 30)) * time.Second
	retryClient.HTTPClient.Timeout = 30 * time.Second

	// Disable default logging to avoid noise
	retryClient.Logger = nil

	return &Client{
		httpClient:       retryClient.StandardClient(),
		retryClient:      retryClient,
		token:            token,
		projectID:        projectID,
		url:              url,
		version:          version,
		terraformVersion: terraformVersion,
	}
}

func getURL() string {
	if value, ok := os.LookupEnv("TIMESCALE_DEV_URL"); ok {
		return value
	}
	return "https://console.cloud.timescale.com/api/query"
}

func getEnvInt(key string, defaultValue int) int {
	if value, ok := os.LookupEnv(key); ok {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

type JWTFromCCResponse struct {
	Token string `json:"getJWTForClientCredentials"`
}

func JWTFromCC(c *Client, accessKey, secretKey string) error {
	req := map[string]interface{}{
		"operationName": "GetJWTForClientCredentials",
		"query":         JWTFromCCQuery,
		"variables": map[string]any{
			"accessKey": accessKey,
			"secretKey": secretKey,
		},
	}
	var resp Response[JWTFromCCResponse]

	if err := c.do(context.Background(), req, &resp); err != nil {
		return err
	}
	if len(resp.Errors) > 0 {
		return resp.Errors[0]
	}
	if resp.Data == nil {
		return errors.New("no response found")
	}
	c.token = resp.Data.Token
	return nil
}

func (e *Error) Error() string {
	return e.Message
}

func (c *Client) do(ctx context.Context, req map[string]interface{}, resp interface{}) error {
	tflog.Trace(ctx, "Client.do")
	jsonValue, err := json.Marshal(req)
	if err != nil {
		return err
	}

	// Create retryable request
	retryableReq, err := retryablehttp.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewBuffer(jsonValue))
	if err != nil {
		return err
	}
	c.setRequestHeaders(retryableReq.Request)

	// Execute with automatic retries
	response, err := c.retryClient.Do(retryableReq)
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("The HTTP request failed with error %s\n", err))
		return err
	}
	defer response.Body.Close()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("Failed to read response body: %s\n", err))
		return err
	}

	// Check HTTP status code before attempting to parse JSON
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		tflog.Error(ctx, fmt.Sprintf("HTTP request returned status code %d", response.StatusCode))

		bodyPreview := string(data)
		if len(bodyPreview) > 500 {
			bodyPreview = bodyPreview[:500] + "..."
		}

		return fmt.Errorf("HTTP request failed with status code %d: %s", response.StatusCode, bodyPreview)
	}

	// Parse JSON response
	if err := json.Unmarshal(data, resp); err != nil {
		tflog.Error(ctx, fmt.Sprintf("Failed to parse JSON response: %s", err))

		bodyPreview := string(data)
		if len(bodyPreview) > 500 {
			bodyPreview = bodyPreview[:500] + "..."
		}

		return fmt.Errorf("failed to parse JSON response: %w. Response body: %s", err, bodyPreview)
	}

	return nil
}

func (c *Client) setRequestHeaders(request *http.Request) {
	if c.token != "" {
		request.Header.Set("Authorization", "Bearer "+c.token)
	}
	request.Header.Set("Content-Type", "application/json")

	userAgent := request.UserAgent()
	// add provider and client terraform version
	userAgent = userAgent + " terraform-provider-timescale/" + c.version
	userAgent = userAgent + " terraform/" + c.terraformVersion
	request.Header.Set("User-Agent", userAgent)
}

func (c *Client) GetProjectID() string {
	return c.projectID
}
