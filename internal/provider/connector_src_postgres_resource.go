package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	tsClient "github.com/timescale/terraform-provider-timescale/internal/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &connectorSrcPostgresResource{}
	_ resource.ResourceWithConfigure   = &connectorSrcPostgresResource{}
	_ resource.ResourceWithImportState = &connectorSrcPostgresResource{}
)

// NewConnectorSrcPostgresResource is a helper function to simplify the provider implementation.
func NewConnectorSrcPostgresResource() resource.Resource {
	return &connectorSrcPostgresResource{}
}

// connectorSrcPostgresResource is the resource implementation.
type connectorSrcPostgresResource struct {
	client *tsClient.Client
}

type connectorSrcPostgresResourceModel struct {
	ID               types.String    `tfsdk:"id"`
	ServiceID        types.String    `tfsdk:"service_id"`
	DisplayName      types.String    `tfsdk:"display_name"`
	Name             types.String    `tfsdk:"name"`
	ConnectionString types.String    `tfsdk:"connection_string"`
	SourceID         types.String    `tfsdk:"source_id"`
	SSHTunnel        *sshTunnelModel `tfsdk:"ssh_tunnel"`
	Tables           []tableModel    `tfsdk:"tables"`
	TableSyncWorkers types.Int64     `tfsdk:"table_sync_workers"`
	Enabled          types.Bool      `tfsdk:"enabled"`
	Status           types.String    `tfsdk:"status"`
	CreatedAt        types.String    `tfsdk:"created_at"`
}

type sshTunnelModel struct {
	SSHTunnelID types.String `tfsdk:"ssh_tunnel_id"`
	Name        types.String `tfsdk:"name"`
	Username    types.String `tfsdk:"username"`
	Host        types.String `tfsdk:"host"`
	Port        types.Int64  `tfsdk:"port"`
	PublicKey   types.String `tfsdk:"public_key"`
}

type tableModel struct {
	SchemaName      types.String       `tfsdk:"schema_name"`
	TableName       types.String       `tfsdk:"table_name"`
	TableMapping    *tableMappingModel `tfsdk:"table_mapping"`
	PublicationName types.String       `tfsdk:"publication_name"`
}

type tableMappingModel struct {
	SchemaName types.String `tfsdk:"schema_name"`
	TableName  types.String `tfsdk:"table_name"`
}

// Metadata returns the resource type name.
func (r *connectorSrcPostgresResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connector_src_postgres"
}

// Schema defines the schema for the resource.
func (r *connectorSrcPostgresResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a PostgreSQL source connector for logical replication from an external PostgreSQL database into a Timescale Cloud service.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Unique identifier for the connector.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"service_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The Timescale Cloud service ID to replicate data into.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"display_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Human-readable display name for the connector.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name for the source configuration.",
			},
			"connection_string": schema.StringAttribute{
				Required:            true,
				Sensitive:           true,
				MarkdownDescription: "PostgreSQL connection string for the source database (e.g. `postgresql://user:password@host:5432/dbname`).",
			},
			"source_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Unique identifier for the source configuration.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ssh_tunnel": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Optional SSH tunnel configuration for connecting to the source database through a bastion host.",
				Attributes: map[string]schema.Attribute{
					"ssh_tunnel_id": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Unique identifier for the SSH tunnel configuration.",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"name": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "Name for the SSH tunnel configuration.",
					},
					"username": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "SSH username for the tunnel connection.",
					},
					"host": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "SSH host (bastion server) for the tunnel connection.",
					},
					"port": schema.Int64Attribute{
						Optional:            true,
						MarkdownDescription: "SSH port for the tunnel connection.",
					},
					"public_key": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Server-generated public key. Add this to the `authorized_keys` file on your SSH bastion host.",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"tables": schema.ListNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Tables to replicate from the source database. Tables are immutable once added. To change a table's configuration, remove it and re-add it. **Warning**: removing a table causes a full re-sync of that table's data when re-added.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"schema_name": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Schema name of the source table.",
						},
						"table_name": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Name of the source table.",
						},
						"table_mapping": schema.SingleNestedAttribute{
							Optional:            true,
							MarkdownDescription: "Optional mapping to a different table name on the target service.",
							Attributes: map[string]schema.Attribute{
								"schema_name": schema.StringAttribute{
									Required:            true,
									MarkdownDescription: "Target schema name.",
								},
								"table_name": schema.StringAttribute{
									Required:            true,
									MarkdownDescription: "Target table name.",
								},
							},
						},
						"publication_name": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Existing PostgreSQL publication name to use for this table. If not provided the connector will create one.",
						},
					},
				},
			},
			"table_sync_workers": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Number of parallel workers for table synchronization.",
			},
			"enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether the connector is enabled (default: true).",
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Current connector status (ok, warning, error, paused).",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the connector was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *connectorSrcPostgresResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	tflog.Trace(ctx, "connectorSrcPostgresResource.Configure")
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*tsClient.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *tsClient.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.client = client
}

