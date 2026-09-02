package client

import (
	"context"
	"errors"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// ParameterInfo is the subset of the GraphQL ParameterInfo type the provider uses.
// Field names are snake_case on the wire.
type ParameterInfo struct {
	Name             string `json:"name"`
	IsUserEditable   bool   `json:"is_user_editable"`
	RequiresRestart  bool   `json:"requires_restart"`
	IsPendingRestart bool   `json:"is_pending_restart"`
	IsUserModified   bool   `json:"is_user_modified"`
}

// StringParameter is a string or boolean parameter. Booleans use "on" and "off"
// and report AllowedValues as exactly that pair.
type StringParameter struct {
	Info          ParameterInfo `json:"info"`
	CurrentValue  string        `json:"current_value"`
	AllowedValues []string      `json:"allowed_values"`
}

// NumericParameter is a numeric parameter. Unit is UNDEFINED for plain numbers.
type NumericParameter struct {
	Info         ParameterInfo `json:"info"`
	Unit         string        `json:"unit"`
	CurrentValue float64       `json:"current_value"`
}

type PostgresParameters struct {
	StringParameters  []StringParameter  `json:"string_parameters"`
	NumericParameters []NumericParameter `json:"numeric_parameters"`
}

type GetPostgresParametersResponse struct {
	PostgresParameters PostgresParameters `json:"getPostgresParameters"`
}

type SetNumericParameter struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

type SetStringParameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type ParameterError struct {
	Name         string `json:"name"`
	ErrorMessage string `json:"errorMessage"`
}

type SetPostgresParametersResponse struct {
	ParameterErrors []ParameterError `json:"setPostgresParameters"`
}

// ParameterErrors is returned when the API accepted the call but rejected individual parameters.
type ParameterErrors []ParameterError

func (e ParameterErrors) Error() string {
	parts := make([]string, 0, len(e))
	for _, pe := range e {
		parts = append(parts, pe.Name+": "+pe.ErrorMessage)
	}
	return strings.Join(parts, "; ")
}

func (c *Client) GetPostgresParameters(ctx context.Context, serviceID string) (*PostgresParameters, error) {
	tflog.Trace(ctx, "Client.GetPostgresParameters")
	req := map[string]interface{}{
		"operationName": "GetPostgresParameters",
		"query":         GetPostgresParametersQuery,
		"variables": map[string]string{
			"projectId": c.projectID,
			"serviceId": serviceID,
		},
	}
	var resp Response[GetPostgresParametersResponse]
	if err := c.do(ctx, req, &resp); err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, resp.Errors[0]
	}
	if resp.Data == nil {
		return nil, errors.New("no response found")
	}
	return &resp.Data.PostgresParameters, nil
}

// SetPostgresParameters sets parameters in one batch. Per-parameter rejections are
// returned as ParameterErrors.
func (c *Client) SetPostgresParameters(ctx context.Context, serviceID string, numeric []SetNumericParameter, str []SetStringParameter) error {
	tflog.Trace(ctx, "Client.SetPostgresParameters")
	if numeric == nil {
		numeric = []SetNumericParameter{}
	}
	if str == nil {
		str = []SetStringParameter{}
	}
	req := map[string]interface{}{
		"operationName": "SetPostgresParameters",
		"query":         SetPostgresParametersMutation,
		"variables": map[string]any{
			"projectId":         c.projectID,
			"serviceId":         serviceID,
			"numericParameters": numeric,
			"stringParameters":  str,
		},
	}
	var resp Response[SetPostgresParametersResponse]
	if err := c.do(ctx, req, &resp); err != nil {
		return err
	}
	if len(resp.Errors) > 0 {
		return resp.Errors[0]
	}
	if resp.Data == nil {
		return errors.New("no response found")
	}
	if len(resp.Data.ParameterErrors) > 0 {
		return ParameterErrors(resp.Data.ParameterErrors)
	}
	return nil
}
