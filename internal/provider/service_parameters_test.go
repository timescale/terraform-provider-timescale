package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
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

func TestChangedParameterKeys(t *testing.T) {
	state := map[string]string{"a": "1", "b": "2", "c": "3"}
	plan := map[string]string{"b": "2", "c": "30", "d": "4"}

	require.Equal(t, []string{"c", "d"}, changedParameterKeys(state, plan))
	require.Equal(t, []string{"b", "c", "d"}, changedParameterKeys(nil, plan))
	require.Empty(t, changedParameterKeys(state, state))
}

func TestRemovedParameterKeysFromPlan(t *testing.T) {
	state := map[string]string{"a": "1", "b": "2"}

	t.Run("genuinely dropped key is reported", func(t *testing.T) {
		plan, diags := types.MapValue(types.StringType, map[string]attr.Value{
			"a": types.StringValue("1"),
		})
		require.False(t, diags.HasError())
		require.Equal(t, []string{"b"}, removedParameterKeysFromPlan(state, plan))
	})

	t.Run("unknown element present in plan is not reported as removed", func(t *testing.T) {
		plan, diags := types.MapValue(types.StringType, map[string]attr.Value{
			"a": types.StringValue("1"),
			"b": types.StringUnknown(),
		})
		require.False(t, diags.HasError())
		require.Empty(t, removedParameterKeysFromPlan(state, plan))
	})

	t.Run("null plan map reports all state keys", func(t *testing.T) {
		require.Equal(t, []string{"a", "b"}, removedParameterKeysFromPlan(state, types.MapNull(types.StringType)))
	})

	t.Run("unknown plan map reports none", func(t *testing.T) {
		require.Empty(t, removedParameterKeysFromPlan(state, types.MapUnknown(types.StringType)))
	})
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
	origInterval, origTimeout, origRetryTimeout := parameterPollInterval, parameterPollTimeout, parameterFetchRetryTimeout
	parameterPollInterval = 5 * time.Millisecond
	parameterPollTimeout = 50 * time.Millisecond
	parameterFetchRetryTimeout = 50 * time.Millisecond
	t.Cleanup(func() {
		parameterPollInterval = origInterval
		parameterPollTimeout = origTimeout
		parameterFetchRetryTimeout = origRetryTimeout
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

// gqlMock serves getBodies/setBodies in order by operationName; the last one repeats once exhausted.
type gqlMock struct {
	t *testing.T

	mu        sync.Mutex
	getBodies []string
	setBodies []string
	getCalls  int
	setCalls  int
	setVars   []map[string]any
}

func newGQLMock(t *testing.T, getBodies, setBodies []string) *gqlMock {
	return &gqlMock{t: t, getBodies: getBodies, setBodies: setBodies}
}

func (m *gqlMock) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, err := io.ReadAll(r.Body)
	require.NoError(m.t, err)
	var body map[string]any
	require.NoError(m.t, json.Unmarshal(data, &body))
	op, _ := body["operationName"].(string)
	w.Header().Set("Content-Type", "application/json")
	switch op {
	case "GetPostgresParameters":
		idx := m.getCalls
		if idx >= len(m.getBodies) {
			idx = len(m.getBodies) - 1
		}
		m.getCalls++
		_, _ = fmt.Fprint(w, m.getBodies[idx])
	case "SetPostgresParameters":
		vars, _ := body["variables"].(map[string]any)
		m.setVars = append(m.setVars, vars)
		idx := m.setCalls
		if idx >= len(m.setBodies) {
			idx = len(m.setBodies) - 1
		}
		m.setCalls++
		_, _ = fmt.Fprint(w, m.setBodies[idx])
	default:
		m.t.Fatalf("unexpected GraphQL operation %q", op)
	}
}

func (m *gqlMock) getCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.getCalls
}

func (m *gqlMock) setCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.setCalls
}

func (m *gqlMock) lastSetVars() map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.setVars) == 0 {
		return nil
	}
	return m.setVars[len(m.setVars)-1]
}

// newTestServiceResourceFor points a fresh client at the mock, following the pattern
// used by TestFetchParameterCatalogWithRetry.
func newTestServiceResourceFor(t *testing.T, mock *gqlMock) *serviceResource {
	t.Helper()
	srv := httptest.NewServer(mock)
	t.Cleanup(srv.Close)
	t.Setenv("TIMESCALE_DEV_URL", srv.URL)
	return &serviceResource{client: tsClient.NewClient("", "project", "test", "test")}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return string(b)
}

