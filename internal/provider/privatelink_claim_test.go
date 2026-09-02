package provider

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestNormalizeClaimIdentifier(t *testing.T) {
	const guid = "a32b6330-57c5-4397-8a5d-739775aed7a7"

	for _, tc := range []struct {
		name string
		in   string
		want string
	}{{
		name: "bare azure guid is unchanged",
		in:   guid,
		want: guid,
	}, {
		name: "guid extracted from the approval message",
		in:   "Connected. Claim this connection in the console using ID " + guid,
		want: guid,
	}, {
		name: "aws vpc endpoint id passes through",
		in:   "vpce-0123456789abcdef0",
		want: "vpce-0123456789abcdef0",
	}, {
		name: "surrounding whitespace is trimmed, not treated as prose",
		in:   "  " + guid + "\n",
		want: guid,
	}, {
		name: "an identifier without a guid survives a prose-shaped value",
		in:   "some message with no identifier",
		want: "some message with no identifier",
	}, {
		name: "empty stays empty so the caller can report it",
		in:   "   ",
		want: "",
	}, {
		name: "uppercase guid is matched",
		in:   "Claim using ID A32B6330-57C5-4397-8A5D-739775AED7A7",
		want: "A32B6330-57C5-4397-8A5D-739775AED7A7",
	}} {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeClaimIdentifier(tc.in); got != tc.want {
				t.Errorf("normalizeClaimIdentifier(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// The messages below are the ones savannah-private-link's ClaimConnection
// actually returns; see internal/services/public.go.
func TestClaimRetryable(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{{
		name: "not synced yet is retryable",
		err:  errors.New("connection not found; verify the identifier and that the connection has been established"),
		want: true,
	}, {
		name: "transaction failure is retryable",
		err:  errors.New("failed to begin transaction: context deadline exceeded"),
		want: true,
	}, {
		name: "commit failure is retryable",
		err:  errors.New("failed to commit transaction: connection reset"),
		want: true,
	}, {
		name: "lookup failure is retryable",
		err:  errors.New("failed to get connection: driver: bad connection"),
		want: true,
	}, {
		name: "claimed by another project is terminal",
		err:  errors.New("connection is already claimed"),
		want: false,
	}, {
		name: "rejected connection is terminal",
		err:  errors.New("connection is in state Rejected and cannot be claimed"),
		want: false,
	}, {
		name: "removed connection is terminal",
		err:  errors.New("connection is in state Removed and cannot be claimed"),
		want: false,
	}, {
		name: "missing claim identifier is terminal",
		err:  errors.New("claim_identifier is required"),
		want: false,
	}, {
		name: "missing project id is terminal",
		err:  errors.New("project_id is required"),
		want: false,
	}, {
		name: "nil is not retryable",
		err:  nil,
		want: false,
	}} {
		t.Run(tc.name, func(t *testing.T) {
			if got := claimRetryable(tc.err); got != tc.want {
				t.Errorf("claimRetryable(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

// A terminal failure must explain itself: the backend's own message does not
// say what to do about it.
func TestClaimFailureDetail(t *testing.T) {
	for _, tc := range []struct {
		name        string
		err         error
		wantPhrases []string
	}{{
		name:        "already claimed names the conflict",
		err:         errors.New("connection is already claimed"),
		wantPhrases: []string{"already claimed by another project", "exactly one project"},
	}, {
		name:        "rejected explains it is terminal",
		err:         errors.New("connection is in state Rejected and cannot be claimed"),
		wantPhrases: []string{"cannot be claimed", "Recreate the private endpoint"},
	}, {
		name:        "not found suggests checking the identifier",
		err:         errors.New("connection not found; verify the identifier"),
		wantPhrases: []string{"vpce-", "resourceGuid", "timeout"},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			got := claimFailureDetail("test-id", tc.err)
			for _, phrase := range tc.wantPhrases {
				if !strings.Contains(got, phrase) {
					t.Errorf("detail for %v is missing %q:\n%s", tc.err, phrase, got)
				}
			}
		})
	}
}

// An import recovers the bare identifier while the configuration may hold the
// Azure approval message containing it. Both name the same connection, so the
// plan must not replace the resource — comparing the raw strings would.
func TestRequiresReplaceOnDifferentClaim(t *testing.T) {
	const guid = "a32b6330-57c5-4397-8a5d-739775aed7a7"
	message := "Connected. Claim this connection in the console using ID " + guid

	for _, tc := range []struct {
		name        string
		state, plan string
		wantReplace bool
	}{{
		name:        "imported bare guid against a configured message",
		state:       guid,
		plan:        message,
		wantReplace: false,
	}, {
		name:        "message against the same bare guid",
		state:       message,
		plan:        guid,
		wantReplace: false,
	}, {
		name:        "identical values",
		state:       guid,
		plan:        guid,
		wantReplace: false,
	}, {
		name:        "a genuinely different connection replaces",
		state:       guid,
		plan:        "bbbbbbbb-1111-2222-3333-444455556666",
		wantReplace: true,
	}, {
		name:        "a different aws endpoint replaces",
		state:       "vpce-1111111111111111",
		plan:        "vpce-2222222222222222",
		wantReplace: true,
	}} {
		t.Run(tc.name, func(t *testing.T) {
			req := planmodifier.StringRequest{
				StateValue: types.StringValue(tc.state),
				PlanValue:  types.StringValue(tc.plan),
			}
			resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
			requiresReplaceOnDifferentClaim{}.PlanModifyString(context.Background(), req, resp)
			if resp.RequiresReplace != tc.wantReplace {
				t.Errorf("RequiresReplace = %v, want %v (state=%q plan=%q)",
					resp.RequiresReplace, tc.wantReplace, tc.state, tc.plan)
			}
		})
	}
}

// Create and destroy must not be mistaken for a change of identifier.
func TestRequiresReplaceOnDifferentClaim_nullValues(t *testing.T) {
	for _, tc := range []struct {
		name  string
		state types.String
		plan  types.String
	}{
		{"create: no prior state", types.StringNull(), types.StringValue("vpce-1")},
		{"destroy: no planned value", types.StringValue("vpce-1"), types.StringNull()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := planmodifier.StringRequest{StateValue: tc.state, PlanValue: tc.plan}
			resp := &planmodifier.StringResponse{PlanValue: tc.plan}
			requiresReplaceOnDifferentClaim{}.PlanModifyString(context.Background(), req, resp)
			if resp.RequiresReplace {
				t.Error("RequiresReplace should be false when either value is null")
			}
		})
	}
}
