package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	tsClient "github.com/timescale/terraform-provider-timescale/internal/client"
)

var (
	_ resource.Resource              = &privateLinkConnectionResource{}
	_ resource.ResourceWithConfigure = &privateLinkConnectionResource{}
)

func NewPrivateLinkConnectionResource() resource.Resource {
	return &privateLinkConnectionResource{}
}

type privateLinkConnectionResource struct {
	client *tsClient.Client
}

type privateLinkConnectionResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	AzureConnectionName types.String `tfsdk:"azure_connection_name"`
	Region              types.String `tfsdk:"region"`
	IPAddress           types.String `tfsdk:"ip_address"`
	Name                types.String `tfsdk:"name"`
	Timeout             types.String `tfsdk:"timeout"`
	// Computed fields
	ConnectionID   types.String `tfsdk:"connection_id"`
	SubscriptionID types.String `tfsdk:"subscription_id"`
	LinkIdentifier types.String `tfsdk:"link_identifier"`
	State          types.String `tfsdk:"state"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

func (r *privateLinkConnectionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_privatelink_connection"
}

func (r *privateLinkConnectionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Azure Private Link connection in a Timescale project.",
		MarkdownDescription: `Manages an Azure Private Link connection in a Timescale project.

This resource discovers an existing Azure Private Link connection (created via Azure Private Endpoint)
and allows you to configure its IP address and name. The connection must already exist in Azure -
this resource syncs with Azure to find it and then manages the Timescale-side configuration.

## Workflow

1. Create an Azure Private Endpoint pointing to the Timescale Private Link Service
2. Use this resource with ` + "`azure_connection_name`" + ` set to the private endpoint name
3. The resource will sync and wait for the connection to appear
4. Set ` + "`ip_address`" + ` to the private IP from the Azure Private Endpoint
5. The connection can then be used with ` + "`timescale_service.private_endpoint_connection_id`" + `

## Important

The ` + "`azure_connection_name`" + ` filter matches using the Azure Private Endpoint name (not the
private_service_connection name). Azure formats the connection name as ` + "`<pe-name>.<guid>`" + `.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Resource identifier (same as connection_id).",
			},
			"azure_connection_name": schema.StringAttribute{
				Required: true,
				Description: "The Azure private endpoint name to match. " +
					"Azure formats the connection name as '<pe-name>.<guid>', so this matches " +
					"connections where the name starts with this value followed by a dot.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"region": schema.StringAttribute{
				Required:    true,
				Description: "The Timescale region (e.g., az-eastus2).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ip_address": schema.StringAttribute{
				Required: true,
				Description: "The private IP address of the Azure Private Endpoint. " +
					"This is required to enable services to connect via this private link.",
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Optional display name for the connection.",
			},
			"timeout": schema.StringAttribute{
				Optional: true,
				Description: "How long to wait for the connection to appear during create. " +
					"Accepts duration strings like '2m', '5m', '30s'. Defaults to '2m'.",
			},
			"connection_id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier for this connection. Use this for timescale_service.private_endpoint_connection_id.",
			},
			"subscription_id": schema.StringAttribute{
				Computed:    true,
				Description: "The Azure subscription ID.",
			},
			"link_identifier": schema.StringAttribute{
				Computed:    true,
				Description: "The Azure private link identifier.",
			},
			"state": schema.StringAttribute{
				Computed:    true,
				Description: "The state of the connection (e.g., APPROVED, PENDING).",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "When the connection was created.",
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "When the connection was last updated.",
			},
		},
	}
}

