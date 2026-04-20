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

// All connector mutations/queries are nested under a `connectors` root field
// in the GraphQL schema (ConnectorsMutator / ConnectorsQuerier).

type createSSHTunnelConfigResponse struct {
	Connectors struct {
		CreateSSHTunnelConfig struct {
			SSHTunnelConfig *SSHTunnelConfig `json:"sshTunnelConfig"`
		} `json:"createSSHTunnelConfig"`
	} `json:"connectors"`
}

type updateSSHTunnelConfigResponse struct {
	Connectors struct {
		UpdateSSHTunnelConfig struct {
			SSHTunnelConfig *SSHTunnelConfig `json:"sshTunnelConfig"`
		} `json:"updateSSHTunnelConfig"`
	} `json:"connectors"`
}

type getSSHTunnelConfigResponse struct {
	Connectors struct {
		GetSSHTunnelConfig struct {
			SSHTunnelConfig *SSHTunnelConfig `json:"sshTunnelConfig"`
		} `json:"getSSHTunnelConfig"`
	} `json:"connectors"`
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
	Connectors struct {
		CreatePgSrcConfig struct {
			SourceConfig *PgSrcConfig `json:"sourceConfig"`
		} `json:"createPgSrcConfig"`
	} `json:"connectors"`
}

type updatePgSrcConfigResponse struct {
	Connectors struct {
		UpdatePgSrcConfig struct {
			SourceConfig *PgSrcConfig `json:"sourceConfig"`
		} `json:"updatePgSrcConfig"`
	} `json:"connectors"`
}

type getPgSrcConfigResponse struct {
	Connectors struct {
		GetPgSrcConfig struct {
			SourceConfig *PgSrcConfig `json:"sourceConfig"`
		} `json:"getPgSrcConfig"`
	} `json:"connectors"`
}

// --- Validation ---

type validatePgSrcConfigResponse struct {
	Connectors struct {
		ValidateConnectorConfigPgSrc struct {
			Valid    bool     `json:"valid"`
			Errors   []string `json:"errors"`
			Warnings []string `json:"warnings"`
		} `json:"validateConnectorConfigPgSrc"`
	} `json:"connectors"`
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
	Connectors struct {
		CreateConnector struct {
			ID        string            `json:"id"`
			Connector *ConnectorDetails `json:"connector"`
		} `json:"createConnector"`
	} `json:"connectors"`
}

type getConnectorResponse struct {
	Connectors struct {
		GetConnector struct {
			Connector *ConnectorDetails `json:"connector"`
		} `json:"getConnector"`
	} `json:"connectors"`
}

type updateConnectorV2Response struct {
	Connectors struct {
		UpdateConnectorV2 struct {
			Connector *ConnectorDetails `json:"connector"`
		} `json:"updateConnectorV2"`
	} `json:"connectors"`
}

type deleteConnectorResponse struct {
	Connectors struct {
		DeleteConnector struct {
			Success bool `json:"success"`
		} `json:"deleteConnector"`
	} `json:"connectors"`
}

// --- Target Tables ---

type ConnectorTableIdentifier struct {
	SchemaName string `json:"schemaName"`
	TableName  string `json:"tableName"`
}

type HypertableRangeDimension struct {
	ColumnName        string `json:"columnName"`
	PartitionInterval string `json:"partitionInterval"`
}

type HypertableHashDimension struct {
	ColumnName       string `json:"columnName"`
	NumberPartitions int    `json:"numberPartitions"`
}

type HypertableDimension struct {
	Range *HypertableRangeDimension `json:"range"`
	Hash  *HypertableHashDimension  `json:"hash"`
}

type HypertableSpec struct {
	PrimaryDimension    *HypertableRangeDimension `json:"primaryDimension"`
	SecondaryDimensions []HypertableDimension     `json:"secondaryDimensions"`
}

