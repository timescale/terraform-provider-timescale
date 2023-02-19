package client

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
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
	//go:embed queries/delete_service.graphql
	DeleteServiceMutation string
	//go:embed queries/get_service.graphql
	GetServiceQuery string
	//go:embed queries/products.graphql
	ProductsQuery string
)

type Client struct {
	httpClient *http.Client
	apiToken   string
	projectID  string
	url        string
}

type Response[T any] struct {
	Data   *T       `json:"data"`
	Errors []*Error `json:"errors"`
}

type Error struct {
	Message string `json:"message"`
}

func NewClient(apiToken, projectID, env string) *Client {
	c := &http.Client{
		Timeout: 5 * time.Second,
	}

	url := getURL(env)

	return &Client{
		httpClient: c,
		apiToken:   apiToken,
		projectID:  projectID,
		url:        url,
	}
}

func getURL(env string) string {
	url := "https://console.cloud.timescale.com/api/query"
	if env != "test" {
		return url
	}
	// This environment variable is used to configure the client for testing.
	value, ok := os.LookupEnv("TIMESCALE_DEV_URL")
	if !ok {
		return url
	}
	return value
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
	request.Header.Set("Authorization", "Bearer "+c.apiToken)
	request.Header.Set("Content-Type", "application/json")
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
	if err = json.Unmarshal(data, resp); err != nil {
		return err
	}
	return nil
}
