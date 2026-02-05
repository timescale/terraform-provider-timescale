package client

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type PrivateLinkBindingType string

const (
	PrivateLinkBindingTypeUnspecified PrivateLinkBindingType = "UNSPECIFIED"
	PrivateLinkBindingTypePrimary     PrivateLinkBindingType = "PRIMARY"
	PrivateLinkBindingTypeReplica     PrivateLinkBindingType = "REPLICA"
	PrivateLinkBindingTypePooler      PrivateLinkBindingType = "POOLER"
)

type PrivateLinkBinding struct {
	ProjectID                   string                 `json:"projectId"`
	ServiceID                   string                 `json:"serviceId"`
	PrivateEndpointConnectionID string                 `json:"privateEndpointConnectionId"`
	BindingType                 PrivateLinkBindingType `json:"bindingType"`
	Port                        int                    `json:"port"`
	Hostname                    string                 `json:"hostname"`
	CreatedAt                   string                 `json:"createdAt"`
}

type ListPrivateLinkBindingsResponse struct {
	Bindings []*PrivateLinkBinding `json:"listPrivateLinkBindings"`
}

type AttachServiceToPrivateLinkResponse struct {
	Result string `json:"attachServiceToPrivateLink"`
}

type DetachServiceFromPrivateLinkResponse struct {
	Result string `json:"detachServiceFromPrivateLink"`
}

type PrivateLinkConnection struct {
	ConnectionID        string                `json:"connectionId"`
	SubscriptionID      string                `json:"subscriptionId"`
	LinkIdentifier      string                `json:"linkIdentifier"`
	State               string                `json:"state"`
	IPAddress           string                `json:"ipAddress"`
	Name                string                `json:"name"`
	AzureConnectionName string                `json:"azureConnectionName"`
	Region              string                `json:"region"`
	CreatedAt           string                `json:"createdAt"`
	UpdatedAt           string                `json:"updatedAt"`
	Bindings            []*PrivateLinkBinding `json:"bindings"`
}

type ListPrivateLinkConnectionsResponse struct {
	Connections []*PrivateLinkConnection `json:"listPrivateLinkConnections"`
}

func (c *Client) ListPrivateLinkBindings(ctx context.Context, serviceID string) ([]*PrivateLinkBinding, error) {
	tflog.Trace(ctx, "Client.ListPrivateLinkBindings")
	req := map[string]interface{}{
		"operationName": "ListPrivateLinkBindings",
		"query":         ListPrivateLinkBindingsQuery,
		"variables": map[string]string{
			"projectId": c.projectID,
			"serviceId": serviceID,
		},
	}
	var resp Response[ListPrivateLinkBindingsResponse]
	if err := c.do(ctx, req, &resp); err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, resp.Errors[0]
	}
	if resp.Data == nil {
		return nil, errors.New("no response found")
	}
	return resp.Data.Bindings, nil
}

