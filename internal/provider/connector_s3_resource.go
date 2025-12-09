package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	tsClient "github.com/timescale/terraform-provider-timescale/internal/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &connectorS3Resource{}
	_ resource.ResourceWithConfigure = &connectorS3Resource{}
)

// NewConnectorS3Resource is a helper function to simplify the provider implementation.
func NewConnectorS3Resource() resource.Resource {
	return &connectorS3Resource{}
}

// connectorS3Resource is the resource implementation.
type connectorS3Resource struct {
	client *tsClient.Client
}

type connectorS3ResourceModel struct {
	ID                  types.String          `tfsdk:"id"`
	ServiceID           types.String          `tfsdk:"service_id"`
	Name                types.String          `tfsdk:"name"`
	Bucket              types.String          `tfsdk:"bucket"`
	Pattern             types.String          `tfsdk:"pattern"`
	Credentials         *credentialsModel     `tfsdk:"credentials"`
	Definition          *definitionModel      `tfsdk:"definition"`
	TableIdentifier     *tableIdentifierModel `tfsdk:"table_identifier"`
	Frequency           types.String          `tfsdk:"frequency"`
	Enabled             types.Bool            `tfsdk:"enabled"`
	OnConflictDoNothing types.Bool            `tfsdk:"on_conflict_do_nothing"`
	CreatedAt           types.String          `tfsdk:"created_at"`
	UpdatedAt           types.String          `tfsdk:"updated_at"`
}

type credentialsModel struct {
	Type    types.String `tfsdk:"type"`
	RoleARN types.String `tfsdk:"role_arn"`
}

type definitionModel struct {
	Type    types.String            `tfsdk:"type"`
	CSV     *csvDefinitionModel     `tfsdk:"csv"`
	Parquet *parquetDefinitionModel `tfsdk:"parquet"`
}

type csvDefinitionModel struct {
	Delimiter         types.String         `tfsdk:"delimiter"`
	SkipHeader        types.Bool           `tfsdk:"skip_header"`
	ColumnNames       []types.String       `tfsdk:"column_names"`
	ColumnMappings    []columnMappingModel `tfsdk:"column_mappings"`
	AutoColumnMapping types.Bool           `tfsdk:"auto_column_mapping"`
}

type parquetDefinitionModel struct {
	ColumnMappings    []columnMappingModel `tfsdk:"column_mappings"`
	AutoColumnMapping types.Bool           `tfsdk:"auto_column_mapping"`
}

type columnMappingModel struct {
	Source      types.String `tfsdk:"source"`
	Destination types.String `tfsdk:"destination"`
}

type tableIdentifierModel struct {
	SchemaName types.String `tfsdk:"schema_name"`
	TableName  types.String `tfsdk:"table_name"`
}

// Metadata returns the resource type name.
func (r *connectorS3Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connector_s3"
}

