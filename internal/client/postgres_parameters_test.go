package client

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPostgresParameters_Decode(t *testing.T) {
	body := `{"data":{"getPostgresParameters":{
		"string_parameters":[{"info":{"name":"array_nulls","description":"d","is_user_editable":true,
			"requires_restart":false,"is_tune_overridable":true,"is_pending_restart":false,
			"is_user_modified":false,"ui_priority":1},"current_value":"on","allowed_values":["on","off"]}],
		"numeric_parameters":[{"info":{"name":"work_mem","description":"d","is_user_editable":true,
			"requires_restart":false,"is_tune_overridable":true,"is_pending_restart":false,
			"is_user_modified":true,"ui_priority":3},"unit":"KILOBYTES","current_value":65536,
			"max_allowed_value":2147483647,"min_allowed_value":64}]}}}`

	var resp Response[GetPostgresParametersResponse]
	require.NoError(t, json.Unmarshal([]byte(body), &resp))
	p := resp.Data.PostgresParameters

	require.Len(t, p.StringParameters, 1)
	require.Equal(t, "array_nulls", p.StringParameters[0].Info.Name)
	require.Equal(t, "on", p.StringParameters[0].CurrentValue)
	require.Equal(t, []string{"on", "off"}, p.StringParameters[0].AllowedValues)

	require.Len(t, p.NumericParameters, 1)
	n := p.NumericParameters[0]
	require.Equal(t, "work_mem", n.Info.Name)
	require.True(t, n.Info.IsUserModified)
	require.Equal(t, "KILOBYTES", n.Unit)
	require.Equal(t, 65536.0, n.CurrentValue)
}

func TestParameterErrors_Error(t *testing.T) {
	err := ParameterErrors{
		{Name: "work_mem", ErrorMessage: "invalid value: value 1.0 is smaller than allowed (64.0)"},
		{Name: "max_connections", ErrorMessage: "configuration not allowed for parameter"},
	}
	require.Equal(t,
		"work_mem: invalid value: value 1.0 is smaller than allowed (64.0); max_connections: configuration not allowed for parameter",
		err.Error())
}
