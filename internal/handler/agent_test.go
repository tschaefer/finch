package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tschaefer/finch/internal/controller"
	"github.com/tschaefer/finch/internal/model"
)

type mockConfig struct {
	version   string
	hostname  string
	database  string
	profiler  string
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
func (m *mockConfig) Profiler() string              { return m.profiler }
func (m *mockConfig) Id() string                    { return m.id }
func (m *mockConfig) CreatedAt() string             { return m.createdAt }
func (m *mockConfig) Library() string               { return m.library }
func (m *mockConfig) Secret() string                { return m.secret }
func (m *mockConfig) Credentials() (string, string) { return m.username, m.password }

var mockedConfig = mockConfig{
	version:   "1.0.0",
	hostname:  "localhost",
	database:  "test.db",
	profiler:  "http://localhost:4040",
	id:        "test-id",
	createdAt: "2025-01-01T00:00:00Z",
	library:   "test-library",
	secret:    "1suNCrW7sWlPbU+YCfdGQI7z3ZMo9Ru2GNV4h69QzaM=",
	username:  "test-user",
	password:  "test-password",
}

type mockController struct{}

func (m *mockController) RegisterAgent(data *controller.Agent) (string, error) {
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
	assert.NotNil(t, handler, "create handler")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/nonexistent", nil)
	rr := httptest.NewRecorder()
	handler.Router().ServeHTTP(rr, req)

	assert.EqualValues(t, http.StatusNotFound, rr.Code, "http status")

	var response map[string]string
	err := json.NewDecoder(rr.Body).Decode(&response)
	assert.NoError(t, err, "decode response")

	assert.Equal(t, "route not found", response["detail"], "error message")
}

func Test_ReturnsError405_InvalidMethod(t *testing.T) {
	handler := New(&mockController{}, &mockedConfig)
	assert.NotNil(t, handler, "create handler")

	req := httptest.NewRequest(http.MethodPut, "/api/v1/agent", nil)
	rr := httptest.NewRecorder()
	handler.Router().ServeHTTP(rr, req)

	assert.EqualValues(t, http.StatusMethodNotAllowed, rr.Code, "http status")

	var response map[string]string
	err := json.NewDecoder(rr.Body).Decode(&response)
	assert.NoError(t, err, "decode response")

	assert.Equal(t, "method not allowed", response["detail"], "error message")
}

func Test_ReturnsError401_Unauthorized(t *testing.T) {
	handler := New(&mockController{}, &mockedConfig)
	assert.NotNil(t, handler, "create handler")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agent/4711/config", nil)
	rr := httptest.NewRecorder()
	handler.Router().ServeHTTP(rr, req)

	assert.EqualValues(t, http.StatusUnauthorized, rr.Code, "http status")

	var response map[string]string
	err := json.NewDecoder(rr.Body).Decode(&response)
	assert.NoError(t, err, "decode response")

	assert.Equal(t, "unauthorized", response["detail"], "error message")
}

func Test_ReturnsInfoHeaders(t *testing.T) {
	handler := New(&mockController{}, &mockedConfig)
	assert.NotNil(t, handler, "create handler")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agent", nil)
	username, password := mockedConfig.Credentials()
	req.SetBasicAuth(username, password)

	rr := httptest.NewRecorder()
	handler.Router().ServeHTTP(rr, req)

	assert.EqualValues(t, http.StatusOK, rr.Code, "http status")

	assert.Contains(t, rr.Header(), "X-Finch-Release", "X-Finch-Release header present")
	assert.Contains(t, rr.Header(), "X-Finch-Commit", "X-Finch-Commit header present")
}

func Test_CreateAgentSuccess(t *testing.T) {
	handler := New(&mockController{}, &mockedConfig)
	assert.NotNil(t, handler, "create handler")

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

	assert.EqualValues(t, http.StatusCreated, rr.Code, "http status")

	var response map[string]string
	err := json.NewDecoder(rr.Body).Decode(&response)
	assert.NoError(t, err, "decode response")

	assert.Equal(t, "rid:12345", response["rid"], "rid value")
}

func Test_DeleteAgentSuccess(t *testing.T) {
	handler := New(&mockController{}, &mockedConfig)
	assert.NotNil(t, handler, "create handler")

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/agent/rid:12345", nil)
	username, password := mockedConfig.Credentials()
	req.SetBasicAuth(username, password)

	rr := httptest.NewRecorder()
	handler.Router().ServeHTTP(rr, req)

	assert.EqualValues(t, http.StatusNoContent, rr.Code, "http status")

	assert.Empty(t, rr.Body.String(), "response body")
}

func Test_GetAgentConfigSuccess(t *testing.T) {
	handler := New(&mockController{}, &mockedConfig)
	assert.NotNil(t, handler, "create handler")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agent/rid:12345/config", nil)
	username, password := mockedConfig.Credentials()
	req.SetBasicAuth(username, password)

	rr := httptest.NewRecorder()
	handler.Router().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}

	assert.EqualValues(t, http.StatusOK, rr.Code, "http status")

	var response map[string]any
	err := json.NewDecoder(rr.Body).Decode(&response)
	assert.NoError(t, err, "decode response")

	assert.Equal(t, true, response["config"], "config value")
}

func Test_ListAgentsSuccess(t *testing.T) {
	handler := New(&mockController{}, &mockedConfig)
	assert.NotNil(t, handler, "create handler")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agent", nil)
	username, password := mockedConfig.Credentials()
	req.SetBasicAuth(username, password)

	rr := httptest.NewRecorder()
	handler.Router().ServeHTTP(rr, req)

	assert.EqualValues(t, http.StatusOK, rr.Code, "http status")

	var response []map[string]string
	err := json.NewDecoder(rr.Body).Decode(&response)
	assert.NoError(t, err, "decode response")

	assert.Len(t, response, 2, "agents count")
}

func Test_GetAgentNotFound(t *testing.T) {
	handler := New(&mockController{}, &mockedConfig)
	assert.NotNil(t, handler, "create handler")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agent/rid:99999", nil)
	username, password := mockedConfig.Credentials()
	req.SetBasicAuth(username, password)

	rr := httptest.NewRecorder()
	handler.Router().ServeHTTP(rr, req)

	assert.EqualValues(t, http.StatusNotFound, rr.Code, "http status")

	var response map[string]string
	err := json.NewDecoder(rr.Body).Decode(&response)
	assert.NoError(t, err, "decode response")

	assert.Contains(t, response["detail"], "agent not found", "error message")
}