func (c *Client) AttachServiceToPrivateLink(ctx context.Context, serviceID, privateEndpointConnectionID string) error {
	tflog.Trace(ctx, "Client.AttachServiceToPrivateLink")
	req := map[string]interface{}{
		"operationName": "AttachServiceToPrivateLink",
		"query":         AttachServiceToPrivateLinkMutation,
		"variables": map[string]string{
			"projectId":                   c.projectID,
			"serviceId":                   serviceID,
			"privateEndpointConnectionId": privateEndpointConnectionID,
		},
	}
	var resp Response[AttachServiceToPrivateLinkResponse]
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

func (c *Client) DetachServiceFromPrivateLink(ctx context.Context, serviceID, privateEndpointConnectionID string) error {
	tflog.Trace(ctx, "Client.DetachServiceFromPrivateLink")
	req := map[string]interface{}{
		"operationName": "DetachServiceFromPrivateLink",
		"query":         DetachServiceFromPrivateLinkMutation,
		"variables": map[string]string{
			"projectId":                   c.projectID,
			"serviceId":                   serviceID,
			"privateEndpointConnectionId": privateEndpointConnectionID,
		},
	}
	var resp Response[DetachServiceFromPrivateLinkResponse]
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

func (c *Client) ListPrivateLinkConnections(ctx context.Context, region string) ([]*PrivateLinkConnection, error) {
	tflog.Trace(ctx, "Client.ListPrivateLinkConnections")
	variables := map[string]interface{}{
		"projectId": c.projectID,
	}
	if region != "" {
		variables["region"] = region
	}
	req := map[string]interface{}{
		"operationName": "ListPrivateLinkConnections",
		"query":         ListPrivateLinkConnectionsQuery,
		"variables":     variables,
	}
	var resp Response[ListPrivateLinkConnectionsResponse]
	if err := c.do(ctx, req, &resp); err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, resp.Errors[0]
	}
	if resp.Data == nil {
		return nil, errors.New("no response found")
	}
	return resp.Data.Connections, nil
}

func (c *Client) SyncPrivateLinkConnections(ctx context.Context) error {
	tflog.Trace(ctx, "Client.SyncPrivateLinkConnections")
	req := map[string]interface{}{
		"operationName": "SyncPrivateLinkConnections",
		"query":         SyncPrivateLinkConnectionsMutation,
		"variables": map[string]string{
			"projectId": c.projectID,
		},
	}
	var resp Response[any]
	if err := c.do(ctx, req, &resp); err != nil {
		return err
	}
	if len(resp.Errors) > 0 {
		return resp.Errors[0]
	}
	return nil
}

type UpdatePrivateLinkConnectionResponse struct {
	Connection *PrivateLinkConnection `json:"updatePrivateLinkConnection"`
}

func (c *Client) UpdatePrivateLinkConnection(ctx context.Context, connectionID string, ipAddress *string, name *string) (*PrivateLinkConnection, error) {
	tflog.Trace(ctx, "Client.UpdatePrivateLinkConnection")
	variables := map[string]interface{}{
		"projectId":    c.projectID,
		"connectionId": connectionID,
	}
	if ipAddress != nil {
		variables["ipAddress"] = *ipAddress
	}
	if name != nil {
		variables["name"] = *name
	}
	req := map[string]interface{}{
		"operationName": "UpdatePrivateLinkConnection",
		"query":         UpdatePrivateLinkConnectionMutation,
		"variables":     variables,
	}
	var resp Response[UpdatePrivateLinkConnectionResponse]
	if err := c.do(ctx, req, &resp); err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, resp.Errors[0]
	}
	if resp.Data == nil {
		return nil, errors.New("no response found")
	}
	return resp.Data.Connection, nil
}

type PrivateLinkAvailableRegion struct {
	Region                  string `json:"region"`
	PrivateLinkServiceAlias string `json:"privateLinkServiceAlias"`
}

type ListPrivateLinkAvailableRegionsResponse struct {
	Regions []*PrivateLinkAvailableRegion `json:"listPrivateLinkAvailableRegions"`
}

func (c *Client) ListPrivateLinkAvailableRegions(ctx context.Context) ([]*PrivateLinkAvailableRegion, error) {
	tflog.Trace(ctx, "Client.ListPrivateLinkAvailableRegions")
	req := map[string]interface{}{
		"operationName": "ListPrivateLinkAvailableRegions",
		"query":         ListPrivateLinkAvailableRegionsQuery,
	}
	var resp Response[ListPrivateLinkAvailableRegionsResponse]
	if err := c.do(ctx, req, &resp); err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, resp.Errors[0]
	}
	if resp.Data == nil {
		return nil, errors.New("no response found")
	}
	return resp.Data.Regions, nil
}

type DeletePrivateLinkConnectionResponse struct {
	Result string `json:"deletePrivateLinkConnection"`
}

func (c *Client) DeletePrivateLinkConnection(ctx context.Context, connectionID string) error {
	tflog.Trace(ctx, "Client.DeletePrivateLinkConnection")
	req := map[string]interface{}{
		"operationName": "DeletePrivateLinkConnection",
		"query":         DeletePrivateLinkConnectionMutation,
		"variables": map[string]string{
			"projectId":    c.projectID,
			"connectionId": connectionID,
		},
	}
	var resp Response[DeletePrivateLinkConnectionResponse]
	if err := c.do(ctx, req, &resp); err != nil {
		return err
	}
	if len(resp.Errors) > 0 {
		return resp.Errors[0]
	}
	return nil
}
