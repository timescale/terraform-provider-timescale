package provider_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/timescale/terraform-provider-timescale/internal/provider"
)

// OperationHandler handles a specific GraphQL operation
type OperationHandler func(t *testing.T, req map[string]interface{}) map[string]interface{}

// MockServer wraps httptest.Server with GraphQL operation routing
type MockServer struct {
	*httptest.Server
	handlers map[string]OperationHandler
	t        *testing.T
}

// NewMockServer creates a mock server with default handlers for common operations
func NewMockServer(t *testing.T) *MockServer {
	m := &MockServer{
		handlers: make(map[string]OperationHandler),
		t:        t,
	}

	// Default handler for JWT authentication
	m.handlers["GetJWTForClientCredentials"] = func(t *testing.T, req map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{
			"data": map[string]interface{}{
				"getJWTForClientCredentials": "mock-jwt-token",
			},
		}
	}

	m.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)

		operationName, _ := req["operationName"].(string)

		var response map[string]interface{}
		if handler, ok := m.handlers[operationName]; ok {
			response = handler(t, req)
		} else {
			t.Logf("Unhandled operation: %s", operationName)
			response = map[string]interface{}{
				"errors": []map[string]string{
					{"message": "unknown operation: " + operationName},
				},
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))

	return m
}

// Handle registers a handler for a specific operation
func (m *MockServer) Handle(operationName string, handler OperationHandler) *MockServer {
	m.handlers[operationName] = handler
	return m
}

// SetupEnv sets the environment variables needed for testing
func (m *MockServer) SetupEnv(t *testing.T) {
	t.Setenv("TF_ACC", "1")
	t.Setenv("TIMESCALE_DEV_URL", m.URL)
	t.Setenv("TF_VAR_ts_access_key", "test-access-key")
	t.Setenv("TF_VAR_ts_secret_key", "test-secret-key")
	t.Setenv("TF_VAR_ts_project_id", "test-project-id")
}

// TestProviderFactories returns the provider factories for testing
func TestProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"timescale": providerserver.NewProtocol6WithError(provider.New("test")()),
	}
}

// ProviderConfig returns the common provider configuration for tests
const ProviderConfig = `
variable "ts_access_key" {
  type = string
}

variable "ts_secret_key" {
  type = string
}

variable "ts_project_id" {
  type = string
}

provider "timescale" {
  access_key = var.ts_access_key
  secret_key = var.ts_secret_key
  project_id = var.ts_project_id
}
`
