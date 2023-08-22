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
	ID                string `json:"id"`
	Name              string `json:"name"`
	AutoscaleSettings struct {
		Enabled bool `json:"enabled"`
	} `json:"autoscaleSettings"`
	Status        string         `json:"status"`
	RegionCode    string         `json:"regionCode"`
	ServiceSpec   ServiceSpec    `json:"spec"`
	Resources     []ResourceSpec `json:"resources"`
	Created       string         `json:"created"`
	ReplicaStatus string         `json:"replicaStatus"`
}

type ServiceSpec struct {
	Hostname string `json:"hostname"`
	Username string `json:"username"`
	Port     int64  `json:"port"`
}

type ResourceSpec struct {
	ID   string `json:"id"`
	Spec struct {
		MilliCPU  int64 `json:"milliCPU"`
		MemoryGB  int64 `json:"memoryGB"`
		StorageGB int64 `json:"storageGB"`
	} `json:"spec"`
}

type CreateServiceRequest struct {
	Name         string
	MilliCPU     string
	StorageGB    string
	MemoryGB     string
	RegionCode   string
	ReplicaCount string
}

type CreateServiceResponseData struct {
	CreateServiceResponse CreateServiceResponse `json:"createService"`
}

type CreateServiceResponse struct {
	Service         Service `json:"service"`
	InitialPassword string  `json:"initialPassword"`
}

type GetServiceResponse struct {
	Service Service `json:"getService"`
}

type DeleteServiceResponse struct {
	Service Service `json:"deleteService"`
}

func (c *Client) CreateService(ctx context.Context, request CreateServiceRequest) (*CreateServiceResponse, error) {
	tflog.Trace(ctx, "Client.CreateService")
	if request.Name == "" {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		request.Name = fmt.Sprintf("db-%d", 10000+r.Intn(90000))
	}

	req := map[string]interface{}{
		"operationName": "CreateService",
		"query":         CreateServiceMutation,
		"variables": map[string]any{
			"projectId":  c.projectID,
			"name":       request.Name,
			"type":       "TIMESCALEDB",
			"regionCode": request.RegionCode,
			"resourceConfig": map[string]string{
				"milliCPU":     request.MilliCPU,
				"storageGB":    request.StorageGB,
				"memoryGB":     request.MemoryGB,
				"replicaCount": request.ReplicaCount,
			},
		},
	}
	var resp Response[CreateServiceResponseData]
	if err := c.do(ctx, req, &resp); err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, resp.Errors[0]
	}
	if resp.Data == nil {
		return nil, errors.New("no response found")
	}
	return &resp.Data.CreateServiceResponse, nil
}

func (c *Client) RenameService(ctx context.Context, serviceID string, newName string) error {
	tflog.Trace(ctx, "Client.RenameService")

	req := map[string]interface{}{
		"operationName": "RenameService",
		"query":         RenameServiceMutation,
		"variables": map[string]any{
			"projectId": c.projectID,
			"serviceId": serviceID,
			"newName":   newName,
		},
	}
	var resp Response[any]
	if err := c.do(ctx, req, &resp); err != nil {
		return err
	}
	if len(resp.Errors) > 0 {
		return resp.Errors[0]
	}
	if resp.Data == nil {
		return errors.New("no response found")
	}
	return nil
}

type ResourceConfig struct {
	MilliCPU     string
	StorageGB    string
	MemoryGB     string
	ReplicaCount string
}

func (c *Client) ResizeInstance(ctx context.Context, serviceID string, config ResourceConfig) error {
	tflog.Trace(ctx, "Client.ResizeInstance")

	req := map[string]interface{}{
		"operationName": "ResizeInstance",
		"query":         ResizeInstanceMutation,
		"variables": map[string]any{
			"projectId": c.projectID,
			"serviceId": serviceID,
			"config": map[string]string{
				"milliCPU":  config.MilliCPU,
				"storageGB": config.StorageGB,
				"memoryGB":  config.MemoryGB,
			},
		},
	}
	var resp Response[any]
	if err := c.do(ctx, req, &resp); err != nil {
		return err
	}
	if len(resp.Errors) > 0 {
		return resp.Errors[0]
	}
	if resp.Data == nil {
		return errors.New("no response found")
	}
	return nil
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