// Create creates a new PostgreSQL source connector.
func (r *connectorSrcPostgresResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "connectorSrcPostgresResource.Create")

	var plan connectorSrcPostgresResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceID := plan.ServiceID.ValueString()
	connectionString := plan.ConnectionString.ValueString()

	// Step 1: Create SSH tunnel config if configured
	var sshTunnelID string
	if plan.SSHTunnel != nil {
		tunnel, err := r.client.CreateSSHTunnelConfig(
			ctx,
			plan.SSHTunnel.Name.ValueString(),
			plan.SSHTunnel.Username.ValueString(),
			plan.SSHTunnel.Host.ValueString(),
			int(plan.SSHTunnel.Port.ValueInt64()),
		)
		if err != nil {
			resp.Diagnostics.AddError("Unable to create SSH tunnel config", err.Error())
			return
		}
		sshTunnelID = tunnel.SSHTunnelID
	}

	// Step 2: Validate the connector configuration
	valid, validationErrors, validationWarnings, err := r.client.ValidatePgSrcConfig(
		ctx, serviceID, connectionString, sshTunnelID,
	)
	if err != nil {
		resp.Diagnostics.AddError("Unable to validate connector configuration", err.Error())
		return
	}

	if !valid {
		for _, w := range validationWarnings {
			resp.Diagnostics.AddWarning("Connector configuration warning", w)
		}
		if len(validationErrors) > 0 {
			resp.Diagnostics.AddError(
				"Invalid connector configuration",
				fmt.Sprintf("Validation failed: %s", strings.Join(validationErrors, "; ")),
			)
			return
		}
	}

	// Step 3: Create PgSrc config
	pgSrcConfig, err := r.client.CreatePgSrcConfig(
		ctx,
		plan.Name.ValueString(),
		connectionString,
		sshTunnelID,
	)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create PostgreSQL source config", err.Error())
		return
	}

	// Step 4: Create the connector
	connectorID, _, err := r.client.CreatePgSrcConnector(
		ctx,
		serviceID,
		plan.DisplayName.ValueString(),
		pgSrcConfig.SourceID,
	)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create connector", err.Error())
		return
	}

	// Step 5: Update connector to add tables and set enabled/tableSyncWorkers
	updateOpts := r.buildUpdateOpts(&plan)
	if updateOpts != nil {
		_, err = r.client.UpdatePgSrcConnector(ctx, serviceID, connectorID, *updateOpts)
		if err != nil {
			resp.Diagnostics.AddError("Unable to configure connector", err.Error())
			// Cleanup: delete the partially configured connector
			deleteErr := r.client.DeletePgSrcConnector(ctx, serviceID, connectorID)
			if deleteErr != nil {
				resp.Diagnostics.AddError(
					"Connector was created but failed to be configured. Cleanup failed. Connector exists in inconsistent state and needs to be manually deleted",
					deleteErr.Error(),
				)
			}
			return
		}
	}

	// Step 6: Read back full state
	r.readIntoModel(ctx, serviceID, connectorID, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *connectorSrcPostgresResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "connectorSrcPostgresResource.Read")

	var state connectorSrcPostgresResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceID := state.ServiceID.ValueString()
	connectorID := state.ID.ValueString()

	// Preserve the sensitive connection_string from state since the API may redact it
	savedConnectionString := state.ConnectionString

	r.readIntoModel(ctx, serviceID, connectorID, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Restore sensitive connection_string if the API returned an empty or redacted value
	if state.ConnectionString.ValueString() == "" && !savedConnectionString.IsNull() {
		state.ConnectionString = savedConnectionString
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates an existing PostgreSQL source connector.
func (r *connectorSrcPostgresResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "connectorSrcPostgresResource.Update")

	var plan, state connectorSrcPostgresResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceID := plan.ServiceID.ValueString()
	connectorID := state.ID.ValueString()
	sourceID := state.SourceID.ValueString()
	sshTunnelID := ""

	// Step 1: Handle SSH tunnel changes
	if state.SSHTunnel != nil {
		sshTunnelID = state.SSHTunnel.SSHTunnelID.ValueString()
	}

	if plan.SSHTunnel != nil && state.SSHTunnel == nil {
		// SSH tunnel added
		tunnel, err := r.client.CreateSSHTunnelConfig(
			ctx,
			plan.SSHTunnel.Name.ValueString(),
			plan.SSHTunnel.Username.ValueString(),
			plan.SSHTunnel.Host.ValueString(),
			int(plan.SSHTunnel.Port.ValueInt64()),
		)
		if err != nil {
			resp.Diagnostics.AddError("Unable to create SSH tunnel config", err.Error())
			return
		}
		sshTunnelID = tunnel.SSHTunnelID
	} else if plan.SSHTunnel != nil && state.SSHTunnel != nil {
		// SSH tunnel changed
		sshTunnelID = state.SSHTunnel.SSHTunnelID.ValueString()
		_, err := r.client.UpdateSSHTunnelConfig(
			ctx,
			sshTunnelID,
			plan.SSHTunnel.Name.ValueString(),
			plan.SSHTunnel.Username.ValueString(),
			plan.SSHTunnel.Host.ValueString(),
			int(plan.SSHTunnel.Port.ValueInt64()),
		)
		if err != nil {
			resp.Diagnostics.AddError("Unable to update SSH tunnel config", err.Error())
			return
		}
	} else if plan.SSHTunnel == nil && state.SSHTunnel != nil {
		// SSH tunnel removed - unlink from source config
		sshTunnelID = ""
	}

	// Step 2: Update PgSrc config if connection_string, name, or sshTunnelId changed
	configChanged := plan.Name.ValueString() != state.Name.ValueString() ||
		plan.ConnectionString.ValueString() != state.ConnectionString.ValueString() ||
		(plan.SSHTunnel == nil) != (state.SSHTunnel == nil) ||
		(plan.SSHTunnel != nil && state.SSHTunnel == nil)

	if configChanged {
		_, err := r.client.UpdatePgSrcConfig(
			ctx,
			sourceID,
			plan.Name.ValueString(),
			plan.ConnectionString.ValueString(),
			sshTunnelID,
		)
		if err != nil {
			resp.Diagnostics.AddError("Unable to update PostgreSQL source config", err.Error())
			return
		}
	}

	// Step 3: Update connector (display name, enabled, tables, workers)
	addTables, dropTables := computeTableDiff(state.Tables, plan.Tables)

	// Emit warnings for tables being dropped
	for _, dt := range dropTables {
		schemaName := dt["schemaName"]
		tableName := dt["tableName"]
		resp.Diagnostics.AddWarning(
			fmt.Sprintf("Table %s.%s is being removed from the connector", schemaName, tableName),
			"Removing a table from the connector will cause a full re-sync of all the table's data if it is added back. All existing replicated data for this table on the target will need to be reprocessed from the beginning.",
		)
	}

	connectorChanged := plan.DisplayName.ValueString() != state.DisplayName.ValueString() ||
		plan.Enabled.ValueBool() != state.Enabled.ValueBool() ||
		plan.TableSyncWorkers.ValueInt64() != state.TableSyncWorkers.ValueInt64() ||
		len(addTables) > 0 || len(dropTables) > 0

	if connectorChanged {
		opts := tsClient.UpdatePgSrcConnectorOpts{}
		displayName := plan.DisplayName.ValueString()
		opts.DisplayName = &displayName
		enabled := plan.Enabled.ValueBool()
		opts.Enabled = &enabled

		if !plan.TableSyncWorkers.IsNull() && !plan.TableSyncWorkers.IsUnknown() {
			workers := int(plan.TableSyncWorkers.ValueInt64())
			opts.TableSyncWorkers = &workers
		}

		if len(dropTables) > 0 {
			opts.DropTables = dropTables
		}
		if len(addTables) > 0 {
			opts.AddTables = addTables
		}

		_, err := r.client.UpdatePgSrcConnector(ctx, serviceID, connectorID, opts)
		if err != nil {
			resp.Diagnostics.AddError("Unable to update connector", err.Error())
			return
		}
	}

	// Step 4: Read back full state
	// Preserve the sensitive connection_string from plan
	savedConnectionString := plan.ConnectionString

	r.readIntoModel(ctx, serviceID, connectorID, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Restore sensitive connection_string
	if plan.ConnectionString.ValueString() == "" && !savedConnectionString.IsNull() {
		plan.ConnectionString = savedConnectionString
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes a PostgreSQL source connector.
func (r *connectorSrcPostgresResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "connectorSrcPostgresResource.Delete")

	var state connectorSrcPostgresResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeletePgSrcConnector(ctx, state.ServiceID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting connector", err.Error())
	}
}

// ImportState supports importing the resource by service_id:connector_id.
func (r *connectorSrcPostgresResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, ":")

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: service_id:connector_id. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), idParts[1])...)
}

