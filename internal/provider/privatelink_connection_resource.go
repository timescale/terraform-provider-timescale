package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	tsClient "github.com/timescale/terraform-provider-timescale/internal/client"
)

var (
	_ resource.Resource                = &privateLinkConnectionResource{}
	_ resource.ResourceWithConfigure   = &privateLinkConnectionResource{}
	_ resource.ResourceWithImportState = &privateLinkConnectionResource{}
)

func NewPrivateLinkConnectionResource() resource.Resource {
	return &privateLinkConnectionResource{}
}

type privateLinkConnectionResource struct {
	client *tsClient.Client
}

type privateLinkConnectionResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	ProviderConnectionID types.String `tfsdk:"provider_connection_id"`
	CloudProvider        types.String `tfsdk:"cloud_provider"`
	Region               types.String `tfsdk:"region"`
	IPAddress            types.String `tfsdk:"ip_address"`
	Name                 types.String `tfsdk:"name"`
	Timeout              types.String `tfsdk:"timeout"`
	// Computed fields
	ConnectionID   types.String `tfsdk:"connection_id"`
	LinkIdentifier types.String `tfsdk:"link_identifier"`
	State          types.String `tfsdk:"state"`
}

func (r *privateLinkConnectionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_privatelink_connection"
}

func (r *privateLinkConnectionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Private Link connection in a Timescale project. Import using `region,connection_id` format: `terraform import timescale_privatelink_connection.example <region>,<connection_id>`.",
		MarkdownDescription: `Manages a Private Link connection in a Timescale project.

This resource discovers an existing Private Link connection (created via Azure Private Endpoint
or AWS VPC Endpoint) and allows you to configure its IP address and name.

## Workflow

### Azure
1. Create an Azure Private Endpoint pointing to the Timescale Private Link Service
2. Use this resource with ` + "`provider_connection_id`" + ` set to the private endpoint name and ` + "`cloud_provider = \"azure\"`" + `
3. The resource will sync and wait for the connection to appear
4. Set ` + "`ip_address`" + ` to the private IP from the Azure Private Endpoint

### AWS
1. Create an AWS VPC Endpoint pointing to the Timescale VPC Endpoint Service
2. Use this resource with ` + "`provider_connection_id`" + ` set to the VPC Endpoint ID and ` + "`cloud_provider = \"aws\"`" + `
3. The resource will sync and find the connection`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Resource identifier (same as connection_id).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"provider_connection_id": schema.StringAttribute{
				Required: true,
				Description: "The cloud provider connection identifier. " +
					"For Azure: the private endpoint name. For AWS: the VPC Endpoint ID (vpce-...).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cloud_provider": schema.StringAttribute{
				Required:    true,
				Description: "The cloud provider: azure or aws.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"region": schema.StringAttribute{
				Required:    true,
				Description: "The Timescale region (e.g., az-eastus2, us-east-1).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ip_address": schema.StringAttribute{
				Required: true,
				Description: "The private IP address of the Private Endpoint or VPC Endpoint. " +
					"Required to enable services to connect via this private link.",
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Optional display name for the connection.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"timeout": schema.StringAttribute{
				Optional: true,
				Description: "How long to wait for the connection to appear during create. " +
					"Accepts duration strings like '2m', '5m', '30s'. Defaults to '2m'.",
			},
			"connection_id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier for this connection. Use this for timescale_service.private_endpoint_connection_id.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"link_identifier": schema.StringAttribute{
				Computed:    true,
				Description: "The private link identifier.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"state": schema.StringAttribute{
				Computed:    true,
				Description: "The state of the connection (e.g., approved, pending).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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

	providerConnectionID := plan.ProviderConnectionID.ValueString()
	cloudProvider := plan.CloudProvider.ValueString()
	region := plan.Region.ValueString()

	// Sync and wait for the connection to appear
	var conn *tsClient.PrivateLinkConnection
	err = retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		tflog.Debug(ctx, "Syncing Private Link connections")
		if syncErr := r.client.SyncPrivateLinkConnections(ctx); syncErr != nil {
			tflog.Warn(ctx, "Failed to sync Private Link connections", map[string]interface{}{"error": syncErr.Error()})
		}

		connections, listErr := r.client.ListPrivateLinkConnections(ctx, region)
		if listErr != nil {
			return retry.NonRetryableError(fmt.Errorf("unable to list Private Link connections: %w", listErr))
		}

		switch cloudProvider {
		case "azure":
			conn = findConnectionByAzureName(connections, providerConnectionID)
		case "aws":
			conn = findConnectionByProviderID(connections, providerConnectionID)
		default:
			return retry.NonRetryableError(fmt.Errorf("unsupported cloud_provider: %s", cloudProvider))
		}

		if conn != nil {
			return nil
		}

		tflog.Info(ctx, "Connection not found yet, retrying...", map[string]interface{}{
			"provider_connection_id": providerConnectionID,
			"cloud_provider":         cloudProvider,
		})
		return retry.RetryableError(fmt.Errorf("connection not found"))
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Timeout waiting for Private Link connection",
			fmt.Sprintf("No connection matching provider_connection_id '%s' (cloud_provider=%s) found after %s. "+
				"Ensure the endpoint is created and the authorization is approved.",
				providerConnectionID, cloudProvider, timeoutStr),
		)
		return
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

	setConnectionState(&plan, updatedConn)
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
		resp.State.RemoveResource(ctx)
		return
	}

	// Preserve user-specified provider_connection_id during import
	if state.ProviderConnectionID.IsNull() || state.ProviderConnectionID.ValueString() == "" {
		state.ProviderConnectionID = types.StringValue(conn.ProviderConnectionID)
	}
	if state.CloudProvider.IsNull() || state.CloudProvider.ValueString() == "" {
		state.CloudProvider = types.StringValue(conn.CloudProvider)
	}

	state.ID = types.StringValue(conn.ConnectionID)
	state.ConnectionID = types.StringValue(conn.ConnectionID)
	state.LinkIdentifier = types.StringValue(conn.LinkIdentifier)
	state.State = types.StringValue(conn.State)
	state.IPAddress = types.StringValue(conn.IPAddress)
	state.Name = types.StringValue(conn.Name)

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

	setConnectionState(&plan, updatedConn)
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
	err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		deleteErr := r.client.DeletePrivateLinkConnection(ctx, connectionID)
		if deleteErr == nil {
			return nil
		}

		if strings.Contains(deleteErr.Error(), "existing bindings") {
			tflog.Info(ctx, "Connection still has bindings, retrying...", map[string]interface{}{
				"connection_id": connectionID,
			})
			return retry.RetryableError(deleteErr)
		}

		return retry.NonRetryableError(deleteErr)
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete Private Link connection", err.Error())
	}
}

func (r *privateLinkConnectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, ",")

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: region,connection_id. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("region"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("connection_id"), idParts[1])...)
}

func setConnectionState(model *privateLinkConnectionResourceModel, conn *tsClient.PrivateLinkConnection) {
	model.ID = types.StringValue(conn.ConnectionID)
	model.ConnectionID = types.StringValue(conn.ConnectionID)
	model.LinkIdentifier = types.StringValue(conn.LinkIdentifier)
	model.State = types.StringValue(conn.State)
	model.IPAddress = types.StringValue(conn.IPAddress)
	model.Name = types.StringValue(conn.Name)
}

func findConnectionByAzureName(connections []*tsClient.PrivateLinkConnection, filter string) *tsClient.PrivateLinkConnection {
	expectedPrefix := filter + "."
	for _, conn := range connections {
		if strings.HasPrefix(conn.ProviderConnectionID, expectedPrefix) {
			return conn
		}
	}
	return nil
}

func findConnectionByProviderID(connections []*tsClient.PrivateLinkConnection, providerConnectionID string) *tsClient.PrivateLinkConnection {
	for _, conn := range connections {
		if conn.ProviderConnectionID == providerConnectionID {
			return conn
		}
	}
	return nil
}