func getParamsBody(t *testing.T, p tsClient.PostgresParameters) string {
	t.Helper()
	return mustJSON(t, tsClient.Response[tsClient.GetPostgresParametersResponse]{
		Data: &tsClient.GetPostgresParametersResponse{PostgresParameters: p},
	})
}

func setParamsBody(t *testing.T, errs []tsClient.ParameterError) string {
	t.Helper()
	return mustJSON(t, tsClient.Response[tsClient.SetPostgresParametersResponse]{
		Data: &tsClient.SetPostgresParametersResponse{ParameterErrors: errs},
	})
}

// diagnosticPath returns the path of a diagnostic that must carry one.
func diagnosticPath(t *testing.T, d diag.Diagnostic) path.Path {
	t.Helper()
	wp, ok := d.(diag.DiagnosticWithPath)
	require.True(t, ok, "diagnostic %v does not carry a path", d)
	return wp.Path()
}

func TestApplyPostgresParameters(t *testing.T) {
	origInterval, origTimeout, origRetryTimeout := parameterPollInterval, parameterPollTimeout, parameterFetchRetryTimeout
	parameterPollInterval = 5 * time.Millisecond
	parameterPollTimeout = 50 * time.Millisecond
	parameterFetchRetryTimeout = 50 * time.Millisecond
	t.Cleanup(func() {
		parameterPollInterval = origInterval
		parameterPollTimeout = origTimeout
		parameterFetchRetryTimeout = origRetryTimeout
	})

	t.Run("unknown key produces one error diagnostic and no set call", func(t *testing.T) {
		catalog := getParamsBody(t, tsClient.PostgresParameters{})
		mock := newGQLMock(t, []string{catalog}, nil)
		r := newTestServiceResourceFor(t, mock)

		diags := r.applyPostgresParameters(context.Background(), "svc-1", map[string]string{"nosuch": "1"})
		require.True(t, diags.HasError())
		require.Len(t, diags, 1)
		require.True(t, path.Root(postgresParametersAttr).AtMapKey("nosuch").Equal(diagnosticPath(t, diags[0])))
		require.Equal(t, 0, mock.setCallCount())
	})

	t.Run("not user-editable produces an error and no set call", func(t *testing.T) {
		catalog := getParamsBody(t, tsClient.PostgresParameters{
			NumericParameters: []tsClient.NumericParameter{
				{Info: tsClient.ParameterInfo{Name: "work_mem", IsUserEditable: false}, Unit: "KILOBYTES", CurrentValue: 4096},
			},
		})
		mock := newGQLMock(t, []string{catalog}, nil)
		r := newTestServiceResourceFor(t, mock)

		diags := r.applyPostgresParameters(context.Background(), "svc-1", map[string]string{"work_mem": "64MB"})
		require.True(t, diags.HasError())
		require.Len(t, diags, 1)
		require.Contains(t, diags[0].Detail(), "not user-editable")
		require.Equal(t, 0, mock.setCallCount())
	})

	t.Run("value already semantically equal produces no set call and no diagnostics", func(t *testing.T) {
		catalog := getParamsBody(t, tsClient.PostgresParameters{
			NumericParameters: []tsClient.NumericParameter{
				{Info: tsClient.ParameterInfo{Name: "work_mem", IsUserEditable: true}, Unit: "KILOBYTES", CurrentValue: 65536},
			},
		})
		mock := newGQLMock(t, []string{catalog}, nil)
		r := newTestServiceResourceFor(t, mock)

		diags := r.applyPostgresParameters(context.Background(), "svc-1", map[string]string{"work_mem": "64MB"})
		require.False(t, diags.HasError())
		require.Empty(t, diags)
		require.Equal(t, 0, mock.setCallCount())
	})

	t.Run("valid change sends one set call and maps returned errors to attribute diagnostics", func(t *testing.T) {
		catalog := getParamsBody(t, tsClient.PostgresParameters{
			NumericParameters: []tsClient.NumericParameter{
				{Info: tsClient.ParameterInfo{Name: "work_mem", IsUserEditable: true}, Unit: "KILOBYTES", CurrentValue: 65536},
			},
		})
		setResp := setParamsBody(t, []tsClient.ParameterError{{Name: "work_mem", ErrorMessage: "rejected by tuner"}})
		mock := newGQLMock(t, []string{catalog}, []string{setResp})
		r := newTestServiceResourceFor(t, mock)

		diags := r.applyPostgresParameters(context.Background(), "svc-1", map[string]string{"work_mem": "128MB"})
		require.Equal(t, 1, mock.setCallCount())

		vars := mock.lastSetVars()
		require.NotNil(t, vars)
		numeric, ok := vars["numericParameters"].([]any)
		require.True(t, ok)
		require.Len(t, numeric, 1)
		entry, ok := numeric[0].(map[string]any)
		require.True(t, ok)
		require.Equal(t, "work_mem", entry["name"])
		require.Equal(t, float64(128), entry["value"])
		require.Equal(t, "MEGABYTES", entry["unit"])

		require.True(t, diags.HasError())
		require.Len(t, diags, 1)
		require.True(t, path.Root(postgresParametersAttr).AtMapKey("work_mem").Equal(diagnosticPath(t, diags[0])))
	})

	t.Run("valid change with no errors and a settled follow-up read produces no diagnostics", func(t *testing.T) {
		before := getParamsBody(t, tsClient.PostgresParameters{
			NumericParameters: []tsClient.NumericParameter{
				{Info: tsClient.ParameterInfo{Name: "work_mem", IsUserEditable: true}, Unit: "KILOBYTES", CurrentValue: 65536},
			},
		})
		after := getParamsBody(t, tsClient.PostgresParameters{
			NumericParameters: []tsClient.NumericParameter{
				{Info: tsClient.ParameterInfo{Name: "work_mem", IsUserEditable: true, IsPendingRestart: false}, Unit: "KILOBYTES", CurrentValue: 131072},
			},
		})
		setResp := setParamsBody(t, nil)
		mock := newGQLMock(t, []string{before, after}, []string{setResp})
		r := newTestServiceResourceFor(t, mock)

		diags := r.applyPostgresParameters(context.Background(), "svc-1", map[string]string{"work_mem": "128MB"})
		require.False(t, diags.HasError())
		require.Empty(t, diags)
		require.Equal(t, 1, mock.setCallCount())
	})
}

