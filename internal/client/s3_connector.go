package client

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// S3Connector represents an S3 Live Sync connector.
type S3Connector struct {
	ID                 string                  `json:"id"`
	ProjectID          string                  `json:"project_id"`
	ServiceID          string                  `json:"service_id"`
	CreatedAt          string                  `json:"created_at"`
	UpdatedAt          string                  `json:"updated_at"`
	Bucket             string                  `json:"bucket"`
	Pattern            string                  `json:"pattern"`
	Credentials        *S3ConnectorCredentials `json:"credentials"`
	Definition         *S3ConnectorDefinition  `json:"definition"`
	TableIdentifier    *S3ConnectorTableID     `json:"table_identifier"`
	Frequency          string                  `json:"frequency"`
	NextTick           string                  `json:"next_tick"`
	LastImportedObject string                  `json:"last_imported_object"`
	Enabled            bool                    `json:"enabled"`
	Name               string                  `json:"name"`
	LastError          string                  `json:"last_error"`
	Settings           *ImportSettings         `json:"settings,omitempty"`
}

// ImportSettings holds import behavior settings.
type ImportSettings struct {
	OnConflictDoNothing bool `json:"on_conflict_do_nothing"`
}

// S3ConnectorCredentials holds authentication configuration.
type S3ConnectorCredentials struct {
	Type string                      `json:"type"`
	Role *S3ConnectorCredentialsRole `json:"role,omitempty"`
}

// S3ConnectorCredentialsRole holds the role ARN.
type S3ConnectorCredentialsRole struct {
	ARN string `json:"arn"`
}

// S3ConnectorDefinition holds file format configuration.
type S3ConnectorDefinition struct {
	Type    string                        `json:"type"`
	CSV     *S3ConnectorDefinitionCSV     `json:"csv,omitempty"`
	Parquet *S3ConnectorDefinitionParquet `json:"parquet,omitempty"`
}

// S3ConnectorDefinitionCSV holds CSV-specific configuration.
type S3ConnectorDefinitionCSV struct {
	Delimiter         string          `json:"delimiter,omitempty"`
	SkipHeader        bool            `json:"skip_header"`
	ColumnNames       []string        `json:"column_names,omitempty"`
	ColumnMappings    []ColumnMapping `json:"column_mappings,omitempty"`
	AutoColumnMapping bool            `json:"auto_column_mapping"`
}

// S3ConnectorDefinitionParquet holds Parquet-specific configuration.
type S3ConnectorDefinitionParquet struct {
	ColumnMappings    []ColumnMapping `json:"column_mappings,omitempty"`
	AutoColumnMapping bool            `json:"auto_column_mapping"`
}

// ColumnMapping maps source to destination columns.
type ColumnMapping struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
}

// S3ConnectorTableID identifies the target table.
type S3ConnectorTableID struct {
	SchemaName string `json:"schema_name,omitempty"`
	TableName  string `json:"table_name"`
}

// CreateS3ConnectorRequest holds parameters for creating an S3 connector.
type CreateS3ConnectorRequest struct {
	ID              string
	ProjectID       string
	ServiceID       string
	Bucket          string
	Pattern         string
	Credentials     *S3ConnectorCredentials
	Definition      *S3ConnectorDefinition
	TableIdentifier *S3ConnectorTableID
	Name            string
	Settings        *ImportSettings
}

// S3ConnectorUpdateRequest represents a single update operation.
type S3ConnectorUpdateRequest struct {
	Type               string                 `json:"type"`
	Bucket             map[string]string      `json:"bucket,omitempty"`
	Pattern            map[string]string      `json:"pattern,omitempty"`
	Credentials        map[string]interface{} `json:"credentials,omitempty"`
	Definition         map[string]interface{} `json:"definition,omitempty"`
	Frequency          map[string]string      `json:"frequency,omitempty"`
	NextTick           map[string]string      `json:"next_tick,omitempty"`
	LastImportedObject map[string]string      `json:"last_imported_object,omitempty"`
	TableIdentifier    map[string]interface{} `json:"table_identifier,omitempty"`
	Enabled            map[string]bool        `json:"enabled,omitempty"`
	Name               map[string]string      `json:"name,omitempty"`
	Settings           map[string]interface{} `json:"settings,omitempty"`
}

// Response types.
type CreateS3ConnectorResponse struct {
	CreateS3LiveSync string `json:"createS3LiveSync"`
}

type UpdateS3ConnectorResponse struct {
	Connector *S3Connector `json:"updateS3LiveSync"`
}

type GetS3ConnectorResponse struct {
	Connector *S3Connector `json:"getS3LiveSync"`
}

type DeleteS3ConnectorResponse struct {
	DeleteS3LiveSync string `json:"deleteS3LiveSync"`
}

