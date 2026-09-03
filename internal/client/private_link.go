package client

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type EndpointBindingType string

const (
	EndpointBindingTypeUnspecified EndpointBindingType = "unspecified"
	EndpointBindingTypePrimary     EndpointBindingType = "primary"
	EndpointBindingTypeReplica     EndpointBindingType = "replica"
	EndpointBindingTypePooler      EndpointBindingType = "pooler"
)

type PrivateLinkBinding struct {
	ProjectID    string              `json:"projectId"`
	ServiceID    string              `json:"serviceId"`
	ConnectionID string              `json:"connectionId"`
	BindingType  EndpointBindingType `json:"bindingType"`
	Port         int                 `json:"port"`
	Hostname     string              `json:"hostname"`
	CreatedAt    string              `json:"createdAt"`
}

type ListPrivateLinkBindingsResponse struct {
	Bindings []*PrivateLinkBinding `json:"listPrivateLinkBindings"`
}

type AttachServiceToPrivateLinkConnectionResponse struct {
	Result string `json:"attachServiceToPrivateLinkConnection"`
}

type DetachServiceFromPrivateLinkConnectionResponse struct {
	Result string `json:"detachServiceFromPrivateLinkConnection"`
}

type PrivateLinkConnection struct {
	ConnectionID         string                `json:"connectionId"`
	LinkIdentifier       string                `json:"linkIdentifier"`
	State                string                `json:"state"`
	Name                 string                `json:"name"`
	Region               string                `json:"region"`
	PrincipalID          string                `json:"principalId"`
	CloudProvider        string                `json:"cloudProvider"`
	ProviderConnectionID string                `json:"providerConnectionId"`
	CreatedAt            string                `json:"createdAt"`
	UpdatedAt            string                `json:"updatedAt"`
	Bindings             []*PrivateLinkBinding `json:"bindings"`
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

func (c *Client) AttachServiceToPrivateLinkConnection(ctx context.Context, serviceID string, connectionIDs []string) error {
	tflog.Trace(ctx, "Client.AttachServiceToPrivateLinkConnection")
	req := map[string]interface{}{
		"operationName": "AttachServiceToPrivateLinkConnection",
		"query":         AttachServiceToPrivateLinkConnectionMutation,
		"variables": map[string]interface{}{
			"projectId":     c.projectID,
			"serviceId":     serviceID,
			"connectionIds": connectionIDs,
		},
	}
	var resp Response[AttachServiceToPrivateLinkConnectionResponse]
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

func (c *Client) DetachServiceFromPrivateLinkConnection(ctx context.Context, serviceID string, connectionIDs []string) error {
	tflog.Trace(ctx, "Client.DetachServiceFromPrivateLinkConnection")
	req := map[string]interface{}{
		"operationName": "DetachServiceFromPrivateLinkConnection",
		"query":         DetachServiceFromPrivateLinkConnectionMutation,
		"variables": map[string]interface{}{
			"projectId":     c.projectID,
			"serviceId":     serviceID,
			"connectionIds": connectionIDs,
		},
	}
	var resp Response[DetachServiceFromPrivateLinkConnectionResponse]
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

type ClaimPrivateLinkConnectionResponse struct {
	Connection *PrivateLinkConnection `json:"claimPrivateLinkConnection"`
}

// ClaimPrivateLinkConnection assigns an unowned connection to the client's
// project. The claim identifier is read by the customer from their own cloud
// resource: the VPC endpoint ID on AWS, the private endpoint's resourceGuid on
// Azure. Re-claiming a connection the project already owns is idempotent.
func (c *Client) ClaimPrivateLinkConnection(ctx context.Context, claimIdentifier string) (*PrivateLinkConnection, error) {
	tflog.Trace(ctx, "Client.ClaimPrivateLinkConnection")
	req := map[string]interface{}{
		"operationName": "ClaimPrivateLinkConnection",
		"query":         ClaimPrivateLinkConnectionMutation,
		"variables": map[string]interface{}{
			"projectId":       c.projectID,
			"claimIdentifier": claimIdentifier,
		},
	}
	var resp Response[ClaimPrivateLinkConnectionResponse]
	if err := c.do(ctx, req, &resp); err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, resp.Errors[0]
	}
	if resp.Data == nil || resp.Data.Connection == nil {
		return nil, errors.New("no response found")
	}
	return resp.Data.Connection, nil
}

type UpdatePrivateLinkConnectionResponse struct {
	Connection *PrivateLinkConnection `json:"updatePrivateLinkConnection"`
}

func (c *Client) UpdatePrivateLinkConnection(ctx context.Context, connectionID string, name *string) (*PrivateLinkConnection, error) {
	tflog.Trace(ctx, "Client.UpdatePrivateLinkConnection")
	variables := map[string]interface{}{
		"projectId":    c.projectID,
		"connectionId": connectionID,
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
	Region        string `json:"region"`
	CloudProvider string `json:"cloudProvider"`
	ServiceName   string `json:"serviceName"`
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

// DeletePrivateLinkConnection rejects the connection provider-side. This is
// terminal: sync skips rejected connections, so the customer must recreate the
// private endpoint to reconnect.
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