func (r *privateLinkConnectionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *privateLinkConnectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan privateLinkConnectionResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse timeout
	timeoutStr := "2m"
	if !plan.Timeout.IsNull() {
		timeoutStr = plan.Timeout.ValueString()
	}
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		resp.Diagnostics.AddError("Invalid timeout", fmt.Sprintf("Failed to parse timeout '%s': %s", timeoutStr, err))
		return
	}

	azureConnectionName := plan.AzureConnectionName.ValueString()
	region := plan.Region.ValueString()

	// Sync and wait for the connection to appear
	var conn *tsClient.PrivateLinkConnection
	deadline := time.Now().Add(timeout)
	retryInterval := 15 * time.Second

	for {
		tflog.Debug(ctx, "Syncing Private Link connections from Azure")
		if err := r.client.SyncPrivateLinkConnections(ctx); err != nil {
			tflog.Warn(ctx, "Failed to sync Private Link connections", map[string]interface{}{"error": err.Error()})
		}

		connections, err := r.client.ListPrivateLinkConnections(ctx, region)
		if err != nil {
			resp.Diagnostics.AddError("Unable to list Private Link connections", err.Error())
			return
		}

		conn = findConnectionByAzureName(connections, azureConnectionName)
		if conn != nil {
			break
		}

		if time.Now().After(deadline) {
			resp.Diagnostics.AddError(
				"Timeout waiting for Private Link connection",
				fmt.Sprintf("No connection matching azure_connection_name '%s' found after %s. "+
					"Ensure the Azure Private Endpoint is created and the authorization is approved.",
					azureConnectionName, timeoutStr),
			)
			return
		}

		tflog.Info(ctx, "Connection not found yet, retrying...", map[string]interface{}{
			"azure_connection_name": azureConnectionName,
			"retry_in":              retryInterval.String(),
		})
		time.Sleep(retryInterval)
	}

	// Update the connection with IP address and name
	ipAddress := plan.IPAddress.ValueString()
	var namePtr *string
	if !plan.Name.IsNull() {
		name := plan.Name.ValueString()
		namePtr = &name
	}

	updatedConn, err := r.client.UpdatePrivateLinkConnection(ctx, conn.ConnectionID, &ipAddress, namePtr)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update Private Link connection", err.Error())
		return
	}

	// Set state
	plan.ID = types.StringValue(updatedConn.ConnectionID)
	plan.ConnectionID = types.StringValue(updatedConn.ConnectionID)
	plan.SubscriptionID = types.StringValue(updatedConn.SubscriptionID)
	plan.LinkIdentifier = types.StringValue(updatedConn.LinkIdentifier)
	plan.State = types.StringValue(updatedConn.State)
	plan.IPAddress = types.StringValue(updatedConn.IPAddress)
	plan.Name = types.StringValue(updatedConn.Name)
	plan.CreatedAt = types.StringValue(updatedConn.CreatedAt)
	plan.UpdatedAt = types.StringValue(updatedConn.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *privateLinkConnectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state privateLinkConnectionResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	region := state.Region.ValueString()
	connectionID := state.ConnectionID.ValueString()

	connections, err := r.client.ListPrivateLinkConnections(ctx, region)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Private Link connections", err.Error())
		return
	}

	var conn *tsClient.PrivateLinkConnection
	for _, c := range connections {
		if c.ConnectionID == connectionID {
			conn = c
			break
		}
	}

	if conn == nil {
		// Connection no longer exists
		resp.State.RemoveResource(ctx)
		return
	}

	// Update state
	state.SubscriptionID = types.StringValue(conn.SubscriptionID)
	state.LinkIdentifier = types.StringValue(conn.LinkIdentifier)
	state.State = types.StringValue(conn.State)
	state.IPAddress = types.StringValue(conn.IPAddress)
	state.Name = types.StringValue(conn.Name)
	state.CreatedAt = types.StringValue(conn.CreatedAt)
	state.UpdatedAt = types.StringValue(conn.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *privateLinkConnectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan privateLinkConnectionResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state privateLinkConnectionResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	connectionID := state.ConnectionID.ValueString()
	ipAddress := plan.IPAddress.ValueString()
	var namePtr *string
	if !plan.Name.IsNull() {
		name := plan.Name.ValueString()
		namePtr = &name
	}

	updatedConn, err := r.client.UpdatePrivateLinkConnection(ctx, connectionID, &ipAddress, namePtr)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update Private Link connection", err.Error())
		return
	}

	// Update state
	plan.ID = types.StringValue(updatedConn.ConnectionID)
	plan.ConnectionID = types.StringValue(updatedConn.ConnectionID)
	plan.SubscriptionID = types.StringValue(updatedConn.SubscriptionID)
	plan.LinkIdentifier = types.StringValue(updatedConn.LinkIdentifier)
	plan.State = types.StringValue(updatedConn.State)
	plan.IPAddress = types.StringValue(updatedConn.IPAddress)
	plan.Name = types.StringValue(updatedConn.Name)
	plan.CreatedAt = types.StringValue(updatedConn.CreatedAt)
	plan.UpdatedAt = types.StringValue(updatedConn.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *privateLinkConnectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state privateLinkConnectionResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	connectionID := state.ConnectionID.ValueString()
	tflog.Info(ctx, "Deleting Private Link connection", map[string]interface{}{
		"connection_id": connectionID,
	})

	// Retry deletion with backoff - bindings may take time to be removed after service deletion
	maxRetries := 10
	retryInterval := 5 * time.Second

	for i := 0; i < maxRetries; i++ {
		err := r.client.DeletePrivateLinkConnection(ctx, connectionID)
		if err == nil {
			return
		}

		// Check if error is due to existing bindings
		if !strings.Contains(err.Error(), "existing bindings") {
			resp.Diagnostics.AddError("Failed to delete Private Link connection", err.Error())
			return
		}

		if i < maxRetries-1 {
			tflog.Info(ctx, "Connection still has bindings, retrying...", map[string]interface{}{
				"connection_id": connectionID,
				"retry":         i + 1,
				"max_retries":   maxRetries,
				"retry_in":      retryInterval.String(),
			})
			time.Sleep(retryInterval)
		}
	}

	resp.Diagnostics.AddError(
		"Failed to delete Private Link connection",
		fmt.Sprintf("Connection %s still has bindings after %d retries. The service may still be detaching.", connectionID, maxRetries),
	)
}

func findConnectionByAzureName(connections []*tsClient.PrivateLinkConnection, filter string) *tsClient.PrivateLinkConnection {
	expectedPrefix := filter + "."
	for _, conn := range connections {
		if strings.HasPrefix(conn.AzureConnectionName, expectedPrefix) {
			return conn
		}
	}
	return nil
}
