package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// useStateUnlessToggleChangesString returns a plan modifier that preserves the
// prior state value (like UseStateForUnknown) unless the attribute at togglePath
// has changed, in which case the value is left as unknown so the provider can
// populate it after apply.
func useStateUnlessToggleChangesString(togglePath string) planmodifier.String {
	return &stringTogglePlanModifier{togglePath: togglePath}
}

func useStateUnlessToggleChangesInt64(togglePath string) planmodifier.Int64 {
	return &int64TogglePlanModifier{togglePath: togglePath}
}

type stringTogglePlanModifier struct {
	togglePath string
}

func (m *stringTogglePlanModifier) Description(_ context.Context) string {
	return "Use state value unless " + m.togglePath + " changes"
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

	if toggleChanged(ctx, req.State, req.Plan, m.togglePath) {
		resp.PlanValue = types.StringUnknown()
		return
	}

	resp.PlanValue = req.StateValue
}

type int64TogglePlanModifier struct {
	togglePath string
}

func (m *int64TogglePlanModifier) Description(_ context.Context) string {
	return "Use state value unless " + m.togglePath + " changes"
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

	if toggleChanged(ctx, req.State, req.Plan, m.togglePath) {
		resp.PlanValue = types.Int64Unknown()
		return
	}

	resp.PlanValue = req.StateValue
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
