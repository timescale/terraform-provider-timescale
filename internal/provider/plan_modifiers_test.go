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
	hostnameAttr, ok := s.Attributes["hostname"].(schema.StringAttribute)
	if !ok {
		t.Fatal("hostname attribute is not a StringAttribute")
	}

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
	portAttr, ok := s.Attributes["port"].(schema.Int64Attribute)
	if !ok {
		t.Fatal("port attribute is not an Int64Attribute")
	}

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
	hostnameAttr, ok := s.Attributes["hostname"].(schema.StringAttribute)
	if !ok {
		t.Fatal("hostname attribute is not a StringAttribute")
	}

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

// TestServiceSchema_StringEndpointsRefreshOnToggleChange verifies that the
// replica/pooler hostname attributes are left unknown when any attribute that
// moves them changes: vpc_id moves every endpoint of the service (attaching or
// detaching a VPC rewrites the replica and pooler DNS names too), ha_replicas
// moves the replica endpoint, and connection_pooler_enabled moves the pooler
// endpoint.
func TestServiceSchema_StringEndpointsRefreshOnToggleChange(t *testing.T) {
	s := getServiceSchema(t)

	cases := []struct {
		attr        string
		toggle      string
		stateToggle tftypes.Value
		planToggle  tftypes.Value
	}{
		{"replica_hostname", "vpc_id", tftypes.NewValue(tftypes.Number, 100), tftypes.NewValue(tftypes.Number, 200)},
		{"pooler_hostname", "vpc_id", tftypes.NewValue(tftypes.Number, 100), tftypes.NewValue(tftypes.Number, 200)},
		{"replica_hostname", "ha_replicas", tftypes.NewValue(tftypes.Number, 0), tftypes.NewValue(tftypes.Number, 1)},
		{"pooler_hostname", "connection_pooler_enabled", tftypes.NewValue(tftypes.Bool, false), tftypes.NewValue(tftypes.Bool, true)},
	}

	for _, tc := range cases {
		t.Run(tc.attr+" on "+tc.toggle, func(t *testing.T) {
			strAttr, ok := s.Attributes[tc.attr].(schema.StringAttribute)
			if !ok {
				t.Fatalf("%s attribute is not a StringAttribute", tc.attr)
			}

			stateRaw := buildTFValues(t, s, map[string]tftypes.Value{
				tc.attr:   tftypes.NewValue(tftypes.String, "old-host.example.com"),
				tc.toggle: tc.stateToggle,
			})
			planRaw := buildTFValues(t, s, map[string]tftypes.Value{
				tc.attr:   tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
				tc.toggle: tc.planToggle,
			})

			req := planmodifier.StringRequest{
				PlanValue:  types.StringUnknown(),
				StateValue: types.StringValue("old-host.example.com"),
				State:      tfsdk.State{Schema: s, Raw: stateRaw},
				Plan:       tfsdk.Plan{Schema: s, Raw: planRaw},
			}
			resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}

			for _, mod := range strAttr.PlanModifiers {
				mod.PlanModifyString(context.Background(), req, resp)
			}

			if !resp.PlanValue.IsUnknown() {
				t.Errorf("%s should be unknown when %s changes, got %q — the plan modifier is not %s-aware", tc.attr, tc.toggle, resp.PlanValue.ValueString(), tc.toggle)
			}
		})
	}
}

// TestServiceSchema_Int64EndpointsRefreshOnToggleChange is the same matrix for
// the replica/pooler port attributes.
func TestServiceSchema_Int64EndpointsRefreshOnToggleChange(t *testing.T) {
	s := getServiceSchema(t)

	cases := []struct {
		attr        string
		toggle      string
		stateToggle tftypes.Value
		planToggle  tftypes.Value
	}{
		{"replica_port", "vpc_id", tftypes.NewValue(tftypes.Number, 100), tftypes.NewValue(tftypes.Number, 200)},
		{"pooler_port", "vpc_id", tftypes.NewValue(tftypes.Number, 100), tftypes.NewValue(tftypes.Number, 200)},
		{"replica_port", "ha_replicas", tftypes.NewValue(tftypes.Number, 0), tftypes.NewValue(tftypes.Number, 1)},
		{"pooler_port", "connection_pooler_enabled", tftypes.NewValue(tftypes.Bool, false), tftypes.NewValue(tftypes.Bool, true)},
	}

	for _, tc := range cases {
		t.Run(tc.attr+" on "+tc.toggle, func(t *testing.T) {
			intAttr, ok := s.Attributes[tc.attr].(schema.Int64Attribute)
			if !ok {
				t.Fatalf("%s attribute is not an Int64Attribute", tc.attr)
			}

			stateRaw := buildTFValues(t, s, map[string]tftypes.Value{
				tc.attr:   tftypes.NewValue(tftypes.Number, 5432),
				tc.toggle: tc.stateToggle,
			})
			planRaw := buildTFValues(t, s, map[string]tftypes.Value{
				tc.attr:   tftypes.NewValue(tftypes.Number, tftypes.UnknownValue),
				tc.toggle: tc.planToggle,
			})

			req := planmodifier.Int64Request{
				PlanValue:  types.Int64Unknown(),
				StateValue: types.Int64Value(5432),
				State:      tfsdk.State{Schema: s, Raw: stateRaw},
				Plan:       tfsdk.Plan{Schema: s, Raw: planRaw},
			}
			resp := &planmodifier.Int64Response{PlanValue: req.PlanValue}

			for _, mod := range intAttr.PlanModifiers {
				mod.PlanModifyInt64(context.Background(), req, resp)
			}

			if !resp.PlanValue.IsUnknown() {
				t.Errorf("%s should be unknown when %s changes, got %d — the plan modifier is not %s-aware", tc.attr, tc.toggle, resp.PlanValue.ValueInt64(), tc.toggle)
			}
		})
	}
}

