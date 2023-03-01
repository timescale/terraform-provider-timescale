package client

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/machinebox/graphql"
)

type Service struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	AutoscaleSettings struct {
		Enabled bool `json:"enabled"`
	} `json:"autoscaleSettings"`
	Status      string         `json:"status"`
	RegionCode  string         `json:"regionCode"`
	ServiceSpec ServiceSpec    `json:"spec"`
	Resources   []ResourceSpec `json:"resources"`
	Created     string         `json:"created"`
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
	Name                     string
	EnableStorageAutoscaling bool
	MilliCPU                 string
	StorageGB                string
	MemoryGB                 string
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
	req := graphql.NewRequest(CreateServiceMutation)
	req.Var("projectId",                c.projectID)
	req.Var("name",               request.Name)
	req.Var("enable_storage_autoscaling",               request.EnableStorageAutoscaling)
	req.Var("type",              "TIMESCALEDB")
	conf := map[string]string{
		"milliCPU":     "500",
		"storageGB":    "10",
		"memoryGB":     "2",
		"replicaCount": "0",
	}
	req.Var("resourceConfig", conf )

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
	req := graphql.NewRequest(GetServiceQuery)
	req.Var("projectId",                c.projectID)
	req.Var("serviceId",               id)
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
	req := graphql.NewRequest(DeleteServiceMutation)
	req.Var("projectId",                c.projectID)
	req.Var("serviceId",               id)
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