// Schema defines the schema for the resource.
func (r *connectorS3Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an S3 connector for continuous data import from S3 buckets.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Unique identifier for the S3 connector.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"service_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The service ID to attach the connector to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Human-readable name for the connector.",
			},
			"bucket": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "S3 bucket name.",
			},
			"pattern": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Pattern to match S3 object keys (supports wildcards).",
			},
			"frequency": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("@always"),
				MarkdownDescription: "Cron expression for sync frequency: @always, @5minutes, @10minutes, @15minutes, @30minutes, @hourly, @daily, @weekly, @monthly, @annually, @yearly",
			},
			"enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether the connector is enabled (default: true) .",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the connector was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the connector was last updated.",
			},
			"on_conflict_do_nothing": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Handle conflicts by doing nothing (ignore conflicting rows). Defaults to false.",
			},
			"credentials": schema.SingleNestedAttribute{
				Required:            true,
				MarkdownDescription: "S3 authentication credentials.",
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "Credential type: 'Public' or 'RoleARN'.",
					},
					"role_arn": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "IAM role ARN (required if type is 'RoleARN').",
					},
				},
			},
			"definition": schema.SingleNestedAttribute{
				Required:            true,
				MarkdownDescription: "File format definition (CSV or PARQUET).",
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "File type: 'CSV' or 'PARQUET'.",
					},
					"csv": schema.SingleNestedAttribute{
						Optional:            true,
						MarkdownDescription: "CSV file configuration.",
						Attributes: map[string]schema.Attribute{
							"delimiter": schema.StringAttribute{
								Optional:            true,
								Computed:            true,
								Default:             stringdefault.StaticString(","),
								MarkdownDescription: "CSV delimiter (default: ',').",
							},
							"skip_header": schema.BoolAttribute{
								Optional:            true,
								Computed:            true,
								Default:             booldefault.StaticBool(false),
								MarkdownDescription: "Whether to skip the first row as header (default: false).",
							},
							"column_names": schema.ListAttribute{
								Optional:            true,
								ElementType:         types.StringType,
								MarkdownDescription: "Column names (mutually exclusive with column_mappings and auto_column_mapping).",
							},
							"column_mappings": schema.ListNestedAttribute{
								Optional:            true,
								MarkdownDescription: "Column mappings from source to destination (mutually exclusive with column_names and auto_column_mapping).",
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"source": schema.StringAttribute{
											Required:            true,
											MarkdownDescription: "Source column name.",
										},
										"destination": schema.StringAttribute{
											Required:            true,
											MarkdownDescription: "Destination column name.",
										},
									},
								},
							},
							"auto_column_mapping": schema.BoolAttribute{
								Optional:            true,
								Computed:            true,
								Default:             booldefault.StaticBool(false),
								MarkdownDescription: "Automatically map columns by name (mutually exclusive with column_names and column_mappings).",
							},
						},
					},
					"parquet": schema.SingleNestedAttribute{
						Optional:            true,
						MarkdownDescription: "Parquet file configuration.",
						Attributes: map[string]schema.Attribute{
							"column_mappings": schema.ListNestedAttribute{
								Optional:            true,
								MarkdownDescription: "Column mappings from source to destination (mutually exclusive with auto_column_mapping).",
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"source": schema.StringAttribute{
											Required:            true,
											MarkdownDescription: "Source column name.",
										},
										"destination": schema.StringAttribute{
											Required:            true,
											MarkdownDescription: "Destination column name.",
										},
									},
								},
							},
							"auto_column_mapping": schema.BoolAttribute{
								Optional:            true,
								Computed:            true,
								Default:             booldefault.StaticBool(false),
								MarkdownDescription: "Automatically map columns by name (mutually exclusive with column_mappings).",
							},
						},
					},
				},
			},
			"table_identifier": schema.SingleNestedAttribute{
				Required:            true,
				MarkdownDescription: "Target table identifier.",
				Attributes: map[string]schema.Attribute{
					"schema_name": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Schema name (defaults to 'public').",
					},
					"table_name": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "Table name.",
					},
				},
			},
		},
	}
}

func (r *connectorS3Resource) newCreateRequest(plan *connectorS3ResourceModel, connectorID string) tsClient.CreateS3ConnectorRequest {
	createReq := tsClient.CreateS3ConnectorRequest{
		ID:        connectorID,
		ProjectID: r.client.GetProjectID(),
		ServiceID: plan.ServiceID.ValueString(),
	}

	if !plan.Name.IsNull() {
		createReq.Name = plan.Name.ValueString()
	}
	if !plan.Bucket.IsNull() {
		createReq.Bucket = plan.Bucket.ValueString()
	}
	if !plan.Pattern.IsNull() {
		createReq.Pattern = plan.Pattern.ValueString()
	}
	if plan.Credentials != nil {
		createReq.Credentials = &tsClient.S3ConnectorCredentials{
			Type: plan.Credentials.Type.ValueString(),
		}
		if !plan.Credentials.RoleARN.IsNull() {
			createReq.Credentials.Role = &tsClient.S3ConnectorCredentialsRole{
				ARN: plan.Credentials.RoleARN.ValueString(),
			}
		}
	}
	if plan.Definition != nil {
		createReq.Definition = r.buildDefinition(plan.Definition)
	}
	if plan.TableIdentifier != nil {
		createReq.TableIdentifier = &tsClient.S3ConnectorTableID{
			TableName:  plan.TableIdentifier.TableName.ValueString(),
			SchemaName: plan.TableIdentifier.SchemaName.ValueString(),
		}
	}
	return createReq
}

