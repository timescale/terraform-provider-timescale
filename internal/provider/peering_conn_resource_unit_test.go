package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/require"

	tsClient "github.com/timescale/terraform-provider-timescale/internal/client"
)

func TestUpgradeString(t *testing.T) {
	require.True(t, upgradeString(nil).IsNull())
	require.True(t, upgradeString(123).IsNull(), "non-string input must yield null")
	got := upgradeString("hello")
	require.False(t, got.IsNull())
	require.Equal(t, "hello", got.ValueString())
}

func TestUpgradeInt64(t *testing.T) {
	require.True(t, upgradeInt64(nil).IsNull())
	require.True(t, upgradeInt64("not-a-number").IsNull())
	require.Equal(t, int64(42), upgradeInt64(float64(42)).ValueInt64())
}

func TestUpgradeStringList(t *testing.T) {
	// nil → typed null list
	got, diags := upgradeStringList(nil)
	require.False(t, diags.HasError())
	require.True(t, got.IsNull())
	require.Equal(t, types.StringType, got.ElementType(context.Background()))

	// empty array → empty typed list (not null)
	got, diags = upgradeStringList([]any{})
	require.False(t, diags.HasError())
	require.False(t, got.IsNull())
	require.Equal(t, 0, len(got.Elements()))
	require.Equal(t, types.StringType, got.ElementType(context.Background()))

	// populated array → typed list with values
	got, diags = upgradeStringList([]any{"10.0.0.0/24", "10.0.1.0/24"})
	require.False(t, diags.HasError())
	require.False(t, got.IsNull())
	require.Equal(t, 2, len(got.Elements()))
	require.Equal(t, types.StringType, got.ElementType(context.Background()))

	// non-array value → typed null list (defensive)
	got, diags = upgradeStringList("not-a-list")
	require.False(t, diags.HasError())
	require.True(t, got.IsNull())
}

// TestPeeringConnectionUpgradeStateV0ToV1 exercises the full upgrader against
// raw v0 state JSON. It covers three cases that matter for issue #309:
//   - peer_cidr_blocks absent (state from before 2.1.0)
//   - peer_cidr_blocks null
//   - peer_cidr_blocks populated
func TestPeeringConnectionUpgradeStateV0ToV1(t *testing.T) {
	ctx := context.Background()
	r := &peeringConnectionResource{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	require.False(t, schemaResp.Diagnostics.HasError())
	require.Equal(t, int64(1), schemaResp.Schema.GetVersion(), "schema version should be 1")

	upgraders := r.UpgradeState(ctx)
	v0, ok := upgraders[0]
	require.True(t, ok, "v0→v1 upgrader must be registered")

	cases := []struct {
		name             string
		json             string
		wantCIDRBlocksIs func(t *testing.T, l types.List)
		wantPeerVPCID    string
		wantTimescaleID  int64
	}{
		{
			name: "field absent (pre-2.1.0 state)",
			json: `{
				"id": 42,
				"timescale_vpc_id": 7,
				"peer_vpc_id": "vpc-aaa",
				"peer_account_id": "111111111111",
				"peer_region_code": "us-east-1",
				"peer_cidr": "10.0.0.0/24",
				"provisioned_id": "pcx-aaa",
				"status": "ACTIVE"
			}`,
			wantCIDRBlocksIs: func(t *testing.T, l types.List) {
				require.True(t, l.IsNull(), "missing field must produce typed null list")
				require.Equal(t, types.StringType, l.ElementType(context.Background()))
			},
			wantPeerVPCID:   "vpc-aaa",
			wantTimescaleID: 7,
		},
		{
			name: "explicit null",
			json: `{
				"id": 1,
				"timescale_vpc_id": 2,
				"peer_vpc_id": "vpc-bbb",
				"peer_account_id": "222222222222",
				"peer_region_code": "us-east-1",
				"peer_cidr_blocks": null
			}`,
			wantCIDRBlocksIs: func(t *testing.T, l types.List) {
				require.True(t, l.IsNull())
				require.Equal(t, types.StringType, l.ElementType(context.Background()))
			},
			wantPeerVPCID:   "vpc-bbb",
			wantTimescaleID: 2,
		},
		{
			name: "populated",
			json: `{
				"id": 9,
				"timescale_vpc_id": 3,
				"peer_vpc_id": "vpc-ccc",
				"peer_account_id": "333333333333",
				"peer_region_code": "us-west-2",
				"peer_cidr_blocks": ["10.0.0.0/24", "10.0.1.0/24"]
			}`,
			wantCIDRBlocksIs: func(t *testing.T, l types.List) {
				require.False(t, l.IsNull())
				require.Equal(t, 2, len(l.Elements()))
				require.Equal(t, types.StringType, l.ElementType(context.Background()))
			},
			wantPeerVPCID:   "vpc-ccc",
			wantTimescaleID: 3,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := resource.UpgradeStateRequest{
				RawState: &tfprotov6.RawState{JSON: []byte(tc.json)},
			}
			resp := &resource.UpgradeStateResponse{
				State: tfsdk.State{Schema: schemaResp.Schema},
			}
			v0.StateUpgrader(ctx, req, resp)
			require.False(t, resp.Diagnostics.HasError(), "diags: %v", resp.Diagnostics)

			var got peeringConnectionResourceModel
			diags := resp.State.Get(ctx, &got)
			require.False(t, diags.HasError(), "diags: %v", diags)

			tc.wantCIDRBlocksIs(t, got.PeerCIDRBlocks)
			require.Equal(t, tc.wantPeerVPCID, got.PeerVPCID.ValueString())
			require.Equal(t, tc.wantTimescaleID, got.TimescaleVPCID.ValueInt64())
		})
	}
}

