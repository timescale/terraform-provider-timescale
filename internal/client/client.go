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
	"time"

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
)

type Client struct {
	httpClient       *http.Client
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

func NewClient(token, projectID, env, terraformVersion string) *Client {
	c := &http.Client{
		Timeout: 30 * time.Second,
	}

	url := getURL(env)

	return &Client{
		httpClient:       c,
		token:            token,
		projectID:        projectID,
		url:              url,
		version:          env,
		terraformVersion: terraformVersion,
	}
}

func getURL(_ string) string {
	if value, ok := os.LookupEnv("TIMESCALE_DEV_URL"); ok {
		return value
	}
	return "https://console.cloud.timescale.com/api/query"
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
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewBuffer(jsonValue))
	if err != nil {
		return err
	}
	c.setRequestHeaders(request)

	response, err := c.httpClient.Do(request)
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("The HTTP request failed with error %s\n", err))
		return err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("The HTTP request failed with error %s\n", err))
		return err
	}

	// Check HTTP status code before attempting to parse JSON
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		tflog.Error(ctx, fmt.Sprintf("HTTP request returned status code %d", response.StatusCode))

		// Try to parse response body to get more context
		bodyPreview := string(data)
		if len(bodyPreview) > 500 {
			bodyPreview = bodyPreview[:500] + "..."
		}

		return fmt.Errorf("HTTP request failed with status code %d: %s", response.StatusCode, bodyPreview)
	}

	// Parse JSON response
	if err := json.Unmarshal(data, resp); err != nil {
		tflog.Error(ctx, fmt.Sprintf("Failed to parse JSON response: %s", err))

		// Include response body preview for debugging
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
