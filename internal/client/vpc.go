package client

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type VPC struct {
	ID                 string               `json:"id"`
	ProvisionedID      string               `json:"provisionedId"`
	ProjectID          string               `json:"projectId"`
	CIDR               string               `json:"cidr"`
	Name               string               `json:"name"`
	RegionCode         string               `json:"regionCode"`
	Status             string               `json:"status"`
	ErrorMessage       string               `json:"errorMessage"`
	Created            string               `json:"created"`
	Updated            string               `json:"updated"`
	PeeringConnections []*PeeringConnection `json:"peeringConnections"`
}

type PeeringConnection struct {
	ID           string     `json:"id"`
	VpcID        string     `json:"vpcId"`
	Status       string     `json:"status"`
	ErrorMessage string     `json:"errorMessage"`
	PeerVpcs     []*PeerVpc `json:"peerVpc"`
}

type PeerVpc struct {
	ID         string `json:"id"`
	CIDR       string `json:"cidr"`
	AccountID  string `json:"accountId"`
	RegionCode string `json:"regionCode"`
}

type VpcsResponse struct {
	Vpcs []*VPC `json:"getAllVpcs"`
}

func (c *Client) GetVPCs(ctx context.Context) ([]*VPC, error) {
	tflog.Trace(ctx, "Client.GetVPCs")
	req := map[string]interface{}{
		"operationName": "GetAllVpcs",
		"query":         GetVPCsQuery,
		"variables": map[string]string{
			"projectId": c.projectID,
		},
	}
	var resp Response[VpcsResponse]
	if err := c.do(ctx, req, &resp); err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, resp.Errors[0]
	}
	if resp.Data == nil {
		return nil, errors.New("no response found")
	}
	return resp.Data.Vpcs, nil
}

func (c *Client) AttachServiceToVpc(ctx context.Context, serviceID string, vpcID int64) error {
	tflog.Trace(ctx, "Client.AttachServiceToVpc")

	req := map[string]interface{}{
		"operationName": "AttachServiceToVpc",
		"query":         AttachServiceToVPCMutation,
		"variables": map[string]any{
			"projectId": c.projectID,
			"serviceId": serviceID,
			"vpcId":     vpcID,
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

func (c *Client) DetachServiceFromVpc(ctx context.Context, serviceID string, vpcID int64) error {
	tflog.Trace(ctx, "Client.DetachServiceFromVpc")

	req := map[string]interface{}{
		"operationName": "DetachServiceFromVpc",
		"query":         DetachServiceFromVPCMutation,
		"variables": map[string]any{
			"projectId": c.projectID,
			"serviceId": serviceID,
			"vpcId":     vpcID,
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