// --- Helper functions ---

// readIntoModel reads the full connector state from the API and maps it into the model.
func (r *connectorSrcPostgresResource) readIntoModel(
	ctx context.Context,
	serviceID, connectorID string,
	model *connectorSrcPostgresResourceModel,
	diags *diag.Diagnostics,
) {
	// Get connector details
	connector, err := r.client.GetPgSrcConnector(ctx, serviceID, connectorID)
	if err != nil {
		diags.AddError("Error reading connector", err.Error())
		return
	}

	model.ID = types.StringValue(connectorID)
	model.ServiceID = types.StringValue(serviceID)
	model.DisplayName = types.StringValue(connector.DisplayName)
	model.Status = types.StringValue(connector.State)
	model.CreatedAt = types.StringValue(connector.Created)

	if connector.Pgsrc != nil {
		model.SourceID = types.StringValue(connector.Pgsrc.SourceConfigID)
		model.Enabled = types.BoolValue(connector.Pgsrc.Enabled)
		if connector.Pgsrc.TableSyncWorkers > 0 {
			model.TableSyncWorkers = types.Int64Value(int64(connector.Pgsrc.TableSyncWorkers))
		}

		// Get PgSrc config for connection details
		pgSrcConfig, err := r.client.GetPgSrcConfig(ctx, connector.Pgsrc.SourceConfigID)
		if err != nil {
			diags.AddError("Error reading source config", err.Error())
			return
		}

		model.Name = types.StringValue(pgSrcConfig.Name)
		if pgSrcConfig.ConnectionString != "" {
			model.ConnectionString = types.StringValue(pgSrcConfig.ConnectionString)
		}

		// Get SSH tunnel config if linked
		if pgSrcConfig.SSHTunnelID != "" {
			sshTunnel, err := r.client.GetSSHTunnelConfig(ctx, pgSrcConfig.SSHTunnelID)
			if err != nil {
				diags.AddError("Error reading SSH tunnel config", err.Error())
				return
			}
			model.SSHTunnel = &sshTunnelModel{
				SSHTunnelID: types.StringValue(sshTunnel.SSHTunnelID),
				Name:        types.StringValue(sshTunnel.Name),
				Username:    types.StringValue(sshTunnel.Username),
				Host:        types.StringValue(sshTunnel.Host),
				Port:        types.Int64Value(int64(sshTunnel.Port)),
				PublicKey:   types.StringValue(sshTunnel.PublicKey),
			}
		} else if model.SSHTunnel != nil {
			// SSH tunnel was unlinked
			model.SSHTunnel = nil
		}
	}

	// Get target tables
	tables, err := r.client.GetPgSrcConnectorTargetTables(ctx, serviceID, connectorID)
	if err != nil {
		diags.AddError("Error reading connector target tables", err.Error())
		return
	}

	// Save existing tables before overwriting so we can preserve publication_name,
	// which is not returned by the API.
	existingTables := model.Tables

	if len(tables) > 0 {
		model.Tables = make([]tableModel, 0, len(tables))
		for _, t := range tables {
			if t.Table == nil || t.Table.Table == nil {
				continue
			}
			tm := tableModel{
				SchemaName: types.StringValue(t.Table.Table.SchemaName),
				TableName:  types.StringValue(t.Table.Table.TableName),
			}
			if t.Table.TableMapping != nil {
				tm.TableMapping = &tableMappingModel{
					SchemaName: types.StringValue(t.Table.TableMapping.SchemaName),
					TableName:  types.StringValue(t.Table.TableMapping.TableName),
				}
			}
			// publication_name is not returned by getPgSrcConnectorTargetTables,
			// so preserve it from the prior model state if present.
			for _, existing := range existingTables {
				if existing.SchemaName.ValueString() == t.Table.Table.SchemaName &&
					existing.TableName.ValueString() == t.Table.Table.TableName {
					tm.PublicationName = existing.PublicationName
					break
				}
			}
			model.Tables = append(model.Tables, tm)
		}
	} else {
		model.Tables = nil
	}
}