// Create creates a new S3 connector.
func (r *connectorS3Resource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	tflog.Trace(ctx, "connectorS3Resource.Create")

	var plan connectorS3ResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate configuration
	if err := r.validateConfig(&plan); err != nil {
		resp.Diagnostics.AddError("Invalid Configuration", err.Error())
		return
	}

	projectID := r.client.GetProjectID()
	serviceID := plan.ServiceID.ValueString()
	connectorID := uuid.New().String()

	// Step 1: Create the connector with core configuration
	// Note: The API's create operation doesn't support all fields (frequency,
	// enabled, on_conflict_do_nothing). These will be configured via an update
	// operation in Step 2
	createReq := r.newCreateRequest(&plan, connectorID)

	if err := r.client.CreateS3Connector(ctx, createReq); err != nil {
		resp.Diagnostics.AddError("Unable to create S3 Connector", err.Error())
		return
	}

	// Step 2: Apply all field configurations via update
	// The create API only supports core fields, so we use update to configure
	// everything including frequency, enabled, on_conflict_do_nothing, and
	// re-applying other fields.
	updateRequests := r.buildUpdateRequests(&plan)

	var connector *tsClient.S3Connector
	var err error

	if len(updateRequests) > 0 {
		connector, err = r.client.UpdateS3Connector(
			ctx,
			connectorID,
			projectID,
			serviceID,
			updateRequests,
		)
		if err != nil {
			resp.Diagnostics.AddError("Unable to configure S3 Connector", err.Error())
			// Cleanup: delete the partially configured connector
			deleteErr := r.client.DeleteS3Connector(ctx, connectorID, projectID, serviceID)
			if deleteErr != nil {
				resp.Diagnostics.AddError(
					"S3 Connector was created but failed to be configured. Cleanup action failed. Connector exists in inconsistent state, needs to be manually deleted",
					deleteErr.Error())
			}
			return
		}
	} else {
		// No updates needed - fetch the created connector state
		connector, err = r.client.GetS3Connector(ctx, connectorID, projectID, serviceID)
		if err != nil {
			resp.Diagnostics.AddError("Unable to fetch created S3 Connector", err.Error())
			return
		}
	}

	// Map response to model
	r.mapConnectorToModel(connector, &plan)

	// Set state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *connectorS3Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "connectorS3Resource.Read")

	var state connectorS3ResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	projectID := r.client.GetProjectID()
	serviceID := state.ServiceID.ValueString()
	connector, err := r.client.GetS3Connector(ctx, id, projectID, serviceID)
	if err != nil {
		resp.Diagnostics.AddError("Error Getting S3 Connector", err.Error())
		return
	}

	r.mapConnectorToModel(connector, &state)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates an existing S3 connector.
func (r *connectorS3Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "connectorS3Resource.Update")

	var plan, state connectorS3ResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate configuration
	if err := r.validateConfig(&plan); err != nil {
		resp.Diagnostics.AddError("Invalid Configuration", err.Error())
		return
	}
	isEnabled := state.Enabled.ValueBool()
	id := state.ID.ValueString()
	projectID := r.client.GetProjectID()
	serviceID := plan.ServiceID.ValueString()

	// If the connector is enabled and an update is going to be executed then we
	// pause the connector.
	if isEnabled {
		_, err := r.disableConnector(ctx, id, projectID, serviceID)
		if err != nil {
			resp.Diagnostics.AddError("Error Disabling S3 Connector for Update", err.Error())
			return
		}
	}

	updateRequests := r.buildUpdateRequests(&plan)
	connector, err := r.client.UpdateS3Connector(ctx, id, projectID, serviceID, updateRequests)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating S3 Connector", err.Error())
		return
	}

	r.mapConnectorToModel(connector, &plan)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes an S3 connector.
func (r *connectorS3Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "connectorS3Resource.Delete")

	var state connectorS3ResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	projectID := r.client.GetProjectID()
	serviceID := state.ServiceID.ValueString()
	err := r.client.DeleteS3Connector(ctx, id, projectID, serviceID)
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting S3 Connector", err.Error())
	}
}

func (r *connectorS3Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, ":")

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: service_id:connector_id. Got: %q", req.ID),
		)
		return
	}

	serviceID := idParts[0]
	connectorID := idParts[1]

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service_id"), serviceID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), connectorID)...)
}