func TestReadPostgresParameters(t *testing.T) {
	ctx := context.Background()

	newPrior := func(t *testing.T, m map[string]string) types.Map {
		t.Helper()
		v, diags := parameterMapValue(ctx, m)
		require.False(t, diags.HasError())
		return v
	}

	t.Run("value semantically equal to running keeps the state string", func(t *testing.T) {
		catalog := getParamsBody(t, tsClient.PostgresParameters{
			NumericParameters: []tsClient.NumericParameter{
				{Info: tsClient.ParameterInfo{Name: "work_mem"}, Unit: "KILOBYTES", CurrentValue: 65536},
			},
		})
		mock := newGQLMock(t, []string{catalog}, nil)
		r := newTestServiceResourceFor(t, mock)
		prior := newPrior(t, map[string]string{"work_mem": "64MB"})

		got, diags := r.readPostgresParameters(ctx, &tsClient.Service{ID: "svc-1", Status: "READY"}, prior, false)
		require.False(t, diags.HasError())
		require.Equal(t, map[string]string{"work_mem": "64MB"}, knownParameterMap(got))
	})

	t.Run("drift replaces the state string with the formatted running value", func(t *testing.T) {
		catalog := getParamsBody(t, tsClient.PostgresParameters{
			NumericParameters: []tsClient.NumericParameter{
				{Info: tsClient.ParameterInfo{Name: "work_mem"}, Unit: "KILOBYTES", CurrentValue: 32768},
			},
		})
		mock := newGQLMock(t, []string{catalog}, nil)
		r := newTestServiceResourceFor(t, mock)
		prior := newPrior(t, map[string]string{"work_mem": "64MB"})

		got, diags := r.readPostgresParameters(ctx, &tsClient.Service{ID: "svc-1", Status: "READY"}, prior, false)
		require.False(t, diags.HasError())
		require.Equal(t, map[string]string{"work_mem": "32MB"}, knownParameterMap(got))
	})

	t.Run("pending restart keeps the state string even if the running value differs", func(t *testing.T) {
		catalog := getParamsBody(t, tsClient.PostgresParameters{
			NumericParameters: []tsClient.NumericParameter{
				{Info: tsClient.ParameterInfo{Name: "max_connections", IsPendingRestart: true}, Unit: unitUndefined, CurrentValue: 150},
			},
		})
		mock := newGQLMock(t, []string{catalog}, nil)
		r := newTestServiceResourceFor(t, mock)
		prior := newPrior(t, map[string]string{"max_connections": "200"})

		got, diags := r.readPostgresParameters(ctx, &tsClient.Service{ID: "svc-1", Status: "READY"}, prior, false)
		require.False(t, diags.HasError())
		require.Equal(t, map[string]string{"max_connections": "200"}, knownParameterMap(got))
	})

	t.Run("key missing from catalog is dropped", func(t *testing.T) {
		catalog := getParamsBody(t, tsClient.PostgresParameters{})
		mock := newGQLMock(t, []string{catalog}, nil)
		r := newTestServiceResourceFor(t, mock)
		prior := newPrior(t, map[string]string{"stale_key": "1"})

		got, diags := r.readPostgresParameters(ctx, &tsClient.Service{ID: "svc-1", Status: "READY"}, prior, false)
		require.False(t, diags.HasError())
		require.Empty(t, knownParameterMap(got))
	})

	t.Run("non-READY service returns the prior map untouched without any request", func(t *testing.T) {
		mock := newGQLMock(t, nil, nil)
		r := newTestServiceResourceFor(t, mock)
		prior := newPrior(t, map[string]string{"work_mem": "64MB"})

		got, diags := r.readPostgresParameters(ctx, &tsClient.Service{ID: "svc-1", Status: "PAUSED"}, prior, false)
		require.False(t, diags.HasError())
		require.True(t, prior.Equal(got))
		require.Equal(t, 0, mock.getCallCount())
	})

	t.Run("import adopts every user-modified editable parameter, formatted", func(t *testing.T) {
		catalog := getParamsBody(t, tsClient.PostgresParameters{
			NumericParameters: []tsClient.NumericParameter{
				{Info: tsClient.ParameterInfo{Name: "work_mem", IsUserModified: true, IsUserEditable: true}, Unit: "KILOBYTES", CurrentValue: 65536},
				{Info: tsClient.ParameterInfo{Name: "max_connections", IsUserModified: false, IsUserEditable: true}, Unit: unitUndefined, CurrentValue: 100},
				{Info: tsClient.ParameterInfo{Name: "shared_buffers", IsUserModified: true, IsUserEditable: false}, Unit: "KILOBYTES", CurrentValue: 1024},
			},
			StringParameters: []tsClient.StringParameter{
				{Info: tsClient.ParameterInfo{Name: "hot_standby_feedback", IsUserModified: true, IsUserEditable: true}, CurrentValue: "on", AllowedValues: []string{"on", "off"}},
			},
		})
		mock := newGQLMock(t, []string{catalog}, nil)
		r := newTestServiceResourceFor(t, mock)

		got, diags := r.readPostgresParameters(ctx, &tsClient.Service{ID: "svc-1", Status: "READY"}, types.MapNull(types.StringType), true)
		require.False(t, diags.HasError())
		require.Equal(t, map[string]string{"work_mem": "64MB", "hot_standby_feedback": "on"}, knownParameterMap(got))
	})

	t.Run("import with nothing user-modified returns null", func(t *testing.T) {
		catalog := getParamsBody(t, tsClient.PostgresParameters{
			NumericParameters: []tsClient.NumericParameter{
				{Info: tsClient.ParameterInfo{Name: "work_mem", IsUserModified: false}, Unit: "KILOBYTES", CurrentValue: 65536},
			},
		})
		mock := newGQLMock(t, []string{catalog}, nil)
		r := newTestServiceResourceFor(t, mock)

		got, diags := r.readPostgresParameters(ctx, &tsClient.Service{ID: "svc-1", Status: "READY"}, types.MapNull(types.StringType), true)
		require.False(t, diags.HasError())
		require.True(t, got.IsNull())
	})

	t.Run("import on a non-READY service returns null with a warning", func(t *testing.T) {
		mock := newGQLMock(t, nil, nil)
		r := newTestServiceResourceFor(t, mock)

		got, diags := r.readPostgresParameters(ctx, &tsClient.Service{ID: "svc-1", Status: "CONFIGURING"}, types.MapNull(types.StringType), true)
		require.False(t, diags.HasError())
		require.Len(t, diags, 1)
		require.True(t, got.IsNull())
		require.Equal(t, 0, mock.getCallCount())
	})
}