type ConnectorTableSpec struct {
	Table          *ConnectorTableIdentifier `json:"table"`
	TableMapping   *ConnectorTableIdentifier `json:"tableMapping"`
	HypertableSpec *HypertableSpec           `json:"hypertableSpec"`
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
	Connectors struct {
		GetPgSrcConnectorTargetTables struct {
			Tables []PgSrcConnectorTargetTable `json:"tables"`
		} `json:"getPgSrcConnectorTargetTables"`
	} `json:"connectors"`
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
	if resp.Data == nil || resp.Data.Connectors.CreateSSHTunnelConfig.SSHTunnelConfig == nil {
		return nil, errors.New("API response did not contain SSH tunnel config data")
	}

	return resp.Data.Connectors.CreateSSHTunnelConfig.SSHTunnelConfig, nil
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
	// Send nil to allow clearing optional fields that were previously set.
	if username != "" {
		variables["username"] = username
	} else {
		variables["username"] = nil
	}
	if host != "" {
		variables["host"] = host
	} else {
		variables["host"] = nil
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
	if resp.Data == nil || resp.Data.Connectors.UpdateSSHTunnelConfig.SSHTunnelConfig == nil {
		return nil, errors.New("API response did not contain SSH tunnel config data")
	}

	return resp.Data.Connectors.UpdateSSHTunnelConfig.SSHTunnelConfig, nil
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
	if resp.Data == nil || resp.Data.Connectors.GetSSHTunnelConfig.SSHTunnelConfig == nil {
		return nil, errors.New("SSH tunnel config not found")
	}

	return resp.Data.Connectors.GetSSHTunnelConfig.SSHTunnelConfig, nil
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
	if resp.Data == nil || resp.Data.Connectors.CreatePgSrcConfig.SourceConfig == nil {
		return nil, errors.New("API response did not contain source config data")
	}

	return resp.Data.Connectors.CreatePgSrcConfig.SourceConfig, nil
}

// UpdatePgSrcConfig updates a PostgreSQL source configuration.
// All string fields use empty-string-means-skip semantics (the API uses nullable fields).
// sshTunnelID follows pointer semantics: nil means "don't change", non-nil means "set to this value"
// (where an empty string clears the tunnel link).
func (c *Client) UpdatePgSrcConfig(ctx context.Context, sourceID, name, connectionString string, sshTunnelID *string) (*PgSrcConfig, error) {
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
	if sshTunnelID != nil {
		if *sshTunnelID != "" {
			variables["sshTunnelId"] = *sshTunnelID
		} else {
			variables["sshTunnelId"] = ""
		}
	}

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
	if resp.Data == nil || resp.Data.Connectors.UpdatePgSrcConfig.SourceConfig == nil {
		return nil, errors.New("API response did not contain source config data")
	}

	return resp.Data.Connectors.UpdatePgSrcConfig.SourceConfig, nil
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
	if resp.Data == nil || resp.Data.Connectors.GetPgSrcConfig.SourceConfig == nil {
		return nil, errors.New("source config not found")
	}

	return resp.Data.Connectors.GetPgSrcConfig.SourceConfig, nil
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

	result := resp.Data.Connectors.ValidateConnectorConfigPgSrc
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
	if resp.Data == nil || resp.Data.Connectors.CreateConnector.Connector == nil {
		return "", nil, errors.New("API response did not contain connector data")
	}

	return resp.Data.Connectors.CreateConnector.ID, resp.Data.Connectors.CreateConnector.Connector, nil
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
	if resp.Data == nil || resp.Data.Connectors.GetConnector.Connector == nil {
		return nil, errors.New("connector not found")
	}

	return resp.Data.Connectors.GetConnector.Connector, nil
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
	if resp.Data == nil || resp.Data.Connectors.UpdateConnectorV2.Connector == nil {
		return nil, errors.New("API response did not contain connector data")
	}

	return resp.Data.Connectors.UpdateConnectorV2.Connector, nil
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

	return resp.Data.Connectors.GetPgSrcConnectorTargetTables.Tables, nil
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