// TestServiceSchema_EndpointsPreservedWhenTogglesUnchanged verifies the
// replica/pooler endpoints keep the state value when neither vpc_id nor their
// feature toggle changes.
func TestServiceSchema_EndpointsPreservedWhenTogglesUnchanged(t *testing.T) {
	s := getServiceSchema(t)

	t.Run("replica_hostname", func(t *testing.T) {
		strAttr, ok := s.Attributes["replica_hostname"].(schema.StringAttribute)
		if !ok {
			t.Fatal("replica_hostname attribute is not a StringAttribute")
		}

		unchanged := map[string]tftypes.Value{
			"replica_hostname": tftypes.NewValue(tftypes.String, "repl.example.com"),
			"ha_replicas":      tftypes.NewValue(tftypes.Number, 1),
			"vpc_id":           tftypes.NewValue(tftypes.Number, 100),
		}
		stateRaw := buildTFValues(t, s, unchanged)
		planRaw := buildTFValues(t, s, map[string]tftypes.Value{
			"replica_hostname": tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
			"ha_replicas":      tftypes.NewValue(tftypes.Number, 1),
			"vpc_id":           tftypes.NewValue(tftypes.Number, 100),
		})

		req := planmodifier.StringRequest{
			PlanValue:  types.StringUnknown(),
			StateValue: types.StringValue("repl.example.com"),
			State:      tfsdk.State{Schema: s, Raw: stateRaw},
			Plan:       tfsdk.Plan{Schema: s, Raw: planRaw},
		}
		resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}

		for _, mod := range strAttr.PlanModifiers {
			mod.PlanModifyString(context.Background(), req, resp)
		}

		if resp.PlanValue.ValueString() != "repl.example.com" {
			t.Errorf("replica_hostname should be preserved when toggles are unchanged, got %s", resp.PlanValue)
		}
	})

	t.Run("pooler_port", func(t *testing.T) {
		intAttr, ok := s.Attributes["pooler_port"].(schema.Int64Attribute)
		if !ok {
			t.Fatal("pooler_port attribute is not an Int64Attribute")
		}

		stateRaw := buildTFValues(t, s, map[string]tftypes.Value{
			"pooler_port":               tftypes.NewValue(tftypes.Number, 6432),
			"connection_pooler_enabled": tftypes.NewValue(tftypes.Bool, true),
			"vpc_id":                    tftypes.NewValue(tftypes.Number, 100),
		})
		planRaw := buildTFValues(t, s, map[string]tftypes.Value{
			"pooler_port":               tftypes.NewValue(tftypes.Number, tftypes.UnknownValue),
			"connection_pooler_enabled": tftypes.NewValue(tftypes.Bool, true),
			"vpc_id":                    tftypes.NewValue(tftypes.Number, 100),
		})

		req := planmodifier.Int64Request{
			PlanValue:  types.Int64Unknown(),
			StateValue: types.Int64Value(6432),
			State:      tfsdk.State{Schema: s, Raw: stateRaw},
			Plan:       tfsdk.Plan{Schema: s, Raw: planRaw},
		}
		resp := &planmodifier.Int64Response{PlanValue: req.PlanValue}

		for _, mod := range intAttr.PlanModifiers {
			mod.PlanModifyInt64(context.Background(), req, resp)
		}

		if resp.PlanValue.ValueInt64() != 6432 {
			t.Errorf("pooler_port should be preserved when toggles are unchanged, got %s", resp.PlanValue)
		}
	})
}
