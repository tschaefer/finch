package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tschaefer/finch/internal/controller"
	"github.com/tschaefer/finch/internal/model"
)

type mockConfig struct {
	version   string
	hostname  string
	database  string
	id        string
	createdAt string
	library   string
	secret    string
	username  string
	password  string
}

func (m *mockConfig) Version() string               { return m.version }
func (m *mockConfig) Hostname() string              { return m.hostname }
func (m *mockConfig) Database() string              { return m.database }
func (m *mockConfig) Id() string                    { return m.id }
func (m *mockConfig) CreatedAt() string             { return m.createdAt }
func (m *mockConfig) Library() string               { return m.library }
func (m *mockConfig) Secret() string                { return m.secret }
func (m *mockConfig) Credentials() (string, string) { return m.username, m.password }

var mockedConfig = mockConfig{
	version:   "1.0.0",
	hostname:  "localhost",
	database:  "test.db",
	id:        "test-id",
	createdAt: "2025-01-01T00:00:00Z",
	library:   "test-library",
	secret:    "1suNCrW7sWlPbU+YCfdGQI7z3ZMo9Ru2GNV4h69QzaM=",
	username:  "test-user",
	password:  "test-password",
}

type mockController struct{}

func (m *mockController) RegisterAgent(hostname string, tags, logSources []string) (string, error) {
	return "rid:12345", nil
}
func (m *mockController) DeregisterAgent(rid string) error {
	return nil
}
func (m *mockController) CreateAgentConfig(rid string) ([]byte, error) {
	return []byte(`{ "config": true }`), nil
}
func (m *mockController) ListAgents() ([]map[string]string, error) {
	return []map[string]string{
		{"rid": "rid:12345", "hostname": "node1"},
		{"rid": "rid:67890", "hostname": "node2"},
	}, nil
}
func (m *mockController) GetAgent(rid string) (*model.Agent, error) {
	return nil, controller.ErrAgentNotFound
}

func Test_ReturnsError404_PathNotFound(t *testing.T) {
	handler := New(&mockController{}, &mockedConfig)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/nonexistent", nil)
	rr := httptest.NewRecorder()
	handler.Router().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", status)
	}

	var response map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["detail"] != "route not found" {
		t.Errorf("expected error message 'not found', got %s", response["detail"])
	}
}

func Test_ReturnsError405_InvalidMethod(t *testing.T) {
	handler := New(&mockController{}, &mockedConfig)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/agent", nil)
	rr := httptest.NewRecorder()
	handler.Router().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", status)
	}

	var response map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["detail"] != "method not allowed" {
		t.Errorf("expected error message 'method not allowed', got %s", response["detail"])
	}
}

func Test_ReturnsError401_Unauthorized(t *testing.T) {
	handler := New(&mockController{}, &mockedConfig)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agent/4711/config", nil)
	rr := httptest.NewRecorder()
	handler.Router().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", status)
	}

	var response map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["detail"] != "unauthorized" {
		t.Errorf("expected error message 'unauthorized', got %s", response["detail"])
	}
}

func Test_CreateAgentSuccess(t *testing.T) {
	handler := New(&mockController{}, &mockedConfig)

	body := `{
		"hostname": "node1",
		"tags": ["web", "prod"],
		"log_sources": ["docker", "journal"]
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	username, password := mockedConfig.Credentials()
	req.SetBasicAuth(username, password)

	rr := httptest.NewRecorder()
	handler.Router().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", status)
	}

	var response map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["rid"] != "rid:12345" {
		t.Errorf("unexpected rid value: %s", response["rid"])
	}
}

func Test_DeleteAgentSuccess(t *testing.T) {
	handler := New(&mockController{}, &mockedConfig)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/agent/rid:12345", nil)
	username, password := mockedConfig.Credentials()
	req.SetBasicAuth(username, password)

	rr := httptest.NewRecorder()
	handler.Router().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", status)
	}

	if rr.Body.Len() != 0 {
		t.Error("expected empty response body")
	}
}

func Test_GetAgentConfigSuccess(t *testing.T) {
	handler := New(&mockController{}, &mockedConfig)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agent/rid:12345/config", nil)
	username, password := mockedConfig.Credentials()
	req.SetBasicAuth(username, password)

	rr := httptest.NewRecorder()
	handler.Router().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}

	var response map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if _, ok := response["config"]; !ok {
		t.Error("expected config key in response")
	}
}

func Test_ListAgentsSuccess(t *testing.T) {
	handler := New(&mockController{}, &mockedConfig)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agent", nil)
	username, password := mockedConfig.Credentials()
	req.SetBasicAuth(username, password)

	rr := httptest.NewRecorder()
	handler.Router().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}

	var response []map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response) == 0 {
		t.Error("expected non-empty agent list")
	}
}

func Test_GetAgentNotFound(t *testing.T) {
	handler := New(&mockController{}, &mockedConfig)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agent/rid:99999", nil)
	username, password := mockedConfig.Credentials()
	req.SetBasicAuth(username, password)

	rr := httptest.NewRecorder()
	handler.Router().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", status)
	}

	var response map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !strings.HasSuffix(response["detail"], "agent not found") {
		t.Errorf("expected error message 'agent not found', got %s", response["detail"])
	}
}
