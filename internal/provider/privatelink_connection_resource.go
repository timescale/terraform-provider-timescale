package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
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

// defaultClaimTimeout bounds the wait for a connection to appear. The backend
// syncs from the cloud providers every 5 minutes, and this resource also asks
// for a sync on every attempt, so the wait is normally much shorter.
const defaultClaimTimeout = 10 * time.Minute

func NewPrivateLinkConnectionResource() resource.Resource {
	return &privateLinkConnectionResource{}
}

type privateLinkConnectionResource struct {
	client *tsClient.Client
}

type privateLinkConnectionResourceModel struct {
	ID              types.String   `tfsdk:"id"`
	ClaimIdentifier types.String   `tfsdk:"claim_identifier"`
	Name            types.String   `tfsdk:"name"`
	RejectOnDestroy types.Bool     `tfsdk:"reject_on_destroy"`
	Timeouts        timeouts.Value `tfsdk:"timeouts"`
	// Computed fields
	ConnectionID         types.String `tfsdk:"connection_id"`
	CloudProvider        types.String `tfsdk:"cloud_provider"`
	Region               types.String `tfsdk:"region"`
	State                types.String `tfsdk:"state"`
	LinkIdentifier       types.String `tfsdk:"link_identifier"`
	ProviderConnectionID types.String `tfsdk:"provider_connection_id"`
	PrincipalID          types.String `tfsdk:"principal_id"`
}

func (r *privateLinkConnectionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_privatelink_connection"
}

