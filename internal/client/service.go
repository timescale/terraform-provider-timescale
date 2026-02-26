package client

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type Service struct {
	ID            string         `json:"id"`
	ProjectID     string         `json:"projectId"`
	Name          string         `json:"name"`
	Status        string         `json:"status"`
	RegionCode    string         `json:"regionCode"`
	Paused        bool           `json:"paused"`
	ServiceSpec   ServiceSpec    `json:"spec"`
	Resources     []ResourceSpec `json:"resources"`
	Created       string         `json:"created"`
	ReplicaStatus string         `json:"replicaStatus"`
	VPCEndpoint   *VPCEndpoint   `json:"vpcEndpoint"`
	ForkSpec      *ForkSpec      `json:"forkedFromId"`
	Metadata      *Metadata      `json:"metadata"`

	// Endpoints contains the all service endpoints
	Endpoints *ServiceEndpoints `json:"endpoints,omitempty"`

	// PrivateLinkEndpointConnectionID is the ID of the private link connection the service is attached to
	PrivateLinkEndpointConnectionID *string `json:"privateLinkEndpointConnectionId"`
}

type ServiceSpec struct {
	Hostname           string  `json:"hostname"`
	Username           string  `json:"username"`
	Port               int64   `json:"port"`
	PoolerHostname     string  `json:"poolerHostName"`
	PoolerPort         int64   `json:"poolerPort"`
	PoolerEnabled      bool    `json:"connectionPoolerEnabled"`
	MetricExporterUUID *string `json:"metricExporterUuid"`
	GenericExporterID  *string `json:"genericExporterID"`
}

type VPCEndpoint struct {
	Host  string `json:"host"`
	Port  int64  `json:"port"`
	VPCId string `json:"vpcId"`
}

type ResourceSpec struct {
	ID   string `json:"id"`
	Spec struct {
		MilliCPU         int64 `json:"milliCPU"`
		MemoryGB         int64 `json:"memoryGB"`
		StorageGB        int64 `json:"storageGB"`
		ReplicaCount     int64 `json:"replicaCount"`
		SyncReplicaCount int64 `json:"syncReplicaCount"`
	} `json:"spec"`
}

type Metadata struct {
	Environment string `json:"environment"`
}

type CreateServiceRequest struct {
	Name     string
	MilliCPU string
	MemoryGB string
	// StorageGB is used for forks, since the CreateServiceRequest expects a storage to be requested
	// and the fork instance should match the storage size of the primary.
	StorageGB        string
	RegionCode       string
	ReplicaCount     string
	SyncReplicaCount string
	VpcID            int64
	ForkConfig       *ForkConfig

	EnableConnectionPooler bool
	EnvironmentTag         string
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

// ServiceEndpoints represents all service endpoints.
type ServiceEndpoints struct {
	Primary *EndpointAddress `json:"primary"`
	Replica *EndpointAddress `json:"replica"`
	Pooler  *EndpointAddress `json:"pooler"`
}

// EndpointAddress represents the endpoint address.
type EndpointAddress struct {
	// The hostname to use for connections.
	Host string `json:"host"`
	// The port to use for connections.
	Port int `json:"port"`
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
			"milliCPU":                request.MilliCPU,
			"storageGB":               request.StorageGB,
			"memoryGB":                request.MemoryGB,
			"replicaCount":            request.ReplicaCount,
			"synchronousReplicaCount": request.SyncReplicaCount,
		},
		"enableConnectionPooler": request.EnableConnectionPooler,
	}
	if request.VpcID > 0 {
		variables["vpcId"] = request.VpcID
	}
	if request.ForkConfig != nil {
		variables["forkConfig"] = request.ForkConfig
	}
	if request.EnvironmentTag != "" {
		variables["environmentTag"] = request.EnvironmentTag
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
	if len(resp.Errors) > 0 && !strings.Contains(resp.Errors[0].Message, "no Endpoint for that service id exists") {
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

func (c *Client) SetReplicaCount(ctx context.Context, serviceID string, replicaCount int, syncReplicaCount int) error {
	tflog.Trace(ctx, "Client.SetReplicaCount")

	req := map[string]interface{}{
		"operationName": "SetReplicaCount",
		"query":         SetReplicaCountMutation,
		"variables": map[string]any{
			"projectId":               c.projectID,
			"serviceId":               serviceID,
			"replicaCount":            replicaCount,
			"synchronousReplicaCount": syncReplicaCount,
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

func (c *Client) ResetServicePassword(ctx context.Context, serviceID string, password string) error {
	tflog.Trace(ctx, "Client.ResetServicePassword")

	req := map[string]interface{}{
		"operationName": "ResetServicePassword",
		"query":         ResetServicePassword,
		"variables": map[string]any{
			"projectId": c.projectID,
			"serviceId": serviceID,
			"password":  password,
			// we only support SCRAM password type, MD5 is deprecated in the backend
			"passwordType": "SCRAM",
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

func (c *Client) SetEnvironmentTag(ctx context.Context, serviceID, environment string) error {
	tflog.Trace(ctx, "Client.SetEnvironmentTag")
	req := map[string]interface{}{
		"operationName": "SetEnvironmentTag",
		"query":         SetEnvironmentTagMutation,
		"variables": map[string]any{
			"projectId":   c.projectID,
			"serviceId":   serviceID,
			"environment": environment,
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
