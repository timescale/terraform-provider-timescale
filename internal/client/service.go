package client

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type Service struct {
	ID                string `json:"id"`
	ProjectID         string `json:"projectId"`
	Name              string `json:"name"`
	AutoscaleSettings struct {
		Enabled bool `json:"enabled"`
	} `json:"autoscaleSettings"`
	Status        string         `json:"status"`
	RegionCode    string         `json:"regionCode"`
	Paused        bool           `json:"paused"`
	ServiceSpec   ServiceSpec    `json:"spec"`
	Resources     []ResourceSpec `json:"resources"`
	Created       string         `json:"created"`
	ReplicaStatus string         `json:"replicaStatus"`
	VPCEndpoint   *VPCEndpoint   `json:"vpcEndpoint"`
	ForkSpec      *ForkSpec      `json:"forkedFromId"`
}

type ServiceSpec struct {
	Hostname       string `json:"hostname"`
	Username       string `json:"username"`
	Port           int64  `json:"port"`
	PoolerHostname string `json:"poolerHostName"`
	PoolerPort     int64  `json:"poolerPort"`
	PoolerEnabled  bool   `json:"connectionPoolerEnabled"`
}

type VPCEndpoint struct {
	Host  string `json:"host"`
	Port  int64  `json:"port"`
	VPCId string `json:"vpcId"`
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
	Name     string
	MilliCPU string
	MemoryGB string
	// StorageGB is used for forks, since the CreateServiceRequest expects a storage to be requested
	// and the fork instance should match the storage size of the primary.
	StorageGB    string
	RegionCode   string
	ReplicaCount string
	VpcID        int64
	ForkConfig   *ForkConfig

	EnableConnectionPooler bool
}

type ForkConfig struct {
	ProjectID string `json:"projectID"`
	ServiceID string `json:"serviceID"`
	IsStandby bool   `json:"isStandby"`
}

type ForkSpec struct {
	ProjectID string `json:"projectId"`
	ServiceID string `json:"serviceId"`
	IsStandby bool   `json:"isStandby"`
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

type GetAllServicesResponse struct {
	Services []*Service `json:"getAllServices"`
}

type DeleteServiceResponse struct {
	Service Service `json:"deleteService"`
}

type ToggleServiceResponse struct {
	Service Service `json:"toggleService"`
}

func (c *Client) CreateService(ctx context.Context, request CreateServiceRequest) (*CreateServiceResponse, error) {
	tflog.Trace(ctx, "Client.CreateService")
	if request.Name == "" {
		r, err := rand.Int(rand.Reader, big.NewInt(90000))
		if err != nil {
			return nil, err
		}
		request.Name = fmt.Sprintf("db-%d", 10000+r.Int64())
	}
	if request.StorageGB == "" {
		request.StorageGB = "50"
	}

	variables := map[string]any{
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
		"enableConnectionPooler": request.EnableConnectionPooler,
	}
	if request.VpcID > 0 {
		variables["vpcId"] = request.VpcID
	}
	if request.ForkConfig != nil {
		variables["forkConfig"] = request.ForkConfig
	}

	req := map[string]interface{}{
		"operationName": "CreateService",
		"query":         CreateServiceMutation,
		"variables":     variables,
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

func (c *Client) SetReplicaCount(ctx context.Context, serviceID string, replicaCount int) error {
	tflog.Trace(ctx, "Client.SetReplicaCount")

	req := map[string]interface{}{
		"operationName": "SetReplicaCount",
		"query":         SetReplicaCountMutation,
		"variables": map[string]any{
			"projectId":    c.projectID,
			"serviceId":    serviceID,
			"replicaCount": replicaCount,
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
				"storageGB": "0",
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

func (c *Client) GetAllServices(ctx context.Context) ([]*Service, error) {
	tflog.Trace(ctx, "Client.GetAllServices")
	req := map[string]interface{}{
		"operationName": "GetAllServices",
		"query":         GetAllServicesQuery,
		"variables": map[string]string{
			"projectId": c.projectID,
		},
	}
	var resp Response[GetAllServicesResponse]
	if err := c.do(ctx, req, &resp); err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, resp.Errors[0]
	}
	if resp.Data == nil {
		return nil, errors.New("no response found")
	}
	return resp.Data.Services, nil
}

func (c *Client) ToggleService(ctx context.Context, id, status string) (*Service, error) {
	tflog.Trace(ctx, "Client.ToggleService")
	req := map[string]interface{}{
		"operationName": "ToggleService",
		"query":         ToggleServiceMutation,
		"variables": map[string]string{
			"projectId": c.projectID,
			"serviceId": id,
			"status":    status,
		},
	}
	var resp Response[ToggleServiceResponse]
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

func (c *Client) ToggleConnectionPooler(ctx context.Context, serviceID string, enable bool) error {
	tflog.Trace(ctx, "Client.ToggleConnectionPooler")
	req := map[string]interface{}{
		"operationName": "ToggleConnectionPooler",
		"query":         ToggleConnectionPoolerMutation,
		"variables": map[string]any{
			"projectId": c.projectID,
			"serviceId": serviceID,
			"enable":    enable,
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
