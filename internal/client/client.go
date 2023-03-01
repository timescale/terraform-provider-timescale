package client

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"time"

	"github.com/machinebox/graphql"
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
	Client    *graphql.Client
	apiToken   string
	projectID  string
	version    string
	terraformVersion string
}

type Response[T any] struct {
	Data   *T       `json:"data"`
	Errors []*Error `json:"errors"`
}

type Error struct {
	Message string `json:"message"`
}

func NewClient(apiToken, projectID, env, terraformVersion string) *Client {
	url := getURL(env)

	client := graphql.NewClient(url)

	return &Client{
		Client: client,
		apiToken:   apiToken,
		projectID:  projectID,
		version: 	env,
		terraformVersion:terraformVersion,
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

func (c *Client) do(ctx context.Context, request *graphql.Request, resp interface{}) error {
	tflog.Trace(ctx, "Client.do")

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	request.Header.Set("authorization", "Bearer "+c.apiToken)
	request.Header.Set("content-type", "application/json")

	userAgent := request.Header.Get("user-agent")
	// add provider and client terraform version
	userAgent = userAgent + " terraform-provider-timescale/"+c.version 
	userAgent = userAgent + " terraform/"+c.terraformVersion 
	request.Header.Set("user-agent", userAgent)
	err := c.Client.Run(ctx, request, &resp)
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("The HTTP request failed with error %s\n", err))
		return err
	}
	return nil
}
