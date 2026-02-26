package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// useStateUnlessToggleChangesString returns a plan modifier that preserves the
// prior state value (like UseStateForUnknown) unless any of the attributes at
// togglePaths have changed, in which case the value is left as unknown so the
// provider can populate it after apply.
func useStateUnlessToggleChangesString(togglePaths ...string) planmodifier.String {
	return &stringTogglePlanModifier{togglePaths: togglePaths}
}

func useStateUnlessToggleChangesInt64(togglePaths ...string) planmodifier.Int64 {
	return &int64TogglePlanModifier{togglePaths: togglePaths}
}

type stringTogglePlanModifier struct {
	togglePaths []string
}

func (m *stringTogglePlanModifier) Description(_ context.Context) string {
	return "Use state value unless " + strings.Join(m.togglePaths, " or ") + " changes"
}

func (m *stringTogglePlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m *stringTogglePlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if !req.PlanValue.IsUnknown() {
		return
	}

	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}

	if anyToggleChanged(ctx, req.State, req.Plan, m.togglePaths) {
		resp.PlanValue = types.StringUnknown()
		return
	}

	resp.PlanValue = req.StateValue
}

type int64TogglePlanModifier struct {
	togglePaths []string
}

func (m *int64TogglePlanModifier) Description(_ context.Context) string {
	return "Use state value unless " + strings.Join(m.togglePaths, " or ") + " changes"
}

func (m *int64TogglePlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m *int64TogglePlanModifier) PlanModifyInt64(ctx context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response) {
	if !req.PlanValue.IsUnknown() {
		return
	}

	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}

	if anyToggleChanged(ctx, req.State, req.Plan, m.togglePaths) {
		resp.PlanValue = types.Int64Unknown()
		return
	}

	resp.PlanValue = req.StateValue
}

func anyToggleChanged(ctx context.Context, state tfsdk.State, plan tfsdk.Plan, togglePaths []string) bool {
	for _, togglePath := range togglePaths {
		if toggleChanged(ctx, state, plan, togglePath) {
			return true
		}
	}
	return false
}

func toggleChanged(ctx context.Context, state tfsdk.State, plan tfsdk.Plan, togglePath string) bool {
	var stateVal, planVal attr.Value
	diags := make(diag.Diagnostics, 0)
	diags.Append(state.GetAttribute(ctx, path.Root(togglePath), &stateVal)...)
	diags.Append(plan.GetAttribute(ctx, path.Root(togglePath), &planVal)...)
	if diags.HasError() {
		return true
	}
	return !stateVal.Equal(planVal)
}
