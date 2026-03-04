package provider

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func init() {
	for _, env := range []string{"PEER_ACCOUNT_ID", "PEER_VPC_ID", "PEER_TGW_ID", "PEER_REGION"} {
		if os.Getenv(env) == "" {
			os.Setenv(env, "unit-test-placeholder")
		}
	}
}

func getServiceSchema(t *testing.T) schema.Schema {
	t.Helper()
	r := NewServiceResource()
	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("failed to get schema: %v", resp.Diagnostics)
	}
	return resp.Schema
}

// buildTFValues builds a tftypes object matching the service schema with
// the given overrides. All other attributes default to null.
func buildTFValues(t *testing.T, s schema.Schema, values map[string]tftypes.Value) tftypes.Value {
	t.Helper()
	attrTypes := map[string]tftypes.Type{}
	attrValues := map[string]tftypes.Value{}

	for name, attr := range s.Attributes {
		tfType := attr.GetType().TerraformType(context.Background())
		attrTypes[name] = tfType
		attrValues[name] = tftypes.NewValue(tfType, nil) // null default
	}
	for k, v := range values {
		attrValues[k] = v
	}

	return tftypes.NewValue(tftypes.Object{AttributeTypes: attrTypes}, attrValues)
}

// TestServiceSchema_HostnameRefreshesOnVpcChange verifies that the hostname
// plan modifier on the actual service resource schema leaves hostname as
// unknown when vpc_id changes. This test fails if hostname uses
// UseStateForUnknown instead of useStateUnlessToggleChangesString("vpc_id").
func TestServiceSchema_HostnameRefreshesOnVpcChange(t *testing.T) {
	s := getServiceSchema(t)
	hostnameAttr := s.Attributes["hostname"].(schema.StringAttribute)

	stateRaw := buildTFValues(t, s, map[string]tftypes.Value{
		"hostname": tftypes.NewValue(tftypes.String, "old-host.example.com"),
		"vpc_id":   tftypes.NewValue(tftypes.Number, 100),
	})
	planRaw := buildTFValues(t, s, map[string]tftypes.Value{
		"hostname": tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"vpc_id":   tftypes.NewValue(tftypes.Number, 200),
	})

	state := tfsdk.State{Schema: s, Raw: stateRaw}
	plan := tfsdk.Plan{Schema: s, Raw: planRaw}

	req := planmodifier.StringRequest{
		PlanValue:  types.StringUnknown(),
		StateValue: types.StringValue("old-host.example.com"),
		State:      state,
		Plan:       plan,
	}
	resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}

	for _, mod := range hostnameAttr.PlanModifiers {
		mod.PlanModifyString(context.Background(), req, resp)
	}

	if !resp.PlanValue.IsUnknown() {
		t.Errorf("hostname should be unknown when vpc_id changes, got %q — the plan modifier is not vpc_id-aware", resp.PlanValue.ValueString())
	}
}

// TestServiceSchema_PortRefreshesOnVpcChange is the same test for the port attribute.
func TestServiceSchema_PortRefreshesOnVpcChange(t *testing.T) {
	s := getServiceSchema(t)
	portAttr := s.Attributes["port"].(schema.Int64Attribute)

	stateRaw := buildTFValues(t, s, map[string]tftypes.Value{
		"port":   tftypes.NewValue(tftypes.Number, 5432),
		"vpc_id": tftypes.NewValue(tftypes.Number, 100),
	})
	planRaw := buildTFValues(t, s, map[string]tftypes.Value{
		"port":   tftypes.NewValue(tftypes.Number, tftypes.UnknownValue),
		"vpc_id": tftypes.NewValue(tftypes.Number, 200),
	})

	state := tfsdk.State{Schema: s, Raw: stateRaw}
	plan := tfsdk.Plan{Schema: s, Raw: planRaw}

	req := planmodifier.Int64Request{
		PlanValue:  types.Int64Unknown(),
		StateValue: types.Int64Value(5432),
		State:      state,
		Plan:       plan,
	}
	resp := &planmodifier.Int64Response{PlanValue: req.PlanValue}

	for _, mod := range portAttr.PlanModifiers {
		mod.PlanModifyInt64(context.Background(), req, resp)
	}

	if !resp.PlanValue.IsUnknown() {
		t.Errorf("port should be unknown when vpc_id changes, got %d — the plan modifier is not vpc_id-aware", resp.PlanValue.ValueInt64())
	}
}

// TestServiceSchema_HostnamePreservedWhenVpcUnchanged verifies hostname is
// preserved from state when vpc_id does not change.
func TestServiceSchema_HostnamePreservedWhenVpcUnchanged(t *testing.T) {
	s := getServiceSchema(t)
	hostnameAttr := s.Attributes["hostname"].(schema.StringAttribute)

	stateRaw := buildTFValues(t, s, map[string]tftypes.Value{
		"hostname": tftypes.NewValue(tftypes.String, "my-host.example.com"),
		"vpc_id":   tftypes.NewValue(tftypes.Number, 100),
	})
	planRaw := buildTFValues(t, s, map[string]tftypes.Value{
		"hostname": tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"vpc_id":   tftypes.NewValue(tftypes.Number, 100),
	})

	state := tfsdk.State{Schema: s, Raw: stateRaw}
	plan := tfsdk.Plan{Schema: s, Raw: planRaw}

	req := planmodifier.StringRequest{
		PlanValue:  types.StringUnknown(),
		StateValue: types.StringValue("my-host.example.com"),
		State:      state,
		Plan:       plan,
	}
	resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}

	for _, mod := range hostnameAttr.PlanModifiers {
		mod.PlanModifyString(context.Background(), req, resp)
	}

	if resp.PlanValue.ValueString() != "my-host.example.com" {
		t.Errorf("hostname should be preserved when vpc_id unchanged, got %s", resp.PlanValue)
	}
}
