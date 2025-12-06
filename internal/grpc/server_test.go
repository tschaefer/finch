package grpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tschaefer/finch/api"
	"github.com/tschaefer/finch/internal/controller"
	"github.com/tschaefer/finch/internal/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	if data.Hostname == "existing" {
		return "", controller.ErrAgentAlreadyExists
	}
	return "rid:12345", nil
}

func (m *mockController) DeregisterAgent(rid string) error {
	if rid == "rid:notfound" {
		return controller.ErrAgentNotFound
	}
	return nil
}

func (m *mockController) CreateAgentConfig(rid string) ([]byte, error) {
	if rid == "rid:notfound" {
		return nil, controller.ErrAgentNotFound
	}
	return []byte("config content"), nil
}

func (m *mockController) ListAgents() ([]map[string]string, error) {
	return []map[string]string{
		{"rid": "rid:12345", "hostname": "node1"},
		{"rid": "rid:67890", "hostname": "node2"},
	}, nil
}

func (m *mockController) GetAgent(rid string) (*model.Agent, error) {
	if rid == "rid:notfound" {
		return nil, controller.ErrAgentNotFound
	}
	return &model.Agent{
		ResourceId: "rid:12345",
		Hostname:   "node1",
	}, nil
}

func TestRegisterAgent_Success(t *testing.T) {
	server := NewAgentServer(&mockController{}, &mockedConfig)

	req := &api.RegisterAgentRequest{
		Hostname:   "test-host",
		LogSources: []string{"journal://"},
		Metrics:    true,
	}

	resp, err := server.RegisterAgent(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "rid:12345", resp.Rid)
}

func TestRegisterAgent_AlreadyExists(t *testing.T) {
	server := NewAgentServer(&mockController{}, &mockedConfig)

	req := &api.RegisterAgentRequest{
		Hostname:   "existing",
		LogSources: []string{"journal://"},
	}

	resp, err := server.RegisterAgent(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.AlreadyExists, st.Code())
}

func TestRegisterAgent_InvalidArgument(t *testing.T) {
	server := NewAgentServer(&mockController{}, &mockedConfig)

	req := &api.RegisterAgentRequest{
		Hostname:   "",
		LogSources: []string{"journal://"},
	}

	resp, err := server.RegisterAgent(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestDeregisterAgent_Success(t *testing.T) {
	server := NewAgentServer(&mockController{}, &mockedConfig)

	req := &api.DeregisterAgentRequest{
		Rid: "rid:12345",
	}

	resp, err := server.DeregisterAgent(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestDeregisterAgent_InvalidArgument(t *testing.T) {
	server := NewAgentServer(&mockController{}, &mockedConfig)

	req := &api.DeregisterAgentRequest{
		Rid: "",
	}

	resp, err := server.DeregisterAgent(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestDeregisterAgent_NotFound(t *testing.T) {
	server := NewAgentServer(&mockController{}, &mockedConfig)

	req := &api.DeregisterAgentRequest{
		Rid: "rid:notfound",
	}

	resp, err := server.DeregisterAgent(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestGetAgent_Success(t *testing.T) {
	server := NewAgentServer(&mockController{}, &mockedConfig)

	req := &api.GetAgentRequest{
		Rid: "rid:12345",
	}

	resp, err := server.GetAgent(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "rid:12345", resp.ResourceId)
	assert.Equal(t, "node1", resp.Hostname)
}

func TestGetAgent_InvalidArgument(t *testing.T) {
	server := NewAgentServer(&mockController{}, &mockedConfig)

	req := &api.GetAgentRequest{
		Rid: "",
	}

	resp, err := server.GetAgent(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestGetAgent_NotFound(t *testing.T) {
	server := NewAgentServer(&mockController{}, &mockedConfig)

	req := &api.GetAgentRequest{
		Rid: "rid:notfound",
	}

	resp, err := server.GetAgent(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestListAgents_Success(t *testing.T) {
	server := NewAgentServer(&mockController{}, &mockedConfig)

	req := &api.ListAgentsRequest{}

	resp, err := server.ListAgents(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Agents, 2)
	assert.Equal(t, "rid:12345", resp.Agents[0].Rid)
	assert.Equal(t, "node1", resp.Agents[0].Hostname)
}

func TestGetAgentConfig_Success(t *testing.T) {
	server := NewAgentServer(&mockController{}, &mockedConfig)

	req := &api.GetAgentConfigRequest{
		Rid: "rid:12345",
	}

	resp, err := server.GetAgentConfig(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, []byte("config content"), resp.Config)
}

func TestGetAgentConfig_InvalidArgument(t *testing.T) {
	server := NewAgentServer(&mockController{}, &mockedConfig)

	req := &api.GetAgentConfigRequest{
		Rid: "",
	}

	resp, err := server.GetAgentConfig(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestGetAgentConfig_NotFound(t *testing.T) {
	server := NewAgentServer(&mockController{}, &mockedConfig)

	req := &api.GetAgentConfigRequest{
		Rid: "rid:notfound",
	}

	resp, err := server.GetAgentConfig(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestGetServiceInfo_Success(t *testing.T) {
	server := NewInfoServer(&mockedConfig)

	req := &api.GetServiceInfoRequest{}

	resp, err := server.GetServiceInfo(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "test-id", resp.Id)
	assert.Equal(t, "localhost", resp.Hostname)
	assert.Equal(t, "2025-01-01T00:00:00Z", resp.CreatedAt)
}