const (
	vpcBusyBody     = `{"errors":[{"message":"VPC is already being updated, try again later"}]}`
	peeringListed   = `{"id":"5","vpcId":"vpc-timescale","provisionedId":"pcx-1","status":"PENDING","peerVpc":{"id":"vpc-peer","accountId":"111111111111","regionCode":"us-east-1","cidrBlocks":["10.0.0.0/24"]}}`
	vpcNotFoundBody = `{"errors":[{"message":"target VPC does not exist"}]}`
)

func vpcBody(peerings ...string) string {
	return fmt.Sprintf(`{"data":{"getVpc":{"id":"7","provisionedId":"vpc-timescale","status":"CREATED","peeringConnections":[%s]}}}`, strings.Join(peerings, ","))
}

// opMock serves the bodies listed for each GraphQL operationName in order; the last one repeats once exhausted.
type opMock struct {
	t      *testing.T
	mu     sync.Mutex
	bodies map[string][]string
	calls  map[string]int
}

func (m *opMock) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var body struct {
		OperationName string `json:"operationName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		m.t.Errorf("decoding request: %v", err)
		return
	}
	bodies, ok := m.bodies[body.OperationName]
	if !ok {
		m.t.Errorf("unexpected GraphQL operation %q", body.OperationName)
		return
	}
	idx := min(m.calls[body.OperationName], len(bodies)-1)
	m.calls[body.OperationName]++
	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprint(w, bodies[idx])
}

func (m *opMock) callCount(op string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls[op]
}

func newTestPeeringResource(t *testing.T, bodies map[string][]string) (*peeringConnectionResource, *opMock) {
	t.Helper()
	origDelay, origInterval, origTimeout := peeringWaitDelay, peeringPollInterval, peeringWaitTimeout
	peeringWaitDelay, peeringPollInterval, peeringWaitTimeout = 0, 10*time.Millisecond, 5*time.Second
	t.Cleanup(func() {
		peeringWaitDelay, peeringPollInterval, peeringWaitTimeout = origDelay, origInterval, origTimeout
	})

	mock := &opMock{t: t, bodies: bodies, calls: map[string]int{}}
	srv := httptest.NewServer(mock)
	t.Cleanup(srv.Close)
	t.Setenv("TIMESCALE_DEV_URL", srv.URL)
	return &peeringConnectionResource{client: tsClient.NewClient("token", "proj", "test", "1.0.0")}, mock
}

func peeringSchema(t *testing.T) schema.Schema {
	t.Helper()
	resp := &resource.SchemaResponse{}
	(&peeringConnectionResource{}).Schema(context.Background(), resource.SchemaRequest{}, resp)
	require.False(t, resp.Diagnostics.HasError())
	return resp.Schema
}

func emptyPeeringState(t *testing.T) tfsdk.State {
	t.Helper()
	s := peeringSchema(t)
	return tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(context.Background()), nil)}
}

func peeringModel() peeringConnectionResourceModel {
	return peeringConnectionResourceModel{
		ID:                    types.Int64Value(5),
		VpcID:                 types.StringNull(),
		ProvisionedID:         types.StringNull(),
		AccepterProvisionedID: types.StringNull(),
		Status:                types.StringNull(),
		ErrorMessage:          types.StringNull(),
		PeerVPCID:             types.StringValue("vpc-peer"),
		PeerTGWID:             types.StringNull(),
		PeerCIDRBlocks:        types.ListNull(types.StringType),
		PeerCIDR:              types.StringNull(),
		PeerAccountID:         types.StringValue("111111111111"),
		PeerRegionCode:        types.StringValue("us-east-1"),
		TimescaleVPCID:        types.Int64Value(7),
		PeeringType:           types.StringNull(),
	}
}

func TestPeeringConnectionCreate_RetriesWhileVPCBusy(t *testing.T) {
	ctx := context.Background()
	r, mock := newTestPeeringResource(t, map[string][]string{
		"OpenPeerRequest": {vpcBusyBody, `{"data":{"openPeerRequest":{"id":"5"}}}`},
		"GetVPCByID":      {vpcBody(peeringListed)},
	})

	plan := tfsdk.Plan{Schema: peeringSchema(t)}
	model := peeringModel()
	model.ID = types.Int64Unknown()
	require.False(t, plan.Set(ctx, model).HasError())
	resp := &resource.CreateResponse{State: emptyPeeringState(t)}

	r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
	require.False(t, resp.Diagnostics.HasError(), "diags: %v", resp.Diagnostics)
	require.Equal(t, 2, mock.callCount("OpenPeerRequest"))

	var got peeringConnectionResourceModel
	require.False(t, resp.State.Get(ctx, &got).HasError())
	require.Equal(t, int64(5), got.ID.ValueInt64())
	require.Equal(t, "pcx-1", got.ProvisionedID.ValueString())
}

func TestPeeringConnectionDelete_RetriesWhileVPCBusy(t *testing.T) {
	ctx := context.Background()
	r, mock := newTestPeeringResource(t, map[string][]string{
		"DeletePeeringConnection": {vpcBusyBody, `{"data":{"deletePeeringConnection":true}}`},
		"GetVPCByID":              {vpcBody(peeringListed), vpcBody()},
	})

	state := emptyPeeringState(t)
	require.False(t, state.Set(ctx, peeringModel()).HasError())
	resp := &resource.DeleteResponse{State: state}

	r.Delete(ctx, resource.DeleteRequest{State: state}, resp)
	require.False(t, resp.Diagnostics.HasError(), "diags: %v", resp.Diagnostics)
	require.Equal(t, 2, mock.callCount("DeletePeeringConnection"))
	require.Equal(t, 2, mock.callCount("GetVPCByID"), "delete must wait for the peering to disappear")
}

func TestPeeringConnectionDelete_Errors(t *testing.T) {
	cases := []struct {
		name      string
		getVPC    string
		wantError bool
	}{
		{name: "peering already gone", getVPC: vpcBody()},
		{name: "vpc already gone", getVPC: vpcNotFoundBody},
		{name: "peering still listed", getVPC: vpcBody(peeringListed), wantError: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			r, mock := newTestPeeringResource(t, map[string][]string{
				"DeletePeeringConnection": {`{"errors":[{"message":"peering connection not found"}]}`},
				"GetVPCByID":              {tc.getVPC},
			})

			state := emptyPeeringState(t)
			require.False(t, state.Set(ctx, peeringModel()).HasError())
			resp := &resource.DeleteResponse{State: state}

			r.Delete(ctx, resource.DeleteRequest{State: state}, resp)
			require.Equal(t, tc.wantError, resp.Diagnostics.HasError(), "diags: %v", resp.Diagnostics)
			require.Equal(t, 1, mock.callCount("DeletePeeringConnection"), "non-busy errors must not be retried")
		})
	}
}

func TestRetryWhileVPCBusy_GivesUpAtTimeout(t *testing.T) {
	calls := 0
	err := retryWhileVPCBusy(context.Background(), time.Second, func() error {
		calls++
		return &tsClient.Error{Message: "VPC is already being updated, try again later"}
	})
	require.ErrorContains(t, err, "already being updated")
	require.Greater(t, calls, 1)
}

func TestPeeringConnectionCreate_KeepsStateWhenNotReady(t *testing.T) {
	ctx := context.Background()
	failed := `{"id":"5","vpcId":"vpc-timescale","provisionedId":"pcx-1","status":"INVALID","errorMessage":"rejected","peerVpc":{"id":"vpc-peer"}}`
	r, _ := newTestPeeringResource(t, map[string][]string{
		"OpenPeerRequest": {`{"data":{"openPeerRequest":{"id":"5"}}}`},
		"GetVPCByID":      {vpcBody(failed)},
	})

	plan := tfsdk.Plan{Schema: peeringSchema(t)}
	model := peeringModel()
	model.ID = types.Int64Unknown()
	model.Status = types.StringUnknown()
	model.PeerCIDRBlocks = types.ListUnknown(types.StringType)
	require.False(t, plan.Set(ctx, model).HasError())
	resp := &resource.CreateResponse{State: emptyPeeringState(t)}

	r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
	require.True(t, resp.Diagnostics.HasError())

	var got peeringConnectionResourceModel
	require.False(t, resp.State.Get(ctx, &got).HasError())
	require.Equal(t, int64(5), got.ID.ValueInt64(), "an opened peering must stay in state so it can be replaced")
	require.True(t, got.PeerCIDRBlocks.IsNull())
}

func TestPeeringConnectionUpdate_RetriesWhileVPCBusy(t *testing.T) {
	ctx := context.Background()
	r, mock := newTestPeeringResource(t, map[string][]string{
		"UpdatePeeringConnectionCIDRs": {vpcBusyBody, `{"data":{"updatePeeringConnectionCIDRs":true}}`},
	})

	model := peeringModel()
	model.PeerCIDRBlocks = types.ListValueMust(types.StringType, []attr.Value{types.StringValue("10.3.0.0/16")})
	plan := tfsdk.Plan{Schema: peeringSchema(t)}
	require.False(t, plan.Set(ctx, model).HasError())
	resp := &resource.UpdateResponse{State: emptyPeeringState(t)}

	r.Update(ctx, resource.UpdateRequest{Plan: plan}, resp)
	require.False(t, resp.Diagnostics.HasError(), "diags: %v", resp.Diagnostics)
	require.Equal(t, 2, mock.callCount("UpdatePeeringConnectionCIDRs"))
}
