package client

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type Service struct {
	ID                       string `json:"id"`
	Name                     string `json:"name"`
	EnableStorageAutoscaling bool   `json:"enable_storage_autoscaling"`
	Status                   string `json:"status"`
}

type CreateServiceRequest struct {
	Name                     string
	EnableStorageAutoscaling bool
}

type CreateServiceResponse struct {
	CreateService struct {
		Service Service `json:"service"`
	} `json:"createService"`
}

type GetServiceResponse struct {
	Service Service `json:"getService"`
}

type DeleteServiceResponse struct {
	Service Service `json:"deleteService"`
}

func (c *Client) CreateService(ctx context.Context, request CreateServiceRequest) (*Service, error) {
	tflog.Trace(ctx, "Client.CreateService")
	if request.Name == "" {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		request.Name = fmt.Sprintf("db-%d", 10000+r.Intn(90000))
	}

	req := map[string]interface{}{
		"operationName": "CreateService",
		"query":         CreateServiceMutation,
		"variables": map[string]any{
			"projectId":                  c.projectID,
			"name":                       request.Name,
			"enable_storage_autoscaling": request.EnableStorageAutoscaling,
			"type":                       "TIMESCALEDB",
			"resourceConfig": map[string]string{
				"milliCPU":     "500",
				"storageGB":    "10",
				"memoryGB":     "2",
				"replicaCount": "0",
			},
		},
	}
	var resp Response[CreateServiceResponse]
	if err := c.do(ctx, req, &resp); err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, resp.Errors[0]
	}
	if resp.Data == nil {
		return nil, errors.New("no response found")
	}
	return &resp.Data.CreateService.Service, nil
}

func (c *Client) GetService(ctx context.Context, id string) (*Service, error) {
	tflog.Trace(ctx, "Client.GetService")
	req := map[string]interface{}{
		"operationName": "GetService",
		"query":         GetServiceQuery,
		"variables": map[string]string{
			"projectId": c.projectID,
			"serviceId": id,
		},
	}
	var resp Response[GetServiceResponse]
	if err := c.do(ctx, req, &resp); err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, resp.Errors[0]
	}
	if resp.Data == nil {
		return nil, errors.New("no response found")
	}
	return &resp.Data.Service, nil
}

func (c *Client) DeleteService(ctx context.Context, id string) (*Service, error) {
	tflog.Trace(ctx, "Client.DeleteService")
	req := map[string]interface{}{
		"operationName": "DeleteService",
		"query":         DeleteServiceMutation,
		"variables": map[string]string{
			"projectId": c.projectID,
			"serviceId": id,
		},
	}
	var resp Response[DeleteServiceResponse]
	if err := c.do(ctx, req, &resp); err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, resp.Errors[0]
	}
	if resp.Data == nil {
		return nil, errors.New("no response found")
	}
	return &resp.Data.Service, nil
}
