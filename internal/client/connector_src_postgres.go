package client

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// --- SSH Tunnel Config ---

type SSHTunnelConfig struct {
	SSHTunnelID string `json:"sshTunnelId"`
	ProjectID   string `json:"projectId"`
	Name        string `json:"name"`
	PublicKey   string `json:"publicKey"`
	Username    string `json:"username"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

type createSSHTunnelConfigResponse struct {
	CreateSSHTunnelConfig struct {
		SSHTunnelConfig *SSHTunnelConfig `json:"sshTunnelConfig"`
	} `json:"createSSHTunnelConfig"`
}

type updateSSHTunnelConfigResponse struct {
	UpdateSSHTunnelConfig struct {
		SSHTunnelConfig *SSHTunnelConfig `json:"sshTunnelConfig"`
	} `json:"updateSSHTunnelConfig"`
}

type getSSHTunnelConfigResponse struct {
	GetSSHTunnelConfig struct {
		SSHTunnelConfig *SSHTunnelConfig `json:"sshTunnelConfig"`
	} `json:"getSSHTunnelConfig"`
}

// --- PgSrc Source Config ---

type PgSrcConfig struct {
	SourceID         string `json:"sourceId"`
	ProjectID        string `json:"projectId"`
	Name             string `json:"name"`
	SSHTunnelID      string `json:"sshTunnelId"`
	ConnectionString string `json:"connectionString"`
	CreatedAt        string `json:"createdAt"`
	UpdatedAt        string `json:"updatedAt"`
}

type createPgSrcConfigResponse struct {
	CreatePgSrcConfig struct {
		SourceConfig *PgSrcConfig `json:"sourceConfig"`
	} `json:"createPgSrcConfig"`
}

type updatePgSrcConfigResponse struct {
	UpdatePgSrcConfig struct {
		SourceConfig *PgSrcConfig `json:"sourceConfig"`
	} `json:"updatePgSrcConfig"`
}

type getPgSrcConfigResponse struct {
	GetPgSrcConfig struct {
		SourceConfig *PgSrcConfig `json:"sourceConfig"`
	} `json:"getPgSrcConfig"`
}

// --- Validation ---

type validatePgSrcConfigResponse struct {
	ValidateConnectorConfigPgSrc struct {
		Valid    bool     `json:"valid"`
		Errors   []string `json:"errors"`
		Warnings []string `json:"warnings"`
	} `json:"validateConnectorConfigPgSrc"`
}

// --- Connector ---

type PgSrcConnectorDetails struct {
	ID               string   `json:"id"`
	SourceConfigID   string   `json:"sourceConfigId"`
	Enabled          bool     `json:"enabled"`
	TableSyncWorkers int      `json:"tableSyncWorkers"`
	PublicationNames []string `json:"publicationNames"`
	CreatedAt        string   `json:"createdAt"`
}

type ConnectorDetails struct {
	DisplayName  string                 `json:"displayName"`
	ProjectID    string                 `json:"projectId"`
	ServiceID    string                 `json:"serviceId"`
	State        string                 `json:"state"`
	StateMessage string                 `json:"stateMessage"`
	Created      string                 `json:"created"`
	Pgsrc        *PgSrcConnectorDetails `json:"pgsrc"`
}

type createConnectorResponse struct {
	CreateConnector struct {
		ID        string            `json:"id"`
		Connector *ConnectorDetails `json:"connector"`
	} `json:"createConnector"`
}

type getConnectorResponse struct {
	GetConnector struct {
		Connector *ConnectorDetails `json:"connector"`
	} `json:"getConnector"`
}

type updateConnectorV2Response struct {
	UpdateConnectorV2 struct {
		Connector *ConnectorDetails `json:"connector"`
	} `json:"updateConnectorV2"`
}

type deleteConnectorResponse struct {
	DeleteConnector struct {
		Success bool `json:"success"`
	} `json:"deleteConnector"`
}

// --- Target Tables ---

type ConnectorTableIdentifier struct {
	SchemaName string `json:"schemaName"`
	TableName  string `json:"tableName"`
}

type ConnectorTableSpec struct {
	Table        *ConnectorTableIdentifier `json:"table"`
	TableMapping *ConnectorTableIdentifier `json:"tableMapping"`
}

type PgSrcConnectorTargetTable struct {
	Table           *ConnectorTableSpec `json:"table"`
	State           string              `json:"state"`
	RowsCopied      float64             `json:"rowsCopied"`
	BytesCopied     float64             `json:"bytesCopied"`
	ApproximateRows float64             `json:"approximateRows"`
	ApproximateSize float64             `json:"approximateSize"`
	LastError       string              `json:"lastError"`
}

type getPgSrcConnectorTargetTablesResponse struct {
	GetPgSrcConnectorTargetTables struct {
		Tables []PgSrcConnectorTargetTable `json:"tables"`
	} `json:"getPgSrcConnectorTargetTables"`
}

// --- Update Options ---

type UpdatePgSrcConnectorOpts struct {
	DisplayName      *string
	Enabled          *bool
	SourceConfigID   *string
	AddTables        []map[string]any
	DropTables       []map[string]any
	TableSyncWorkers *int
}

// --- Client Methods ---

func (c *Client) CreateSSHTunnelConfig(ctx context.Context, name, username, host string, port int) (*SSHTunnelConfig, error) {
	tflog.Trace(ctx, "Client.CreateSSHTunnelConfig")

	variables := map[string]any{
		"projectId": c.projectID,
		"name":      name,
	}
	if username != "" {
		variables["username"] = username
	}
	if host != "" {
		variables["host"] = host
	}
	if port > 0 {
		variables["port"] = port
	}

	request := map[string]any{
		"operationName": "CreateSSHTunnelConfig",
		"query":         CreateSSHTunnelConfigMutation,
		"variables":     variables,
	}

	var resp Response[createSSHTunnelConfigResponse]
	if err := c.do(ctx, request, &resp); err != nil {
		return nil, fmt.Errorf("error executing API request: %w", err)
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("API returned an error: %w", resp.Errors[0])
	}
	if resp.Data == nil || resp.Data.CreateSSHTunnelConfig.SSHTunnelConfig == nil {
		return nil, errors.New("API response did not contain SSH tunnel config data")
	}

	return resp.Data.CreateSSHTunnelConfig.SSHTunnelConfig, nil
}

func (c *Client) UpdateSSHTunnelConfig(ctx context.Context, sshTunnelID, name, username, host string, port int) (*SSHTunnelConfig, error) {
	tflog.Trace(ctx, "Client.UpdateSSHTunnelConfig")

	variables := map[string]any{
		"projectId":   c.projectID,
		"sshTunnelId": sshTunnelID,
	}
	if name != "" {
		variables["name"] = name
	}
	if username != "" {
		variables["username"] = username
	}
	if host != "" {
		variables["host"] = host
	}
	if port > 0 {
		variables["port"] = port
	}

	request := map[string]any{
		"operationName": "UpdateSSHTunnelConfig",
		"query":         UpdateSSHTunnelConfigMutation,
		"variables":     variables,
	}

	var resp Response[updateSSHTunnelConfigResponse]
	if err := c.do(ctx, request, &resp); err != nil {
		return nil, fmt.Errorf("error executing API request: %w", err)
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("API returned an error: %w", resp.Errors[0])
	}
	if resp.Data == nil || resp.Data.UpdateSSHTunnelConfig.SSHTunnelConfig == nil {
		return nil, errors.New("API response did not contain SSH tunnel config data")
	}

	return resp.Data.UpdateSSHTunnelConfig.SSHTunnelConfig, nil
}

func (c *Client) GetSSHTunnelConfig(ctx context.Context, sshTunnelID string) (*SSHTunnelConfig, error) {
	tflog.Trace(ctx, "Client.GetSSHTunnelConfig")

	request := map[string]any{
		"operationName": "GetSSHTunnelConfig",
		"query":         GetSSHTunnelConfigQuery,
		"variables": map[string]any{
			"projectId":   c.projectID,
			"sshTunnelId": sshTunnelID,
		},
	}

	var resp Response[getSSHTunnelConfigResponse]
	if err := c.do(ctx, request, &resp); err != nil {
		return nil, fmt.Errorf("error executing API request: %w", err)
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("API returned an error: %w", resp.Errors[0])
	}
	if resp.Data == nil || resp.Data.GetSSHTunnelConfig.SSHTunnelConfig == nil {
		return nil, errors.New("SSH tunnel config not found")
	}

	return resp.Data.GetSSHTunnelConfig.SSHTunnelConfig, nil
}

func (c *Client) CreatePgSrcConfig(ctx context.Context, name, connectionString, sshTunnelID string) (*PgSrcConfig, error) {
	tflog.Trace(ctx, "Client.CreatePgSrcConfig")

	variables := map[string]any{
		"projectId":        c.projectID,
		"name":             name,
		"connectionString": connectionString,
	}
	if sshTunnelID != "" {
		variables["sshTunnelId"] = sshTunnelID
	}

	request := map[string]any{
		"operationName": "CreatePgSrcConfig",
		"query":         CreatePgSrcConfigMutation,
		"variables":     variables,
	}

	var resp Response[createPgSrcConfigResponse]
	if err := c.do(ctx, request, &resp); err != nil {
		return nil, fmt.Errorf("error executing API request: %w", err)
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("API returned an error: %w", resp.Errors[0])
	}
	if resp.Data == nil || resp.Data.CreatePgSrcConfig.SourceConfig == nil {
		return nil, errors.New("API response did not contain source config data")
	}

	return resp.Data.CreatePgSrcConfig.SourceConfig, nil
}

func (c *Client) UpdatePgSrcConfig(ctx context.Context, sourceID, name, connectionString, sshTunnelID string) (*PgSrcConfig, error) {
	tflog.Trace(ctx, "Client.UpdatePgSrcConfig")

	variables := map[string]any{
		"projectId": c.projectID,
		"sourceId":  sourceID,
	}
	if name != "" {
		variables["name"] = name
	}
	if connectionString != "" {
		variables["connectionString"] = connectionString
	}
	// Allow setting sshTunnelId to empty to unlink
	variables["sshTunnelId"] = sshTunnelID

	request := map[string]any{
		"operationName": "UpdatePgSrcConfig",
		"query":         UpdatePgSrcConfigMutation,
		"variables":     variables,
	}

	var resp Response[updatePgSrcConfigResponse]
	if err := c.do(ctx, request, &resp); err != nil {
		return nil, fmt.Errorf("error executing API request: %w", err)
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("API returned an error: %w", resp.Errors[0])
	}
	if resp.Data == nil || resp.Data.UpdatePgSrcConfig.SourceConfig == nil {
		return nil, errors.New("API response did not contain source config data")
	}

	return resp.Data.UpdatePgSrcConfig.SourceConfig, nil
}

func (c *Client) GetPgSrcConfig(ctx context.Context, sourceID string) (*PgSrcConfig, error) {
	tflog.Trace(ctx, "Client.GetPgSrcConfig")

	request := map[string]any{
		"operationName": "GetPgSrcConfig",
		"query":         GetPgSrcConfigQuery,
		"variables": map[string]any{
			"projectId": c.projectID,
			"sourceId":  sourceID,
		},
	}

	var resp Response[getPgSrcConfigResponse]
	if err := c.do(ctx, request, &resp); err != nil {
		return nil, fmt.Errorf("error executing API request: %w", err)
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("API returned an error: %w", resp.Errors[0])
	}
	if resp.Data == nil || resp.Data.GetPgSrcConfig.SourceConfig == nil {
		return nil, errors.New("source config not found")
	}

	return resp.Data.GetPgSrcConfig.SourceConfig, nil
}

func (c *Client) ValidatePgSrcConfig(ctx context.Context, serviceID, connectionString, sshTunnelID string) (bool, []string, []string, error) {
	tflog.Trace(ctx, "Client.ValidatePgSrcConfig")

	variables := map[string]any{
		"projectId":        c.projectID,
		"serviceId":        serviceID,
		"connectionString": connectionString,
	}
	if sshTunnelID != "" {
		variables["sshTunnelId"] = sshTunnelID
	}

	request := map[string]any{
		"operationName": "ValidateConnectorConfigPgSrc",
		"query":         ValidatePgSrcConfigMutation,
		"variables":     variables,
	}

	var resp Response[validatePgSrcConfigResponse]
	if err := c.do(ctx, request, &resp); err != nil {
		return false, nil, nil, fmt.Errorf("error executing API request: %w", err)
	}
	if len(resp.Errors) > 0 {
		return false, nil, nil, fmt.Errorf("API returned an error: %w", resp.Errors[0])
	}
	if resp.Data == nil {
		return false, nil, nil, errors.New("API response did not contain validation data")
	}

	result := resp.Data.ValidateConnectorConfigPgSrc
	return result.Valid, result.Errors, result.Warnings, nil
}

func (c *Client) CreatePgSrcConnector(ctx context.Context, serviceID, displayName, sourceConfigID string) (string, *ConnectorDetails, error) {
	tflog.Trace(ctx, "Client.CreatePgSrcConnector")

	request := map[string]any{
		"operationName": "CreateConnector",
		"query":         CreatePgSrcConnectorMutation,
		"variables": map[string]any{
			"projectId":      c.projectID,
			"serviceId":      serviceID,
			"displayName":    displayName,
			"sourceConfigId": sourceConfigID,
		},
	}

	var resp Response[createConnectorResponse]
	if err := c.do(ctx, request, &resp); err != nil {
		return "", nil, fmt.Errorf("error executing API request: %w", err)
	}
	if len(resp.Errors) > 0 {
		return "", nil, fmt.Errorf("API returned an error: %w", resp.Errors[0])
	}
	if resp.Data == nil || resp.Data.CreateConnector.Connector == nil {
		return "", nil, errors.New("API response did not contain connector data")
	}

	return resp.Data.CreateConnector.ID, resp.Data.CreateConnector.Connector, nil
}

func (c *Client) GetPgSrcConnector(ctx context.Context, serviceID, connectorID string) (*ConnectorDetails, error) {
	tflog.Trace(ctx, "Client.GetPgSrcConnector")

	request := map[string]any{
		"operationName": "GetConnector",
		"query":         GetPgSrcConnectorQuery,
		"variables": map[string]any{
			"projectId":   c.projectID,
			"serviceId":   serviceID,
			"connectorId": connectorID,
		},
	}

	var resp Response[getConnectorResponse]
	if err := c.do(ctx, request, &resp); err != nil {
		return nil, fmt.Errorf("error executing API request: %w", err)
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("API returned an error: %w", resp.Errors[0])
	}
	if resp.Data == nil || resp.Data.GetConnector.Connector == nil {
		return nil, errors.New("connector not found")
	}

	return resp.Data.GetConnector.Connector, nil
}

func (c *Client) UpdatePgSrcConnector(ctx context.Context, serviceID, connectorID string, opts UpdatePgSrcConnectorOpts) (*ConnectorDetails, error) {
	tflog.Trace(ctx, "Client.UpdatePgSrcConnector")

	variables := map[string]any{
		"projectId":   c.projectID,
		"serviceId":   serviceID,
		"connectorId": connectorID,
	}

	if opts.DisplayName != nil {
		variables["displayName"] = *opts.DisplayName
	}
	if opts.Enabled != nil {
		variables["enabled"] = *opts.Enabled
	}

	// Build pgsrc spec if any pgsrc-specific fields are set
	pgsrc := map[string]any{}
	hasPgsrc := false

	if opts.SourceConfigID != nil {
		pgsrc["sourceConfigId"] = *opts.SourceConfigID
		hasPgsrc = true
	}
	if opts.TableSyncWorkers != nil {
		pgsrc["tableSyncWorkers"] = *opts.TableSyncWorkers
		hasPgsrc = true
	}
	if len(opts.AddTables) > 0 {
		pgsrc["addTables"] = opts.AddTables
		hasPgsrc = true
	}
	if len(opts.DropTables) > 0 {
		pgsrc["dropTables"] = opts.DropTables
		hasPgsrc = true
	}

	if hasPgsrc {
		variables["pgsrc"] = pgsrc
	}

	request := map[string]any{
		"operationName": "UpdateConnectorV2",
		"query":         UpdatePgSrcConnectorMutation,
		"variables":     variables,
	}

	var resp Response[updateConnectorV2Response]
	if err := c.do(ctx, request, &resp); err != nil {
		return nil, fmt.Errorf("error executing API request: %w", err)
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("API returned an error: %w", resp.Errors[0])
	}
	if resp.Data == nil || resp.Data.UpdateConnectorV2.Connector == nil {
		return nil, errors.New("API response did not contain connector data")
	}

	return resp.Data.UpdateConnectorV2.Connector, nil
}

func (c *Client) GetPgSrcConnectorTargetTables(ctx context.Context, serviceID, connectorID string) ([]PgSrcConnectorTargetTable, error) {
	tflog.Trace(ctx, "Client.GetPgSrcConnectorTargetTables")

	request := map[string]any{
		"operationName": "GetPgSrcConnectorTargetTables",
		"query":         GetPgSrcConnectorTargetTablesQuery,
		"variables": map[string]any{
			"projectId":   c.projectID,
			"serviceId":   serviceID,
			"connectorId": connectorID,
		},
	}

	var resp Response[getPgSrcConnectorTargetTablesResponse]
	if err := c.do(ctx, request, &resp); err != nil {
		return nil, fmt.Errorf("error executing API request: %w", err)
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("API returned an error: %w", resp.Errors[0])
	}
	if resp.Data == nil {
		return nil, errors.New("API response did not contain target tables data")
	}

	return resp.Data.GetPgSrcConnectorTargetTables.Tables, nil
}

func (c *Client) DeletePgSrcConnector(ctx context.Context, serviceID, connectorID string) error {
	tflog.Trace(ctx, "Client.DeletePgSrcConnector")

	request := map[string]any{
		"operationName": "DeleteConnector",
		"query":         DeletePgSrcConnectorMutation,
		"variables": map[string]any{
			"projectId":   c.projectID,
			"serviceId":   serviceID,
			"connectorId": connectorID,
		},
	}

	var resp Response[deleteConnectorResponse]
	if err := c.do(ctx, request, &resp); err != nil {
		return fmt.Errorf("error executing API request: %w", err)
	}
	if len(resp.Errors) > 0 {
		return fmt.Errorf("API returned an error: %w", resp.Errors[0])
	}

	return nil
}
