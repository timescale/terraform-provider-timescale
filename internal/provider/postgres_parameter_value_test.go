package provider

import (
	"testing"

	"github.com/stretchr/testify/require"

	tsClient "github.com/timescale/terraform-provider-timescale/internal/client"
)

func numericEntry(name, unit string, value float64) parameterCatalogEntry {
	return parameterCatalogEntry{
		info:    tsClient.ParameterInfo{Name: name, IsUserEditable: true},
		numeric: true,
		unit:    unit,
		value:   value,
	}
}

func stringEntry(name, value string) parameterCatalogEntry {
	return parameterCatalogEntry{
		info:     tsClient.ParameterInfo{Name: name, IsUserEditable: true},
		strValue: value,
	}
}

func TestParseParameterValue(t *testing.T) {
	tests := []struct {
		name    string
		entry   parameterCatalogEntry
		raw     string
		want    parsedParameterValue
		wantErr string
	}{
		{
			name:  "string passes through",
			entry: stringEntry("log_statement", "none"),
			raw:   "all",
			want:  parsedParameterValue{str: "all"},
		},
		{
			name:  "boolean is a string",
			entry: stringEntry("hot_standby_feedback", "off"),
			raw:   "on",
			want:  parsedParameterValue{str: "on"},
		},
		{
			name:  "plain integer",
			entry: numericEntry("max_connections", unitUndefined, 100),
			raw:   "200",
			want:  parsedParameterValue{numeric: true, value: 200, unit: unitUndefined},
		},
		{
			name:  "plain decimal",
			entry: numericEntry("random_page_cost", unitUndefined, 4),
			raw:   "2.5",
			want:  parsedParameterValue{numeric: true, value: 2.5, unit: unitUndefined},
		},
		{
			name:    "plain numeric rejects text",
			entry:   numericEntry("max_connections", unitUndefined, 100),
			raw:     "many",
			wantErr: "is not a number",
		},
		{
			name:  "memory with unit",
			entry: numericEntry("work_mem", "KILOBYTES", 4096),
			raw:   "64MB",
			want:  parsedParameterValue{numeric: true, value: 64, unit: "MEGABYTES"},
		},
		{
			name:  "memory with space before unit",
			entry: numericEntry("work_mem", "KILOBYTES", 4096),
			raw:   "64 MB",
			want:  parsedParameterValue{numeric: true, value: 64, unit: "MEGABYTES"},
		},
		{
			name:  "time with unit",
			entry: numericEntry("statement_timeout", "MILLISECONDS", 0),
			raw:   "30s",
			want:  parsedParameterValue{numeric: true, value: 30, unit: "SECONDS"},
		},
		{
			name:  "minutes",
			entry: numericEntry("max_standby_streaming_delay", "MILLISECONDS", 30000),
			raw:   "5min",
			want:  parsedParameterValue{numeric: true, value: 5, unit: "MINUTES"},
		},
		{
			name:  "fraction with unit moves to smaller unit",
			entry: numericEntry("shared_buffers", "KILOBYTES", 1024),
			raw:   "1.5GB",
			want:  parsedParameterValue{numeric: true, value: 1536, unit: "MEGABYTES"},
		},
		{
			name:  "fraction needing two steps",
			entry: numericEntry("statement_timeout", "MILLISECONDS", 0),
			raw:   "0.0005s",
			want:  parsedParameterValue{numeric: true, value: 500, unit: "MICROSECONDS"},
		},
		{
			name:    "fraction below smallest unit",
			entry:   numericEntry("work_mem", "KILOBYTES", 4096),
			raw:     "0.5B",
			wantErr: "cannot be expressed as a whole number",
		},
		{
			name:    "bare positive number on unit parameter",
			entry:   numericEntry("work_mem", "KILOBYTES", 4096),
			raw:     "65536",
			wantErr: "needs an explicit unit",
		},
		{
			name:  "bare zero on unit parameter is allowed",
			entry: numericEntry("statement_timeout", "MILLISECONDS", 0),
			raw:   "0",
			want:  parsedParameterValue{numeric: true, value: 0, unit: "MILLISECONDS"},
		},
		{
			name:  "bare negative on unit parameter is allowed",
			entry: numericEntry("max_slot_wal_keep_size", "MEGABYTES", 10240),
			raw:   "-1",
			want:  parsedParameterValue{numeric: true, value: -1, unit: "MEGABYTES"},
		},
		{
			name:    "memory unit on time parameter",
			entry:   numericEntry("statement_timeout", "MILLISECONDS", 0),
			raw:     "30MB",
			wantErr: "takes time units",
		},
		{
			name:    "unknown suffix",
			entry:   numericEntry("work_mem", "KILOBYTES", 4096),
			raw:     "64mb",
			wantErr: "unknown unit \"mb\"",
		},
		{
			name:  "1.001s with exact arithmetic",
			entry: numericEntry("statement_timeout", "MILLISECONDS", 0),
			raw:   "1.001s",
			want:  parsedParameterValue{numeric: true, value: 1001, unit: "MILLISECONDS"},
		},
		{
			name:    "fractional microseconds below smallest unit",
			entry:   numericEntry("statement_timeout", "MICROSECONDS", 0),
			raw:     "0.5us",
			wantErr: "cannot be expressed as a whole number",
		},
		{
			name:    "garbage on unit parameter",
			entry:   numericEntry("work_mem", "KILOBYTES", 4096),
			raw:     "lots",
			wantErr: "is not a number followed by a unit",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseParameterValue(tc.entry, tc.raw)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestFormatParameterValue(t *testing.T) {
	require.Equal(t, "65536kB", formatParameterValue(numericEntry("work_mem", "KILOBYTES", 65536)))
	require.Equal(t, "30000ms", formatParameterValue(numericEntry("statement_timeout", "MILLISECONDS", 30000)))
	require.Equal(t, "200", formatParameterValue(numericEntry("max_connections", unitUndefined, 200)))
	require.Equal(t, "2.5", formatParameterValue(numericEntry("random_page_cost", unitUndefined, 2.5)))
	require.Equal(t, "-1MB", formatParameterValue(numericEntry("max_slot_wal_keep_size", "MEGABYTES", -1)))
	require.Equal(t, "on", formatParameterValue(stringEntry("hot_standby_feedback", "on")))
}

func TestParameterValueEqual(t *testing.T) {
	tests := []struct {
		name  string
		entry parameterCatalogEntry
		raw   string
		want  bool
	}{
		{"same unit", numericEntry("work_mem", "KILOBYTES", 65536), "65536kB", true},
		{"different unit same amount", numericEntry("work_mem", "KILOBYTES", 65536), "64MB", true},
		{"different amount", numericEntry("work_mem", "KILOBYTES", 65536), "32MB", false},
		{"time across units", numericEntry("statement_timeout", "MILLISECONDS", 30000), "30s", true},
		{"bare zero", numericEntry("statement_timeout", "MILLISECONDS", 0), "0", true},
		{"negative sentinel", numericEntry("max_slot_wal_keep_size", "MEGABYTES", -1), "-1", true},
		{"plain float formatting", numericEntry("random_page_cost", unitUndefined, 2), "2.0", true},
		{"plain differs", numericEntry("max_connections", unitUndefined, 100), "200", false},
		{"string equal", stringEntry("log_statement", "all"), "all", true},
		{"string case differs", stringEntry("hot_standby_feedback", "on"), "ON", false},
		{"unparseable", numericEntry("work_mem", "KILOBYTES", 65536), "lots", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, parameterValueEqual(tc.entry, tc.raw))
		})
	}
}

func TestBuildParameterCatalog(t *testing.T) {
	maxVal := 500.0
	p := &tsClient.PostgresParameters{
		StringParameters: []tsClient.StringParameter{
			{Info: tsClient.ParameterInfo{Name: "log_statement"}, CurrentValue: "none", AllowedValues: []string{"none", "all"}},
		},
		NumericParameters: []tsClient.NumericParameter{
			{Info: tsClient.ParameterInfo{Name: "max_connections", RequiresRestart: true}, Unit: "UNDEFINED", CurrentValue: 100, MaxAllowedValue: &maxVal},
			{Info: tsClient.ParameterInfo{Name: "work_mem"}, Unit: "KILOBYTES", CurrentValue: 4096},
		},
	}
	catalog := buildParameterCatalog(p)
	require.Len(t, catalog, 3)

	require.False(t, catalog["log_statement"].numeric)
	require.Equal(t, "none", catalog["log_statement"].strValue)

	require.True(t, catalog["max_connections"].numeric)
	require.Equal(t, unitUndefined, catalog["max_connections"].unit)
	require.Equal(t, 100.0, catalog["max_connections"].value)
	require.True(t, catalog["max_connections"].info.RequiresRestart)

	require.Equal(t, "KILOBYTES", catalog["work_mem"].unit)
}
