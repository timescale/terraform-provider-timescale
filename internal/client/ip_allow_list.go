package client

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type IPAllowList struct {
	ListID             int64    `json:"listId"`
	Description        string   `json:"description"`
	CIDRBlocks         []string `json:"cidrBlocks"`
	AssociatedServices []string `json:"associatedServices"`
}

type ipAllowListsResponse struct {
	Lists []*IPAllowList `json:"getIPAllowLists"`
}

type configureIPAllowListResponse struct {
	Result struct {
		ListID int64 `json:"listId"`
	} `json:"configureIPAllowList"`
}

// GetIPAllowLists returns all IP allow lists of the project.
func (c *Client) GetIPAllowLists(ctx context.Context) ([]*IPAllowList, error) {
	tflog.Trace(ctx, "Client.GetIPAllowLists")
	req := map[string]interface{}{
		"operationName": "GetIPAllowLists",
		"query":         GetIPAllowListsQuery,
		"variables":     map[string]any{"projectId": c.projectID},
	}
	var resp Response[ipAllowListsResponse]
	if err := c.do(ctx, req, &resp); err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, resp.Errors[0]
	}
	if resp.Data == nil {
		return nil, errors.New("no response found")
	}
	return resp.Data.Lists, nil
}

// GetIPAllowList returns the list with the given ID, or nil if it doesn't exist.
func (c *Client) GetIPAllowList(ctx context.Context, listID int64) (*IPAllowList, error) {
	lists, err := c.GetIPAllowLists(ctx)
	if err != nil {
		return nil, err
	}
	for _, l := range lists {
		if l.ListID == listID {
			return l, nil
		}
	}
	return nil, nil
}

// ConfigureIPAllowList creates a list (listID nil) or replaces an existing one.
// The API deletes the list when cidrBlocks is empty, so use DeleteIPAllowList for that.
func (c *Client) ConfigureIPAllowList(ctx context.Context, listID *int64, description string, cidrBlocks []string) (int64, error) {
	tflog.Trace(ctx, "Client.ConfigureIPAllowList")
	vars := map[string]any{
		"projectId":   c.projectID,
		"description": description,
		"cidrBlocks":  cidrBlocks,
	}
	if listID != nil {
		vars["listId"] = *listID
	}
	req := map[string]interface{}{
		"operationName": "ConfigureIPAllowList",
		"query":         ConfigureIPAllowListMutation,
		"variables":     vars,
	}
	var resp Response[configureIPAllowListResponse]
	if err := c.do(ctx, req, &resp); err != nil {
		return 0, err
	}
	if len(resp.Errors) > 0 {
		return 0, resp.Errors[0]
	}
	if resp.Data == nil {
		return 0, errors.New("no response found")
	}
	return resp.Data.Result.ListID, nil
}

// DeleteIPAllowList deletes a list by configuring it with no CIDR blocks.
func (c *Client) DeleteIPAllowList(ctx context.Context, listID int64) error {
	tflog.Trace(ctx, "Client.DeleteIPAllowList")
	req := map[string]interface{}{
		"operationName": "ConfigureIPAllowList",
		"query":         ConfigureIPAllowListMutation,
		"variables": map[string]any{
			"projectId":   c.projectID,
			"description": "",
			"cidrBlocks":  nil,
			"listId":      listID,
		},
	}
	var resp Response[configureIPAllowListResponse]
	if err := c.do(ctx, req, &resp); err != nil {
		return err
	}
	if len(resp.Errors) > 0 {
		return resp.Errors[0]
	}
	return nil
}

func (c *Client) bindOp(ctx context.Context, op, query, serviceID string, listID int64) error {
	req := map[string]interface{}{
		"operationName": op,
		"query":         query,
		"variables": map[string]any{
			"projectId": c.projectID,
			"serviceId": serviceID,
			"listId":    listID,
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

// BindIPAllowList associates a list with a service.
func (c *Client) BindIPAllowList(ctx context.Context, serviceID string, listID int64) error {
	tflog.Trace(ctx, "Client.BindIPAllowList")
	return c.bindOp(ctx, "BindIPAllowList", BindIPAllowListMutation, serviceID, listID)
}

// UnbindIPAllowList removes the association between a list and a service.
func (c *Client) UnbindIPAllowList(ctx context.Context, serviceID string, listID int64) error {
	tflog.Trace(ctx, "Client.UnbindIPAllowList")
	return c.bindOp(ctx, "UnbindIPAllowList", UnbindIPAllowListMutation, serviceID, listID)
}