// Configure adds the provider configured client to the resource.
func (r *connectorS3Resource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	tflog.Trace(ctx, "connectorS3Resource.Configure")
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

// Helper functions

func (r *connectorS3Resource) validateConfig(model *connectorS3ResourceModel) error {
	// Validate credentials
	if model.Credentials != nil {
		credType := model.Credentials.Type.ValueString()
		if credType != "Public" && credType != "RoleARN" {
			return fmt.Errorf("credentials type must be 'Public' or 'RoleARN'")
		}
		if credType == "RoleARN" && model.Credentials.RoleARN.IsNull() {
			return fmt.Errorf("role_arn is required when credentials type is 'RoleARN'")
		}
	}

	// Validate definition
	if model.Definition != nil {
		defType := model.Definition.Type.ValueString()
		if defType != "CSV" && defType != "PARQUET" {
			return fmt.Errorf("definition type must be 'CSV' or 'PARQUET'")
		}

		if defType == "CSV" && model.Definition.CSV != nil {
			csv := model.Definition.CSV
			configCount := 0
			if len(csv.ColumnNames) > 0 {
				configCount++
			}
			if len(csv.ColumnMappings) > 0 {
				configCount++
			}
			if csv.AutoColumnMapping.ValueBool() {
				configCount++
			}
			if configCount > 1 {
				return fmt.Errorf("only one of column_names, column_mappings, or auto_column_mapping can be specified")
			}

			// column_mappings and auto_column_mapping require skip_header
			if (len(csv.ColumnMappings) > 0 || csv.AutoColumnMapping.ValueBool()) && !csv.SkipHeader.ValueBool() {
				return fmt.Errorf("skip_header must be true when using column_mappings or auto_column_mapping")
			}
		}

		if defType == "PARQUET" && model.Definition.Parquet != nil {
			parquet := model.Definition.Parquet
			if len(parquet.ColumnMappings) > 0 && parquet.AutoColumnMapping.ValueBool() {
				return fmt.Errorf("only one of column_mappings or auto_column_mapping can be specified for Parquet")
			}
		}
	}

	return nil
}

func (r *connectorS3Resource) buildUpdateRequests(model *connectorS3ResourceModel) []tsClient.S3ConnectorUpdateRequest {
	var requests []tsClient.S3ConnectorUpdateRequest

	// Bucket
	if !model.Bucket.IsNull() {
		requests = append(requests, tsClient.S3ConnectorUpdateRequest{
			Type:   tsClient.S3ConnectorUpdateTypeBucket,
			Bucket: map[string]string{"value": model.Bucket.ValueString()},
		})
	}

	// Pattern
	if !model.Pattern.IsNull() {
		requests = append(requests, tsClient.S3ConnectorUpdateRequest{
			Type:    tsClient.S3ConnectorUpdateTypePattern,
			Pattern: map[string]string{"value": model.Pattern.ValueString()},
		})
	}

	// Credentials
	if model.Credentials != nil {
		creds := &tsClient.S3ConnectorCredentials{
			Type: model.Credentials.Type.ValueString(),
		}
		if !model.Credentials.RoleARN.IsNull() {
			creds.Role = &tsClient.S3ConnectorCredentialsRole{
				ARN: model.Credentials.RoleARN.ValueString(),
			}
		}
		requests = append(requests, tsClient.S3ConnectorUpdateRequest{
			Type: tsClient.S3ConnectorUpdateTypeCredentials,
			Credentials: map[string]interface{}{
				"value": map[string]interface{}{
					"type": creds.Type,
					"role": func() interface{} {
						if creds.Role != nil {
							return map[string]interface{}{"arn": creds.Role.ARN}
						}
						return nil
					}(),
				},
			},
		})
	}

	// Definition
	if model.Definition != nil {
		def := r.buildDefinition(model.Definition)
		requests = append(requests, tsClient.S3ConnectorUpdateRequest{
			Type:       tsClient.S3ConnectorUpdateTypeDefinition,
			Definition: map[string]interface{}{"value": buildDefinitionInput(def)},
		})
	}

	// Table Identifier
	if model.TableIdentifier != nil {
		requests = append(requests, tsClient.S3ConnectorUpdateRequest{
			Type: tsClient.S3ConnectorUpdateTypeTableIdentifier,
			TableIdentifier: map[string]interface{}{
				"value": map[string]interface{}{
					"table_name":  model.TableIdentifier.TableName.ValueString(),
					"schema_name": model.TableIdentifier.SchemaName.ValueString(),
				},
			},
		})
	}

	// Frequency
	if !model.Frequency.IsNull() {
		requests = append(requests, tsClient.S3ConnectorUpdateRequest{
			Type:      tsClient.S3ConnectorUpdateTypeFrequency,
			Frequency: map[string]string{"value": model.Frequency.ValueString()},
		})
	}

	// Name
	if !model.Name.IsNull() {
		requests = append(requests, tsClient.S3ConnectorUpdateRequest{
			Type: tsClient.S3ConnectorUpdateTypeName,
			Name: map[string]string{"value": model.Name.ValueString()},
		})
	}

	// OnConflictDoNothing (sent as settings to API)
	if !model.OnConflictDoNothing.IsNull() {
		requests = append(requests, tsClient.S3ConnectorUpdateRequest{
			Type: tsClient.S3ConnectorUpdateTypeSettings,
			Settings: map[string]interface{}{
				"value": map[string]interface{}{
					"on_conflict_do_nothing": model.OnConflictDoNothing.ValueBool(),
				},
			},
		})
	}

	// HERE BE DRAGON
	// Connectors can't be updated if they are enabled. When sending an update
	// request with multiple items, enabled needs to be the last update applied.
	// Otherwise an error that the connector needs to be paused first will be
	// thrown.
	if !model.Enabled.IsNull() {
		requests = append(requests, tsClient.S3ConnectorUpdateRequest{
			Type:    tsClient.S3ConnectorUpdateTypeEnabled,
			Enabled: map[string]bool{"value": model.Enabled.ValueBool()},
		})
	}

	return requests
}

// disableConnector sets enabled to false. The connector needs to be disabled
// in order to be updated. This functions disables the connector to allow
// updates in the connector config.
func (r *connectorS3Resource) disableConnector(
	ctx context.Context,
	connectorID string,
	projectID string,
	serviceID string,
) (*tsClient.S3Connector, error) {
	enableRequest := []tsClient.S3ConnectorUpdateRequest{
		{
			Type:    tsClient.S3ConnectorUpdateTypeEnabled,
			Enabled: map[string]bool{"value": false},
		},
	}
	return r.client.UpdateS3Connector(ctx, connectorID, projectID, serviceID, enableRequest)
}

func (r *connectorS3Resource) buildDefinition(model *definitionModel) *tsClient.S3ConnectorDefinition {
	def := &tsClient.S3ConnectorDefinition{
		Type: model.Type.ValueString(),
	}

	if model.CSV != nil {
		csvDef := &tsClient.S3ConnectorDefinitionCSV{
			SkipHeader:        model.CSV.SkipHeader.ValueBool(),
			AutoColumnMapping: model.CSV.AutoColumnMapping.ValueBool(),
		}
		if !model.CSV.Delimiter.IsNull() {
			csvDef.Delimiter = model.CSV.Delimiter.ValueString()
		}
		if len(model.CSV.ColumnNames) > 0 {
			csvDef.ColumnNames = make([]string, len(model.CSV.ColumnNames))
			for i, col := range model.CSV.ColumnNames {
				csvDef.ColumnNames[i] = col.ValueString()
			}
		}
		if len(model.CSV.ColumnMappings) > 0 {
			csvDef.ColumnMappings = make([]tsClient.ColumnMapping, len(model.CSV.ColumnMappings))
			for i, m := range model.CSV.ColumnMappings {
				csvDef.ColumnMappings[i] = tsClient.ColumnMapping{
					Source:      m.Source.ValueString(),
					Destination: m.Destination.ValueString(),
				}
			}
		}
		def.CSV = csvDef
	}

	if model.Parquet != nil {
		parquetDef := &tsClient.S3ConnectorDefinitionParquet{
			AutoColumnMapping: model.Parquet.AutoColumnMapping.ValueBool(),
		}
		if len(model.Parquet.ColumnMappings) > 0 {
			parquetDef.ColumnMappings = make([]tsClient.ColumnMapping, len(model.Parquet.ColumnMappings))
			for i, m := range model.Parquet.ColumnMappings {
				parquetDef.ColumnMappings[i] = tsClient.ColumnMapping{
					Source:      m.Source.ValueString(),
					Destination: m.Destination.ValueString(),
				}
			}
		}
		def.Parquet = parquetDef
	}

	return def
}

// buildDefinitionInput is a helper to build GraphQL input (reuse from client).
func buildDefinitionInput(def *tsClient.S3ConnectorDefinition) map[string]interface{} {
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
			mappings := make([]map[string]interface{}, len(def.CSV.ColumnMappings))
			for i, m := range def.CSV.ColumnMappings {
				mappings[i] = map[string]interface{}{
					"source":      m.Source,
					"destination": m.Destination,
				}
			}
			csvInput["column_mappings"] = mappings
		}
		input["csv"] = csvInput
	}

	if def.Parquet != nil {
		parquetInput := map[string]interface{}{
			"auto_column_mapping": def.Parquet.AutoColumnMapping,
		}
		if len(def.Parquet.ColumnMappings) > 0 {
			mappings := make([]map[string]interface{}, len(def.Parquet.ColumnMappings))
			for i, m := range def.Parquet.ColumnMappings {
				mappings[i] = map[string]interface{}{
					"source":      m.Source,
					"destination": m.Destination,
				}
			}
			parquetInput["column_mappings"] = mappings
		}
		input["parquet"] = parquetInput
	}

	return input
}