func (r *privateLinkConnectionResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Claims a Private Link connection for this project. Import using the connection ID: `terraform import timescale_privatelink_connection.example <connection_id>`.",
		MarkdownDescription: `Claims a Private Link connection for this project.

Tiger Cloud's Private Link endpoint service is open: anyone can create a private
endpoint against it, and every connection is accepted but left unowned. Ownership
is established by claiming the connection with an identifier you read from your
own cloud resource. Authentication to the project is the proof of ownership —
nothing about the cloud account is trusted, which is what makes a shared
third-party provider account work.

This resource does not create the cloud-side endpoint. You create that with your
cloud provider (` + "`aws_vpc_endpoint`" + `, ` + "`azurerm_private_endpoint`" + `),
then claim it here.

## Claim identifier

| Cloud | ` + "`claim_identifier`" + ` | Where to read it |
| ----- | ------------------------- | ---------------- |
| AWS   | VPC endpoint ID (` + "`vpce-…`" + `) | ` + "`aws_vpc_endpoint.example.id`" + ` |
| Azure | Private endpoint's ` + "`resourceGuid`" + ` | ` + "`azurerm`" + ` does not expose it, so pass ` + "`azurerm_private_endpoint_connection`" + `'s ` + "`private_service_connection[0].request_response`" + ` — the approval message Tiger Cloud writes back. The GUID is extracted from it automatically. |

Passing the approval message directly means no extra provider is needed. A bare
` + "`resourceGuid`" + ` works too, whether read with the ` + "`azapi`" + ` provider or
copied from the Azure portal.

## Workflow

1. Look up the endpoint service name with ` + "`timescale_privatelink_region`" + `.
2. Create a private endpoint or VPC endpoint against that service name.
3. Claim it with this resource.
4. Attach services with ` + "`timescale_service.private_endpoint_connection_ids`" + `.
5. Create a private DNS zone in your own VPC or VNet pointing the service
   hostname at your endpoint. Tiger Cloud does not publish these records.

## Connecting over Private Link

The hostname is the same one the public endpoint uses, so the existing
certificate validates with ` + "`sslmode=verify-full`" + `. Only the resolution
and the port differ, and the port is allocated per binding — read it from the
Tiger Cloud console or the API rather than assuming 5432.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Resource identifier (same as connection_id).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"claim_identifier": schema.StringAttribute{
				Required: true,
				Description: "Identifier read from your own cloud resource. AWS: the VPC endpoint ID (vpce-...). " +
					"Azure: the private endpoint's resourceGuid, or the whole connection approval message " +
					"containing it — the GUID is extracted for you, so you can pass " +
					"azurerm_private_endpoint_connection's request_response directly. Changing this claims a " +
					"different connection, so the resource is replaced, which is what happens when the " +
					"underlying endpoint is recreated. Replacement releases the old claim; whether it also " +
					"rejects the old connection is governed by reject_on_destroy.",
				PlanModifiers: []planmodifier.String{
					requiresReplaceOnDifferentClaim{},
				},
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Optional display name for the connection.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"reject_on_destroy": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
				Description: "Whether destroying this resource also rejects the connection in your cloud account. " +
					"Defaults to false, so a destroy releases the claim from Terraform state and leaves the " +
					"connection in place — re-applying claims it again. Setting this to true makes destroy " +
					"irreversible: a rejected connection cannot be re-claimed, and you must recreate the " +
					"private endpoint to reconnect.",
			},
			"connection_id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier for this connection. Use this for timescale_service.private_endpoint_connection_ids.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cloud_provider": schema.StringAttribute{
				Computed:    true,
				Description: "The cloud provider the connection originates from, as recorded when it was synced.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"region": schema.StringAttribute{
				Computed:    true,
				Description: "The Tiger Cloud region of the endpoint service this connection reaches.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"state": schema.StringAttribute{
				Computed:    true,
				Description: "The state of the connection (approved, pending, rejected, removed).",
			},
			"link_identifier": schema.StringAttribute{
				Computed: true,
				Description: "The cloud provider's own identifier for the connection. This is not the claim " +
					"identifier: on Azure it is a provider-side value the customer cannot read.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"provider_connection_id": schema.StringAttribute{
				Computed: true,
				Description: "Provider-specific connection identifier: the VPC endpoint ID on AWS, " +
					"\"<endpoint name>.<resourceGuid>\" on Azure.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"principal_id": schema.StringAttribute{
				Computed:    true,
				Description: "The cloud account the connection originates from. Informational only — it carries no ownership meaning.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true,
			}),
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
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := plan.Timeouts.Create(ctx, defaultClaimTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The configured value may be the Azure approval message rather than a bare
	// identifier. Normalize for the API call only: state keeps what the practitioner
	// wrote, so the next plan compares like with like and reports no drift.
	configured := plan.ClaimIdentifier.ValueString()
	claimIdentifier := normalizeClaimIdentifier(configured)
	if claimIdentifier == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("claim_identifier"),
			"claim_identifier is empty",
			"No identifier could be read from the configured value. If this is an Azure approval message "+
				"from azurerm_private_endpoint_connection, it is not populated until Tiger Cloud approves "+
				"the connection — wait for the next sync and apply again.",
		)
		return
	}
	if claimIdentifier != configured {
		tflog.Info(ctx, "Extracted claim identifier from the configured value", map[string]interface{}{
			"claim_identifier": claimIdentifier,
		})
	}

	// The connection only exists in the backend once a sync has picked it up
	// from the cloud provider, so ask for one on every attempt rather than
	// waiting out the periodic tick. Claiming a connection this project already
	// owns succeeds, which is what makes a repeated apply safe.
	var conn *tsClient.PrivateLinkConnection
	err := retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		if syncErr := r.client.SyncPrivateLinkConnections(ctx); syncErr != nil {
			tflog.Warn(ctx, "Failed to sync Private Link connections", map[string]interface{}{"error": syncErr.Error()})
		}

		claimed, claimErr := r.client.ClaimPrivateLinkConnection(ctx, claimIdentifier)
		if claimErr == nil {
			conn = claimed
			return nil
		}

		if claimRetryable(claimErr) {
			tflog.Info(ctx, "Connection not claimable yet, retrying...", map[string]interface{}{
				"claim_identifier": claimIdentifier,
				"error":            claimErr.Error(),
			})
			return retry.RetryableError(claimErr)
		}
		return retry.NonRetryableError(claimErr)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to claim Private Link connection",
			claimFailureDetail(claimIdentifier, err),
		)
		return
	}

	// Claim does not take a name, so set it separately when one is configured.
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		name := plan.Name.ValueString()
		updated, err := r.client.UpdatePrivateLinkConnection(ctx, conn.ConnectionID, &name)
		if err != nil {
			resp.Diagnostics.AddError("Failed to set Private Link connection name", err.Error())
			return
		}
		conn = updated
	}

	setConnectionState(&plan, conn)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *privateLinkConnectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state privateLinkConnectionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Only connections owned by this project are listable, so an empty result
	// means the claim is gone rather than that the endpoint disappeared.
	connections, err := r.client.ListPrivateLinkConnections(ctx, "")
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Private Link connections", err.Error())
		return
	}

	connectionID := state.ConnectionID.ValueString()
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

	// An import supplies only the connection ID; recover the claim identifier
	// from the provider-side value the backend derives it from.
	if state.ClaimIdentifier.IsNull() || state.ClaimIdentifier.ValueString() == "" {
		state.ClaimIdentifier = types.StringValue(claimIdentifierFor(conn))
	}

	setConnectionState(&state, conn)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *privateLinkConnectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state privateLinkConnectionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	desiredName := plan.Name

	// Everything the backend owns is unchanged by an update, so carry it over
	// from state. Only name is remotely mutable; reject_on_destroy is local.
	plan.ID = state.ID
	plan.ConnectionID = state.ConnectionID
	plan.CloudProvider = state.CloudProvider
	plan.Region = state.Region
	plan.State = state.State
	plan.LinkIdentifier = state.LinkIdentifier
	plan.ProviderConnectionID = state.ProviderConnectionID
	plan.PrincipalID = state.PrincipalID
	plan.Name = state.Name

	if !desiredName.Equal(state.Name) && !desiredName.IsUnknown() {
		name := desiredName.ValueString()
		conn, err := r.client.UpdatePrivateLinkConnection(ctx, state.ConnectionID.ValueString(), &name)
		if err != nil {
			resp.Diagnostics.AddError("Failed to update Private Link connection", err.Error())
			return
		}
		setConnectionState(&plan, conn)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *privateLinkConnectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state privateLinkConnectionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Rejecting is terminal cloud-side, so by default a destroy only drops the
	// resource from state. The claim survives and re-applying reclaims it; if
	// the customer removes their endpoint, sync marks the connection removed.
	if !state.RejectOnDestroy.ValueBool() {
		tflog.Info(ctx, "Releasing Private Link connection from state without rejecting it", map[string]interface{}{
			"connection_id": state.ConnectionID.ValueString(),
		})
		return
	}

	connectionID := state.ConnectionID.ValueString()
	tflog.Info(ctx, "Rejecting Private Link connection", map[string]interface{}{
		"connection_id": connectionID,
	})

	// Bindings are removed asynchronously when a service detaches, and the
	// backend refuses to reject a connection that still has them.
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
		resp.Diagnostics.AddError("Failed to reject Private Link connection", err.Error())
	}
}

func (r *privateLinkConnectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if strings.TrimSpace(req.ID) == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			"Expected a connection ID. Got an empty string.",
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("connection_id"), req.ID)...)
}

func setConnectionState(model *privateLinkConnectionResourceModel, conn *tsClient.PrivateLinkConnection) {
	model.ID = types.StringValue(conn.ConnectionID)
	model.ConnectionID = types.StringValue(conn.ConnectionID)
	model.CloudProvider = types.StringValue(conn.CloudProvider)
	model.Region = types.StringValue(conn.Region)
	model.State = types.StringValue(conn.State)
	model.LinkIdentifier = types.StringValue(conn.LinkIdentifier)
	model.ProviderConnectionID = types.StringValue(conn.ProviderConnectionID)
	model.PrincipalID = types.StringValue(conn.PrincipalID)
	model.Name = types.StringValue(conn.Name)
}

// isConnectionNotFound reports whether the claim failed because the connection
// has not been synced from the cloud provider yet, which is worth retrying.
// Every other failure — an identifier claimed by another project, a rejected
// connection — is permanent.
// claimRetryable reports whether a failed claim is worth retrying.
//
// The backend distinguishes two retryable conditions from the permanent ones:
// a connection that has not been synced from the cloud provider yet, and
// transient database failures. Everything else — an identifier already claimed
// by another project, a rejected or removed connection, a missing argument —
// will fail identically no matter how long we wait.
func claimRetryable(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())

	// Permanent, and checked first: a 500-class message could otherwise mask
	// one of these if the backend ever wraps them.
	for _, terminal := range []string{
		"already claimed",   // owned by a different project
		"cannot be claimed", // state is rejected or removed
		"is required",       // empty claim_identifier or project_id
	} {
		if strings.Contains(msg, terminal) {
			return false
		}
	}

	// Not yet visible to the backend: sync has not picked it up.
	if strings.Contains(msg, "connection not found") {
		return true
	}

	// Transient server-side failures: transaction begin/commit and lookup
	// errors are all reported as "failed to ...".
	return strings.Contains(msg, "failed to ")
}

// requiresReplaceOnDifferentClaim replaces the resource only when the
// configured value names a *different* connection.
//
// A plain RequiresReplace compares the raw strings, which would replace the
// resource whenever the text changes but the identifier does not — most
// visibly after an import, which recovers the bare identifier while the
// configuration holds the Azure approval message that contains it.
type requiresReplaceOnDifferentClaim struct{}

func (requiresReplaceOnDifferentClaim) Description(_ context.Context) string {
	return "Replace the resource when the claim identifier names a different connection"
}

func (m requiresReplaceOnDifferentClaim) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (requiresReplaceOnDifferentClaim) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// Nothing to compare on create or destroy.
	if req.StateValue.IsNull() || req.PlanValue.IsNull() {
		return
	}
	if normalizeClaimIdentifier(req.StateValue.ValueString()) != normalizeClaimIdentifier(req.PlanValue.ValueString()) {
		resp.RequiresReplace = true
	}
}

// claimFailureDetail explains a failed claim in terms of what the practitioner
// can do about it, since the backend's own message is terse and the same call
// fails for several unrelated reasons.
func claimFailureDetail(claimIdentifier string, err error) string {
	msg := strings.ToLower(err.Error())

	switch {
	case strings.Contains(msg, "already claimed"):
		return fmt.Sprintf("Identifier %q is already claimed by another project.\n\n"+
			"A connection belongs to exactly one project. Claiming a connection this project "+
			"already owns succeeds, so this means a different project holds it. Verify the "+
			"identifier, or detach it from the other project first.\n\nBackend error: %s",
			claimIdentifier, err)

	case strings.Contains(msg, "cannot be claimed"):
		return fmt.Sprintf("Connection %q is in a state that cannot be claimed.\n\n"+
			"A rejected or removed connection is terminal: Tiger Cloud's sync skips it, so it "+
			"never returns. Recreate the private endpoint in your cloud account to get a new "+
			"connection.\n\nBackend error: %s", claimIdentifier, err)

	case strings.Contains(msg, "connection not found"):
		return fmt.Sprintf("No connection matching %q was established before the timeout.\n\n"+
			"Check that the endpoint exists in your cloud account and that claim_identifier "+
			"matches it: the VPC endpoint ID (vpce-...) on AWS, the private endpoint's "+
			"resourceGuid on Azure. Raise the create timeout if your endpoint is slow to "+
			"appear.\n\nBackend error: %s", claimIdentifier, err)

	default:
		return fmt.Sprintf("Claiming %q failed: %s", claimIdentifier, err)
	}
}

// guidPattern matches a cloud resource GUID anywhere in a string.
var guidPattern = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)

// normalizeClaimIdentifier extracts the identifier the API expects from what a
// practitioner has to hand.
//
// On Azure the claim identifier is the private endpoint's resourceGuid, which
// the azurerm provider does not expose. What it does expose, on the
// azurerm_private_endpoint_connection data source, is the approval message Tiger
// Cloud writes back — which contains the GUID. Accepting that message directly
// spares practitioners a second provider or a regex in their own HCL.
//
// A bare identifier never contains whitespace, so only a value that does is
// treated as prose to mine a GUID from. That keeps identifiers that merely look
// GUID-adjacent, such as an AWS vpce- ID, passing through untouched.
func normalizeClaimIdentifier(v string) string {
	trimmed := strings.TrimSpace(v)
	if strings.ContainsAny(trimmed, " \t\n") {
		if guid := guidPattern.FindString(trimmed); guid != "" {
			return guid
		}
	}
	return trimmed
}

// claimIdentifierFor recovers the claim identifier from a connection, for
// imports where the practitioner supplied only the connection ID. On Azure the
// provider connection ID is "<endpoint name>.<resourceGuid>" and the claim
// identifier is the GUID; endpoint names may contain periods, so split on the
// last one.
func claimIdentifierFor(conn *tsClient.PrivateLinkConnection) string {
	if conn.CloudProvider != "azure" {
		return conn.ProviderConnectionID
	}
	if i := strings.LastIndex(conn.ProviderConnectionID, "."); i >= 0 {
		return conn.ProviderConnectionID[i+1:]
	}
	return conn.ProviderConnectionID
}