// CreateS3Connector creates a new S3 connector (returns success only).
func (c *Client) CreateS3Connector(ctx context.Context, req CreateS3ConnectorRequest) error {
	tflog.Trace(ctx, "Client.CreateS3Connector")

	variables := map[string]interface{}{
		"id":        req.ID,
		"projectId": req.ProjectID,
		"serviceId": req.ServiceID,
	}

	// Add optional fields if provided
	if req.Bucket != "" {
		variables["bucket"] = req.Bucket
	}
	if req.Pattern != "" {
		variables["pattern"] = req.Pattern
	}
	if req.Name != "" {
		variables["name"] = req.Name
	}
	if req.Credentials != nil {
		variables["credentials"] = buildCredentialsInput(req.Credentials)
	}
	if req.Definition != nil {
		variables["definition"] = buildDefinitionInput(req.Definition)
	}
	if req.TableIdentifier != nil {
		variables["tableIdentifier"] = map[string]interface{}{
			"table_name":  req.TableIdentifier.TableName,
			"schema_name": req.TableIdentifier.SchemaName,
		}
	}
	// Note: settings are not supported during creation, only during updates

	request := map[string]interface{}{
		"operationName": "CreateS3Connector",
		"query":         CreateS3ConnectorMutation,
		"variables":     variables,
	}

	var resp Response[CreateS3ConnectorResponse]
	if err := c.do(ctx, request, &resp); err != nil {
		return fmt.Errorf("error executing API request: %w", err)
	}
	if len(resp.Errors) > 0 {
		return fmt.Errorf("API returned an error: %w", resp.Errors[0])
	}

	return nil
}

// UpdateS3Connector updates an existing S3 connector.
func (c *Client) UpdateS3Connector(ctx context.Context, id, projectID, serviceID string, requests []S3ConnectorUpdateRequest) (*S3Connector, error) {
	tflog.Trace(ctx, "Client.UpdateS3Connector")

	req := map[string]interface{}{
		"operationName": "UpdateS3Connector",
		"query":         UpdateS3ConnectorMutation,
		"variables": map[string]interface{}{
			"id":        id,
			"projectId": projectID,
			"serviceId": serviceID,
			"requests":  requests,
		},
	}

	var resp Response[UpdateS3ConnectorResponse]
	if err := c.do(ctx, req, &resp); err != nil {
		return nil, fmt.Errorf("error executing API request: %w", err)
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("API returned an error: %w", resp.Errors[0])
	}
	if resp.Data == nil || resp.Data.Connector == nil {
		return nil, errors.New("API response did not contain connector data")
	}

	return resp.Data.Connector, nil
}

// GetS3Connector retrieves an S3 connector by ID.
func (c *Client) GetS3Connector(ctx context.Context, id, projectID, serviceID string) (*S3Connector, error) {
	tflog.Trace(ctx, "Client.GetS3Connector")

	req := map[string]interface{}{
		"operationName": "GetS3Connector",
		"query":         GetS3ConnectorQuery,
		"variables": map[string]interface{}{
			"id":        id,
			"projectId": projectID,
			"serviceId": serviceID,
		},
	}

	var resp Response[GetS3ConnectorResponse]
	if err := c.do(ctx, req, &resp); err != nil {
		return nil, fmt.Errorf("error executing API request: %w", err)
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("API returned an error: %w", resp.Errors[0])
	}
	if resp.Data == nil || resp.Data.Connector == nil {
		return nil, errors.New("connector not found")
	}

	return resp.Data.Connector, nil
}

// DeleteS3Connector deletes an S3 connector.
func (c *Client) DeleteS3Connector(ctx context.Context, id, projectID, serviceID string) error {
	tflog.Trace(ctx, "Client.DeleteS3Connector")

	req := map[string]interface{}{
		"operationName": "DeleteS3Connector",
		"query":         DeleteS3ConnectorMutation,
		"variables": map[string]interface{}{
			"id":        id,
			"projectId": projectID,
			"serviceId": serviceID,
		},
	}

	var resp Response[DeleteS3ConnectorResponse]
	if err := c.do(ctx, req, &resp); err != nil {
		return fmt.Errorf("error executing API request: %w", err)
	}
	if len(resp.Errors) > 0 {
		return fmt.Errorf("API returned an error: %w", resp.Errors[0])
	}

	return nil
}

// Helper functions to build GraphQL input structures.
func buildCredentialsInput(creds *S3ConnectorCredentials) map[string]interface{} {
	input := map[string]interface{}{
		"type": creds.Type,
	}
	if creds.Role != nil {
		input["role"] = map[string]interface{}{
			"arn": creds.Role.ARN,
		}
	}
	return input
}

func buildDefinitionInput(def *S3ConnectorDefinition) map[string]interface{} {
	input := map[string]interface{}{
		"type": def.Type,
	}

	if def.CSV != nil {
		csvInput := map[string]interface{}{
			"skip_header":         def.CSV.SkipHeader,
			"auto_column_mapping": def.CSV.AutoColumnMapping,
		}
		if def.CSV.Delimiter != "" {
			csvInput["delimiter"] = def.CSV.Delimiter
		}
		if len(def.CSV.ColumnNames) > 0 {
			csvInput["column_names"] = def.CSV.ColumnNames
		}
		if len(def.CSV.ColumnMappings) > 0 {
			csvInput["column_mappings"] = buildColumnMappingsInput(def.CSV.ColumnMappings)
		}
		input["csv"] = csvInput
	}

	if def.Parquet != nil {
		parquetInput := map[string]interface{}{
			"auto_column_mapping": def.Parquet.AutoColumnMapping,
		}
		if len(def.Parquet.ColumnMappings) > 0 {
			parquetInput["column_mappings"] = buildColumnMappingsInput(def.Parquet.ColumnMappings)
		}
		input["parquet"] = parquetInput
	}

	return input
}

func buildColumnMappingsInput(mappings []ColumnMapping) []map[string]interface{} {
	result := make([]map[string]interface{}, len(mappings))
	for i, m := range mappings {
		result[i] = map[string]interface{}{
			"source":      m.Source,
			"destination": m.Destination,
		}
	}
	return result
}