func (r *connectorS3Resource) mapConnectorToModel(connector *tsClient.S3Connector, model *connectorS3ResourceModel) {
	model.ID = types.StringValue(connector.ID)
	model.ServiceID = types.StringValue(connector.ServiceID)
	model.CreatedAt = types.StringValue(connector.CreatedAt)
	model.UpdatedAt = types.StringValue(connector.UpdatedAt)
	model.Bucket = types.StringValue(connector.Bucket)
	model.Pattern = types.StringValue(connector.Pattern)
	model.Enabled = types.BoolValue(connector.Enabled)

	// Optional field - only set if provided
	if connector.Name != "" {
		model.Name = types.StringValue(connector.Name)
	}

	// Computed fields with defaults - set from API or use empty string
	if connector.Frequency != "" {
		model.Frequency = types.StringValue(connector.Frequency)
	} else {
		model.Frequency = types.StringValue("")
	}

	// Credentials
	if connector.Credentials != nil {
		if model.Credentials == nil {
			model.Credentials = &credentialsModel{}
		}
		model.Credentials.Type = types.StringValue(connector.Credentials.Type)
		if connector.Credentials.Role != nil {
			model.Credentials.RoleARN = types.StringValue(connector.Credentials.Role.ARN)
		}
	}

	// Definition
	if connector.Definition != nil {
		if model.Definition == nil {
			model.Definition = &definitionModel{}
		}
		model.Definition.Type = types.StringValue(connector.Definition.Type)

		if connector.Definition.CSV != nil {
			if model.Definition.CSV == nil {
				model.Definition.CSV = &csvDefinitionModel{}
			}
			model.Definition.CSV.SkipHeader = types.BoolValue(connector.Definition.CSV.SkipHeader)
			model.Definition.CSV.AutoColumnMapping = types.BoolValue(connector.Definition.CSV.AutoColumnMapping)
			if connector.Definition.CSV.Delimiter != "" {
				model.Definition.CSV.Delimiter = types.StringValue(connector.Definition.CSV.Delimiter)
			}
			if len(connector.Definition.CSV.ColumnNames) > 0 {
				model.Definition.CSV.ColumnNames = make([]types.String, len(connector.Definition.CSV.ColumnNames))
				for i, name := range connector.Definition.CSV.ColumnNames {
					model.Definition.CSV.ColumnNames[i] = types.StringValue(name)
				}
			}
			if len(connector.Definition.CSV.ColumnMappings) > 0 {
				model.Definition.CSV.ColumnMappings = make([]columnMappingModel, len(connector.Definition.CSV.ColumnMappings))
				for i, m := range connector.Definition.CSV.ColumnMappings {
					model.Definition.CSV.ColumnMappings[i] = columnMappingModel{
						Source:      types.StringValue(m.Source),
						Destination: types.StringValue(m.Destination),
					}
				}
			}
		}

		if connector.Definition.Parquet != nil {
			if model.Definition.Parquet == nil {
				model.Definition.Parquet = &parquetDefinitionModel{}
			}
			model.Definition.Parquet.AutoColumnMapping = types.BoolValue(connector.Definition.Parquet.AutoColumnMapping)
			if len(connector.Definition.Parquet.ColumnMappings) > 0 {
				model.Definition.Parquet.ColumnMappings = make([]columnMappingModel, len(connector.Definition.Parquet.ColumnMappings))
				for i, m := range connector.Definition.Parquet.ColumnMappings {
					model.Definition.Parquet.ColumnMappings[i] = columnMappingModel{
						Source:      types.StringValue(m.Source),
						Destination: types.StringValue(m.Destination),
					}
				}
			}
		}
	}

	// Table Identifier
	if connector.TableIdentifier != nil {
		if model.TableIdentifier == nil {
			model.TableIdentifier = &tableIdentifierModel{}
		}
		model.TableIdentifier.TableName = types.StringValue(connector.TableIdentifier.TableName)
		if connector.TableIdentifier.SchemaName != "" {
			model.TableIdentifier.SchemaName = types.StringValue(connector.TableIdentifier.SchemaName)
		}
	}

	// OnConflictDoNothing - set from API or default
	if connector.Settings != nil {
		model.OnConflictDoNothing = types.BoolValue(connector.Settings.OnConflictDoNothing)
	} else {
		model.OnConflictDoNothing = types.BoolValue(false)
	}
}
