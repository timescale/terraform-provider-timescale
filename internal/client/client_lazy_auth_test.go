package client

import (
	"context"
	"testing"
)

func TestClientIsConfigured(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		projectID string
		expected  bool
	}{
		{
			name:      "configured with token and project",
			token:     "valid-token",
			projectID: "valid-project",
			expected:  true,
		},
		{
			name:      "missing token",
			token:     "",
			projectID: "valid-project",
			expected:  false,
		},
		{
			name:      "missing project",
			token:     "valid-token",
			projectID: "",
			expected:  false,
		},
		{
			name:      "missing both token and project",
			token:     "",
			projectID: "",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				token:     tt.token,
				projectID: tt.projectID,
			}
			if got := client.IsConfigured(); got != tt.expected {
				t.Errorf("IsConfigured() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestClientDoUnconfigured(t *testing.T) {
	// Client with projectID but no token should error
	client := NewClient("", "valid-project", "test", "1.0")
	req := map[string]interface{}{
		"operationName": "TestOperation",
		"query":         "query { test }",
	}
	var resp interface{}

	err := client.do(context.Background(), req, &resp)
	if err == nil {
		t.Errorf("Expected error when client has no token, got nil")
	}
	if err.Error() != "timescale provider is not configured. Please provide project_id and either access_token or (access_key and secret_key) to use Timescale resources" {
		t.Errorf("Got unexpected error message: %v", err)
	}
}
