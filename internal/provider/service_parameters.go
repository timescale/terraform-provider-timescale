package provider

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	tsClient "github.com/timescale/terraform-provider-timescale/internal/client"
)

const (
	// privateKeyImportParameters marks the Read that follows an import so it can
	// populate postgres_parameters from the service instead of from prior state.
	privateKeyImportParameters = "import_postgres_parameters"
	errPostgresParameters      = "Error applying postgres_parameters"
	postgresParametersAttr     = "postgres_parameters"
)

// Poll settings are variables so tests can shorten them.
var (
	parameterPollInterval = 10 * time.Second
	parameterPollTimeout  = 5 * time.Minute
)

// knownParameterMap returns the known, non-null elements of a string map attribute.
func knownParameterMap(m types.Map) map[string]string {
	out := map[string]string{}
	if m.IsNull() || m.IsUnknown() {
		return out
	}
	for k, v := range m.Elements() {
		s, ok := v.(types.String)
		if !ok || s.IsNull() || s.IsUnknown() {
			continue
		}
		out[k] = s.ValueString()
	}
	return out
}

// parameterMapValue converts a Go map to the attribute value. nil becomes null.
func parameterMapValue(ctx context.Context, in map[string]string) (types.Map, diag.Diagnostics) {
	if in == nil {
		return types.MapNull(types.StringType), nil
	}
	return types.MapValueFrom(ctx, types.StringType, in)
}

func removedParameterKeys(state, plan map[string]string) []string {
	var removed []string
	for k := range state {
		if _, ok := plan[k]; !ok {
			removed = append(removed, k)
		}
	}
	slices.Sort(removed)
	return removed
}

// changedParameterKeys returns keys that are new in plan or whose value differs from state.
func changedParameterKeys(state, plan map[string]string) []string {
	var changed []string
	for k, v := range plan {
		if old, ok := state[k]; !ok || old != v {
			changed = append(changed, k)
		}
	}
	slices.Sort(changed)
	return changed
}

func restartRequiredKeys(catalog map[string]parameterCatalogEntry, keys []string) []string {
	var out []string
	for _, k := range keys {
		if entry, ok := catalog[k]; ok && entry.info.RequiresRestart {
			out = append(out, k)
		}
	}
	return out
}

// parametersSettled reports whether every desired value is live and not pending a restart.
func parametersSettled(catalog map[string]parameterCatalogEntry, desired map[string]string) bool {
	for name, raw := range desired {
		entry, ok := catalog[name]
		if !ok || entry.info.IsPendingRestart || !parameterValueEqual(entry, raw) {
			return false
		}
	}
	return true
}

func (r *serviceResource) fetchParameterCatalog(ctx context.Context, serviceID string) (map[string]parameterCatalogEntry, error) {
	p, err := r.client.GetPostgresParameters(ctx, serviceID)
	if err != nil {
		return nil, err
	}
	return buildParameterCatalog(p), nil
}

