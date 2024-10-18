package client

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"math/big"
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
	ID            string   `json:"id"`
	VPCID         string   `json:"vpcId"`
	ProvisionedID string   `json:"provisionedId"`
	Status        string   `json:"status"`
	ErrorMessage  string   `json:"errorMessage"`
	PeerVPC       *PeerVPC `json:"peerVPC"`
}

type PeerVPC struct {
	ID         string `json:"id"`
	CIDR       string `json:"cidr"`
	AccountID  string `json:"accountId"`
	RegionCode string `json:"regionCode"`
}

type VPCsResponse struct {
	VPCs []*VPC `json:"getAllVPCs"`
}

type CreateVPCResponse struct {
	VPC *VPC `json:"createVPC"`
}

type VPCResponse struct {
	VPC *VPC `json:"getVPC"`
}
type VPCNameResponse struct {
	VPC *VPC `json:"getVPCByName"`
}

func (c *Client) GetVPCs(ctx context.Context) ([]*VPC, error) {
	tflog.Trace(ctx, "Client.GetVPCs")
	req := map[string]interface{}{
		"operationName": "GetAllVPCs",
		"query":         GetVPCsQuery,
		"variables": map[string]string{
			"projectId": c.projectID,
		},
	}
	var resp Response[VPCsResponse]
	if err := c.do(ctx, req, &resp); err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, resp.Errors[0]
	}
	if resp.Data == nil {
		return nil, errors.New("no response found")
	}
	return resp.Data.VPCs, nil
}

func (c *Client) GetVPCByName(ctx context.Context, name string) (*VPC, error) {
	tflog.Trace(ctx, "Client.GetVPCByName")
	req := map[string]interface{}{
		"operationName": "GetVPCByName",
		"query":         GetVPCByNameQuery,
		"variables": map[string]string{
			"projectId": c.projectID,
			"name":      name,
		},
	}
	var resp Response[VPCNameResponse]
	if err := c.do(ctx, req, &resp); err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, resp.Errors[0]
	}
	if resp.Data == nil {
		return nil, errors.New("no vpc found")
	}
	return resp.Data.VPC, nil
}

func (c *Client) GetVPCByID(ctx context.Context, vpcID int64) (*VPC, error) {
	tflog.Trace(ctx, "Client.GetVPCByID")
	req := map[string]interface{}{
		"operationName": "GetVPCByID",
		"query":         GetVPCByIDQuery,
		"variables": map[string]any{
			"vpcId": vpcID,
		},
	}
	var resp Response[VPCResponse]
	if err := c.do(ctx, req, &resp); err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, resp.Errors[0]
	}
	if resp.Data == nil {
		return nil, errors.New("no vpc found")
	}
	return resp.Data.VPC, nil
}

func (c *Client) AttachServiceToVPC(ctx context.Context, serviceID string, vpcID int64) error {
	tflog.Trace(ctx, "Client.AttachServiceToVPC")

	req := map[string]interface{}{
		"operationName": "AttachServiceToVPC",
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

func (c *Client) DetachServiceFromVPC(ctx context.Context, serviceID string, vpcID int64) error {
	tflog.Trace(ctx, "Client.DetachServiceFromVPC")

	req := map[string]interface{}{
		"operationName": "DetachServiceFromVPC",
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

func (c *Client) CreateVPC(ctx context.Context, name, cidr, regionCode string) (*VPC, error) {
	tflog.Trace(ctx, "Client.CreateVPC")

	if name == "" {
		r, err := rand.Int(rand.Reader, big.NewInt(90000))
		if err != nil {
			return nil, err
		}
		name = fmt.Sprintf("vpc-%d", 10000+r.Int64())

	}

	req := map[string]interface{}{
		"operationName": "CreateVPC",
		"query":         CreateVPCMutation,
		"variables": map[string]string{
			"projectId":  c.projectID,
			"name":       name,
			"cidr":       cidr,
			"regionCode": regionCode,
		},
	}
	var resp Response[CreateVPCResponse]
	if err := c.do(ctx, req, &resp); err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, resp.Errors[0]
	}
	if resp.Data == nil {
		return nil, errors.New("no response found")
	}
	return resp.Data.VPC, nil
}

func (c *Client) RenameVPC(ctx context.Context, vpcID int64, newName string) error {
	tflog.Trace(ctx, "Client.GetVPCs")
	req := map[string]interface{}{
		"operationName": "RenameVPC",
		"query":         RenameVPCMutation,
		"variables": map[string]any{
			"projectId":  c.projectID,
			"forgeVpcId": vpcID,
			"newName":    newName,
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

func (c *Client) DeleteVPC(ctx context.Context, vpcID int64) error {
	tflog.Trace(ctx, "Client.DeleteVPC")

	req := map[string]interface{}{
		"operationName": "DeleteVPC",
		"query":         DeleteVPCMutation,
		"variables": map[string]any{
			"projectId": c.projectID,
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
	return nil
}

func (c *Client) OpenPeerRequest(ctx context.Context, vpcID int64, externalVpcID, accountID, regionCode string) error {
	tflog.Trace(ctx, "Client.OpenPeerRequest")

	req := map[string]interface{}{
		"operationName": "OpenPeerRequest",
		"query":         OpenPeerRequestMutation,
		"variables": map[string]any{
			"projectId":     c.projectID,
			"vpcId":         vpcID,
			"externalVpcId": externalVpcID,
			"accountId":     accountID,
			"regionCode":    regionCode,
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

func (c *Client) DeletePeeringConnection(ctx context.Context, vpcID, id int64) error {
	tflog.Trace(ctx, "Client.DeletePeeringConnection")

	req := map[string]interface{}{
		"operationName": "DeletePeeringConnection",
		"query":         DeletePeeringConnectionMutation,
		"variables": map[string]any{
			"projectId": c.projectID,
			"vpcId":     vpcID,
			"id":        id,
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
