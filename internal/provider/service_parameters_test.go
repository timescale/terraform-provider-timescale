package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"

	tsClient "github.com/timescale/terraform-provider-timescale/internal/client"
)

func TestKnownParameterMap(t *testing.T) {
	require.Empty(t, knownParameterMap(types.MapNull(types.StringType)))
	require.Empty(t, knownParameterMap(types.MapUnknown(types.StringType)))

	m, diags := types.MapValue(types.StringType, map[string]attr.Value{
		"work_mem":        types.StringValue("64MB"),
		"max_connections": types.StringUnknown(),
	})
	require.False(t, diags.HasError())
	require.Equal(t, map[string]string{"work_mem": "64MB"}, knownParameterMap(m))
}

func TestParameterMapValue(t *testing.T) {
	ctx := context.Background()

	null, diags := parameterMapValue(ctx, nil)
	require.False(t, diags.HasError())
	require.True(t, null.IsNull())

	empty, diags := parameterMapValue(ctx, map[string]string{})
	require.False(t, diags.HasError())
	require.False(t, empty.IsNull())
	require.Empty(t, empty.Elements())

	full, diags := parameterMapValue(ctx, map[string]string{"work_mem": "64MB"})
	require.False(t, diags.HasError())
	require.Equal(t, map[string]string{"work_mem": "64MB"}, knownParameterMap(full))
}

func TestRemovedAndChangedParameterKeys(t *testing.T) {
	state := map[string]string{"a": "1", "b": "2", "c": "3"}
	plan := map[string]string{"b": "2", "c": "30", "d": "4"}

	require.Equal(t, []string{"a"}, removedParameterKeys(state, plan))
	require.Equal(t, []string{"c", "d"}, changedParameterKeys(state, plan))
	require.Empty(t, removedParameterKeys(nil, plan))
	require.Equal(t, []string{"b", "c", "d"}, changedParameterKeys(nil, plan))
	require.Empty(t, changedParameterKeys(state, state))
}

func TestRestartRequiredKeys(t *testing.T) {
	catalog := map[string]parameterCatalogEntry{
		"max_connections": {info: tsClient.ParameterInfo{Name: "max_connections", RequiresRestart: true}},
		"work_mem":        {info: tsClient.ParameterInfo{Name: "work_mem"}},
	}
	got := restartRequiredKeys(catalog, []string{"work_mem", "max_connections", "unknown"})
	require.Equal(t, []string{"max_connections"}, got)
}

func TestParametersSettled(t *testing.T) {
	catalog := map[string]parameterCatalogEntry{
		"work_mem":        numericEntry("work_mem", "KILOBYTES", 65536),
		"max_connections": numericEntry("max_connections", unitUndefined, 150),
	}
	require.True(t, parametersSettled(catalog, map[string]string{"work_mem": "64MB", "max_connections": "150"}))
	require.False(t, parametersSettled(catalog, map[string]string{"work_mem": "32MB"}))
	require.False(t, parametersSettled(catalog, map[string]string{"missing": "1"}))

	pending := numericEntry("max_connections", unitUndefined, 150)
	pending.info.IsPendingRestart = true
	catalog["max_connections"] = pending
	require.False(t, parametersSettled(catalog, map[string]string{"max_connections": "150"}))
}

// mockPostgresParametersServer returns an httptest server for GetPostgresParameters:
// it fails the first N requests with a GraphQL error, then succeeds.
func mockPostgresParametersServer(t *testing.T, failFirst int32) (*httptest.Server, *int32) {
	t.Helper()
	var requests int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		n := atomic.AddInt32(&requests, 1)
		if failFirst < 0 || n <= failFirst {
			_, _ = fmt.Fprint(w, `{"errors":[{"message":"service is restarting"}]}`)
			return
		}
		_, _ = fmt.Fprint(w, `{"data":{"getPostgresParameters":{"string_parameters":[],"numeric_parameters":[]}}}`)
	}))
	return srv, &requests
}

func TestFetchParameterCatalogWithRetry(t *testing.T) {
	origInterval, origTimeout := parameterPollInterval, parameterPollTimeout
	parameterPollInterval = 5 * time.Millisecond
	parameterPollTimeout = 50 * time.Millisecond
	t.Cleanup(func() {
		parameterPollInterval = origInterval
		parameterPollTimeout = origTimeout
	})

	t.Run("retries a transient failure then succeeds", func(t *testing.T) {
		srv, requests := mockPostgresParametersServer(t, 1)
		defer srv.Close()
		t.Setenv("TIMESCALE_DEV_URL", srv.URL)
		r := &serviceResource{client: tsClient.NewClient("token", "proj", "test", "1.0.0")}

		catalog, err := r.fetchParameterCatalogWithRetry(context.Background(), "svc-1")
		require.NoError(t, err)
		require.NotNil(t, catalog)
		require.Equal(t, int32(2), atomic.LoadInt32(requests))
	})

	t.Run("returns the last error after the timeout", func(t *testing.T) {
		srv, _ := mockPostgresParametersServer(t, -1)
		defer srv.Close()
		t.Setenv("TIMESCALE_DEV_URL", srv.URL)
		r := &serviceResource{client: tsClient.NewClient("token", "proj", "test", "1.0.0")}

		_, err := r.fetchParameterCatalogWithRetry(context.Background(), "svc-1")
		require.Error(t, err)
		require.Contains(t, err.Error(), "service is restarting")
	})
}