// buildUpdateOpts builds update options for the initial updateConnectorV2 call after creation.
func (r *connectorSrcPostgresResource) buildUpdateOpts(plan *connectorSrcPostgresResourceModel) *tsClient.UpdatePgSrcConnectorOpts {
	opts := tsClient.UpdatePgSrcConnectorOpts{}

	// Tables
	if len(plan.Tables) > 0 {
		opts.AddTables = buildAddTablesInput(plan.Tables)
	}

	// Table sync workers
	if !plan.TableSyncWorkers.IsNull() && !plan.TableSyncWorkers.IsUnknown() {
		workers := int(plan.TableSyncWorkers.ValueInt64())
		opts.TableSyncWorkers = &workers
	}

	// Enabled - always set on create to match the desired state
	enabled := plan.Enabled.ValueBool()
	opts.Enabled = &enabled

	return &opts
}

// computeTableDiff computes addTables and dropTables by comparing current state vs desired plan.
// Tables are identified by (schema_name, table_name) key.
// If a table exists in both state and plan but with a different configuration (table_mapping
// or publication_name), it is treated as a drop + re-add since tables are immutable once added.
func computeTableDiff(stateTables, planTables []tableModel) (addTables []map[string]any, dropTables []map[string]any) {
	type tableKey struct {
		schema string
		table  string
	}

	stateMap := make(map[tableKey]tableModel)
	for _, t := range stateTables {
		key := tableKey{schema: t.SchemaName.ValueString(), table: t.TableName.ValueString()}
		stateMap[key] = t
	}

	planMap := make(map[tableKey]tableModel)
	for _, t := range planTables {
		key := tableKey{schema: t.SchemaName.ValueString(), table: t.TableName.ValueString()}
		planMap[key] = t
	}

	// Tables in state but not in plan → drop
	for key := range stateMap {
		if _, exists := planMap[key]; !exists {
			dropTables = append(dropTables, map[string]any{
				"schemaName": key.schema,
				"tableName":  key.table,
			})
		}
	}

	// Tables in plan: add if new, or drop+re-add if config changed
	for key, planTable := range planMap {
		stateTable, existsInState := stateMap[key]
		if !existsInState {
			// New table → add
			addTables = append(addTables, buildTableSpecInput(planTable))
		} else if tableConfigChanged(stateTable, planTable) {
			// Config changed → drop then re-add
			dropTables = append(dropTables, map[string]any{
				"schemaName": key.schema,
				"tableName":  key.table,
			})
			addTables = append(addTables, buildTableSpecInput(planTable))
		}
	}

	return addTables, dropTables
}

