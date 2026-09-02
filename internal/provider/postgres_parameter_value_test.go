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

func booleanEntry(value string) parameterCatalogEntry {
	return parameterCatalogEntry{
		info:     tsClient.ParameterInfo{Name: "hot_standby_feedback", IsUserEditable: true},
		strValue: value,
		boolean:  true,
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
			name:  "boolean accepts true",
			entry: booleanEntry("off"),
			raw:   "true",
			want:  parsedParameterValue{str: "on"},
		},
		{
			name:  "boolean accepts TRUE case-insensitively",
			entry: booleanEntry("off"),
			raw:   "TRUE",
			want:  parsedParameterValue{str: "on"},
		},
		{
			name:  "boolean accepts yes",
			entry: booleanEntry("off"),
			raw:   "yes",
			want:  parsedParameterValue{str: "on"},
		},
		{
			name:  "boolean accepts 1",
			entry: booleanEntry("off"),
			raw:   "1",
			want:  parsedParameterValue{str: "on"},
		},
		{
			name:  "boolean accepts off",
			entry: booleanEntry("on"),
			raw:   "off",
			want:  parsedParameterValue{str: "off"},
		},
		{
			name:  "boolean accepts False case-insensitively",
			entry: booleanEntry("on"),
			raw:   "False",
			want:  parsedParameterValue{str: "off"},
		},
		{
			name:  "boolean accepts 0",
			entry: booleanEntry("on"),
			raw:   "0",
			want:  parsedParameterValue{str: "off"},
		},
		{
			name:    "boolean rejects unknown spelling",
			entry:   booleanEntry("on"),
			raw:     "maybe",
			wantErr: "is not a boolean; use on or off",
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
			name:    "plain numeric rejects NaN",
			entry:   numericEntry("max_connections", unitUndefined, 100),
			raw:     "NaN",
			wantErr: "is not a number",
		},
		{
			name:    "unit parameter rejects bare Inf",
			entry:   numericEntry("work_mem", "KILOBYTES", 4096),
			raw:     "-Inf",
			wantErr: "is not a number followed by a unit",
		},
		{
			name:  "unknown API unit takes a bare number",
			entry: numericEntry("some_param", "BLOCKS", 8),
			raw:   "16",
			want:  parsedParameterValue{numeric: true, value: 16, unit: "BLOCKS"},
		},
		{
			name:  "unit is not promoted above the written one",
			entry: numericEntry("work_mem", "KILOBYTES", 4096),
			raw:   "1024kB",
			want:  parsedParameterValue{numeric: true, value: 1024, unit: "KILOBYTES"},
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
		{
			name:  "large fractional day value within relative tolerance",
			entry: numericEntry("statement_timeout", "MILLISECONDS", 0),
			raw:   "0.7d",
			want:  parsedParameterValue{numeric: true, value: 1008, unit: "MINUTES"},
		},
		{
			name:  "another large fractional day value within relative tolerance",
			entry: numericEntry("statement_timeout", "MILLISECONDS", 0),
			raw:   "1.1d",
			want:  parsedParameterValue{numeric: true, value: 1584, unit: "MINUTES"},
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
	require.Equal(t, "64MB", formatParameterValue(numericEntry("work_mem", "KILOBYTES", 65536)))
	require.Equal(t, "30s", formatParameterValue(numericEntry("statement_timeout", "MILLISECONDS", 30000)))
	require.Equal(t, "32MB", formatParameterValue(numericEntry("work_mem", "KILOBYTES", 32768)))
	require.Equal(t, "1GB", formatParameterValue(numericEntry("shared_buffers", "KILOBYTES", 1048576)))
	require.Equal(t, "90s", formatParameterValue(numericEntry("statement_timeout", "MILLISECONDS", 90000)))
	require.Equal(t, "1d", formatParameterValue(numericEntry("statement_timeout", "MILLISECONDS", 86400000)))
	require.Equal(t, "1500ms", formatParameterValue(numericEntry("statement_timeout", "MILLISECONDS", 1500)))
	require.Equal(t, "0ms", formatParameterValue(numericEntry("statement_timeout", "MILLISECONDS", 0)))
	require.Equal(t, "-1MB", formatParameterValue(numericEntry("max_slot_wal_keep_size", "MEGABYTES", -1)))
	require.Equal(t, "3kB", formatParameterValue(numericEntry("work_mem", "KILOBYTES", 3)))
	require.Equal(t, "200", formatParameterValue(numericEntry("max_connections", unitUndefined, 200)))
	require.Equal(t, "2.5", formatParameterValue(numericEntry("random_page_cost", unitUndefined, 2.5)))
	require.Equal(t, "8", formatParameterValue(numericEntry("some_param", "BLOCKS", 8)))
	require.Equal(t, "512kB", formatParameterValue(numericEntry("work_mem", "MEGABYTES", 0.5)))
	require.Equal(t, "on", formatParameterValue(stringEntry("hot_standby_feedback", "on")))
}

func TestFormatParameterValueRoundTrip(t *testing.T) {
	entries := []parameterCatalogEntry{
		numericEntry("work_mem", "KILOBYTES", 65536),
		numericEntry("statement_timeout", "MILLISECONDS", 90000),
		numericEntry("statement_timeout", "MILLISECONDS", 0),
		numericEntry("max_slot_wal_keep_size", "MEGABYTES", -1),
		numericEntry("max_connections", unitUndefined, 200),
	}
	for _, entry := range entries {
		require.True(t, parameterValueEqual(entry, formatParameterValue(entry)))
	}
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
		{"boolean true equals on", booleanEntry("on"), "true", true},
		{"boolean OFF equals off", booleanEntry("off"), "OFF", true},
		{"unparseable", numericEntry("work_mem", "KILOBYTES", 65536), "lots", false},
		{"unknown API unit", numericEntry("some_param", "BLOCKS", 8), "8", true},
		{"unknown API unit differs", numericEntry("some_param", "BLOCKS", 8), "16", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, parameterValueEqual(tc.entry, tc.raw))
		})
	}
}