// applyPostgresParameters sets desired values on a READY service and waits for them to be live.
func (r *serviceResource) applyPostgresParameters(ctx context.Context, serviceID string, desired map[string]string) diag.Diagnostics {
	var diags diag.Diagnostics
	if len(desired) == 0 {
		return diags
	}
	catalog, err := r.fetchParameterCatalog(ctx, serviceID)
	if err != nil {
		diags.AddError(errPostgresParameters, fmt.Sprintf("unable to read current parameters: %s", err))
		return diags
	}

	var numeric []tsClient.SetNumericParameter
	var str []tsClient.SetStringParameter
	for _, name := range slices.Sorted(maps.Keys(desired)) {
		raw := desired[name]
		attrPath := path.Root(postgresParametersAttr).AtMapKey(name)
		entry, ok := catalog[name]
		if !ok {
			diags.AddAttributeError(attrPath, errPostgresParameters, fmt.Sprintf("unknown parameter %q", name))
			continue
		}
		if !entry.info.IsUserEditable {
			diags.AddAttributeError(attrPath, errPostgresParameters,
				fmt.Sprintf("parameter %q is not user-editable on this service. On read replicas, session-level parameters such as work_mem cannot be changed.", name))
			continue
		}
		parsed, err := parseParameterValue(entry, raw)
		if err != nil {
			diags.AddAttributeError(attrPath, errPostgresParameters, err.Error())
			continue
		}
		if !entry.info.IsPendingRestart && parameterValueEqual(entry, raw) {
			continue
		}
		if parsed.numeric {
			numeric = append(numeric, tsClient.SetNumericParameter{Name: name, Value: parsed.value, Unit: parsed.unit})
		} else {
			str = append(str, tsClient.SetStringParameter{Name: name, Value: parsed.str})
		}
	}
	if diags.HasError() || len(numeric)+len(str) == 0 {
		return diags
	}

	if err := r.client.SetPostgresParameters(ctx, serviceID, numeric, str); err != nil {
		var perrs tsClient.ParameterErrors
		if errors.As(err, &perrs) {
			for _, pe := range perrs {
				diags.AddAttributeError(path.Root(postgresParametersAttr).AtMapKey(pe.Name), errPostgresParameters, pe.ErrorMessage)
			}
			return diags
		}
		diags.AddError(errPostgresParameters, err.Error())
		return diags
	}

	if err := r.waitForPostgresParameters(ctx, serviceID, desired); err != nil {
		diags.AddWarning("Postgres parameters pending",
			fmt.Sprintf("The parameters were accepted but are not all live yet: %s. Parameters that require a restart apply when the restart completes. Run terraform refresh later to verify.", err))
	}
	return diags
}

func (r *serviceResource) waitForPostgresParameters(ctx context.Context, serviceID string, desired map[string]string) error {
	deadline := time.Now().Add(parameterPollTimeout)
	for {
		catalog, err := r.fetchParameterCatalog(ctx, serviceID)
		if err != nil {
			// The service is unreadable while it restarts. Keep polling.
			tflog.Debug(ctx, "polling postgres parameters", map[string]any{"service_id": serviceID, "error": err.Error()})
		} else if parametersSettled(catalog, desired) {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out after %s waiting for parameters to become live", parameterPollTimeout)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(parameterPollInterval):
		}
	}
}

// readPostgresParameters refreshes the managed keys. Only keys already in prior state are
// tracked, except right after an import, where every user-modified parameter is adopted.
func (r *serviceResource) readPostgresParameters(ctx context.Context, service *tsClient.Service, prior types.Map, imported bool) (types.Map, diag.Diagnostics) {
	var diags diag.Diagnostics
	priorMap := knownParameterMap(prior)
	if !imported && (prior.IsNull() || prior.IsUnknown() || len(priorMap) == 0) {
		return prior, diags
	}
	// Parameters cannot be read unless the service is running; keep what we have.
	if service.Status != "READY" {
		if imported {
			diags.AddWarning("Postgres parameters not imported",
				fmt.Sprintf("Service %s is %s. Parameters can only be read from a READY service.", service.ID, service.Status))
			return types.MapNull(types.StringType), diags
		}
		return prior, diags
	}

	catalog, err := r.fetchParameterCatalog(ctx, service.ID)
	if err != nil {
		diags.AddError("Error reading postgres_parameters", err.Error())
		return prior, diags
	}

	if imported {
		var adopted map[string]string
		for name, entry := range catalog {
			if entry.info.IsUserModified {
				if adopted == nil {
					adopted = map[string]string{}
				}
				adopted[name] = formatParameterValue(entry)
			}
		}
		return parameterMapValue(ctx, adopted)
	}

	refreshed := make(map[string]string, len(priorMap))
	var dropped []string
	for name, raw := range priorMap {
		entry, ok := catalog[name]
		switch {
		case !ok:
			dropped = append(dropped, name)
		case entry.info.IsPendingRestart, parameterValueEqual(entry, raw):
			refreshed[name] = raw
		default:
			refreshed[name] = formatParameterValue(entry)
		}
	}
	if len(dropped) > 0 {
		slices.Sort(dropped)
		tflog.Warn(ctx, "postgres parameters no longer exist on the service and were removed from state",
			map[string]any{"service_id": service.ID, "parameters": strings.Join(dropped, ", ")})
	}
	return parameterMapValue(ctx, refreshed)
}