// tableConfigChanged returns true if the table's immutable configuration fields differ
// between state and plan (table_mapping or publication_name).
func tableConfigChanged(state, plan tableModel) bool {
	// Check publication_name
	if state.PublicationName.ValueString() != plan.PublicationName.ValueString() {
		return true
	}

	// Check table_mapping
	stateHasMapping := state.TableMapping != nil
	planHasMapping := plan.TableMapping != nil

	if stateHasMapping != planHasMapping {
		return true
	}
	if stateHasMapping && planHasMapping {
		if state.TableMapping.SchemaName.ValueString() != plan.TableMapping.SchemaName.ValueString() ||
			state.TableMapping.TableName.ValueString() != plan.TableMapping.TableName.ValueString() {
			return true
		}
	}

	return false
}

// buildAddTablesInput converts a list of tableModels to the API input format.
func buildAddTablesInput(tables []tableModel) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(tables))
	for _, t := range tables {
		result = append(result, buildTableSpecInput(t))
	}
	return result
}

// buildTableSpecInput converts a single tableModel to the API ConnectorTableSpecInput format.
func buildTableSpecInput(t tableModel) map[string]interface{} {
	spec := map[string]interface{}{
		"table": map[string]interface{}{
			"schemaName": t.SchemaName.ValueString(),
			"tableName":  t.TableName.ValueString(),
		},
	}

	if t.TableMapping != nil {
		spec["tableMapping"] = map[string]interface{}{
			"schemaName": t.TableMapping.SchemaName.ValueString(),
			"tableName":  t.TableMapping.TableName.ValueString(),
		}
	}

	if !t.PublicationName.IsNull() && t.PublicationName.ValueString() != "" {
		spec["publicationName"] = t.PublicationName.ValueString()
	}

	return spec
}