func TestBuildParameterCatalog(t *testing.T) {
	p := &tsClient.PostgresParameters{
		StringParameters: []tsClient.StringParameter{
			{Info: tsClient.ParameterInfo{Name: "log_statement"}, CurrentValue: "none"},
			{Info: tsClient.ParameterInfo{Name: "hot_standby_feedback"}, CurrentValue: "on", AllowedValues: []string{"off", "on"}},
			{Info: tsClient.ParameterInfo{Name: "constraint_exclusion"}, CurrentValue: "none", AllowedValues: []string{"none", "all"}},
			{Info: tsClient.ParameterInfo{Name: "empty_allowed"}, CurrentValue: "x", AllowedValues: []string{}},
		},
		NumericParameters: []tsClient.NumericParameter{
			{Info: tsClient.ParameterInfo{Name: "max_connections", RequiresRestart: true}, Unit: "UNDEFINED", CurrentValue: 100},
			{Info: tsClient.ParameterInfo{Name: "work_mem"}, Unit: "KILOBYTES", CurrentValue: 4096},
		},
	}
	catalog := buildParameterCatalog(p)
	require.Len(t, catalog, 6)

	require.False(t, catalog["log_statement"].numeric)
	require.Equal(t, "none", catalog["log_statement"].strValue)

	require.True(t, catalog["hot_standby_feedback"].boolean)
	require.False(t, catalog["constraint_exclusion"].boolean)
	require.False(t, catalog["empty_allowed"].boolean)

	require.True(t, catalog["max_connections"].numeric)
	require.Equal(t, unitUndefined, catalog["max_connections"].unit)
	require.Equal(t, 100.0, catalog["max_connections"].value)
	require.True(t, catalog["max_connections"].info.RequiresRestart)

	require.Equal(t, "KILOBYTES", catalog["work_mem"].unit)
}
