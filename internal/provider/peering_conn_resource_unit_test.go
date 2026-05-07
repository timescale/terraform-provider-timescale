package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/stretchr/testify/require"
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
