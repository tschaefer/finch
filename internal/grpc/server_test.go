package grpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tschaefer/finch/api"
	"github.com/tschaefer/finch/internal/config"
	"github.com/tschaefer/finch/internal/controller"
	"github.com/tschaefer/finch/internal/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var testServerCfg = config.NewFromData(&config.Data{
	Id:        "test-id",
	Hostname:  "localhost",
	CreatedAt: "2025-01-01T00:00:00Z",
}, "")

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

func TestRegisterAgentReturnsResourceId(t *testing.T) {
	server := NewAgentServer(&mockController{}, testServerCfg)

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

func TestRegisterAgentReturnsError_AgentAlreadyExists(t *testing.T) {
	server := NewAgentServer(&mockController{}, testServerCfg)

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

func TestRegisterAgentReturnsError_InvalidArguments(t *testing.T) {
	server := NewAgentServer(&mockController{}, testServerCfg)

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

func TestDeregisterAgentSucceeds(t *testing.T) {
	server := NewAgentServer(&mockController{}, testServerCfg)

	req := &api.DeregisterAgentRequest{
		Rid: "rid:12345",
	}

	resp, err := server.DeregisterAgent(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestDeregisterAgentReturnsError_InvalidArguments(t *testing.T) {
	server := NewAgentServer(&mockController{}, testServerCfg)

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

func TestDeregisterAgentReturnsError_AgentNotFound(t *testing.T) {
	server := NewAgentServer(&mockController{}, testServerCfg)

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

func TestGetAgentReturnsAgent(t *testing.T) {
	server := NewAgentServer(&mockController{}, testServerCfg)

	req := &api.GetAgentRequest{
		Rid: "rid:12345",
	}

	resp, err := server.GetAgent(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "rid:12345", resp.ResourceId)
	assert.Equal(t, "node1", resp.Hostname)
}

func TestGetAgentReturnsError_InvalidArguments(t *testing.T) {
	server := NewAgentServer(&mockController{}, testServerCfg)

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

func TestGetAgentReturnsError_AgentNotFound(t *testing.T) {
	server := NewAgentServer(&mockController{}, testServerCfg)

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

func TestListAgentsReturnsAgentList(t *testing.T) {
	server := NewAgentServer(&mockController{}, testServerCfg)

	req := &api.ListAgentsRequest{}

	resp, err := server.ListAgents(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Agents, 2)
	assert.Equal(t, "rid:12345", resp.Agents[0].Rid)
	assert.Equal(t, "node1", resp.Agents[0].Hostname)
}

func TestGetAgentConfigReturnsConfig(t *testing.T) {
	server := NewAgentServer(&mockController{}, testServerCfg)

	req := &api.GetAgentConfigRequest{
		Rid: "rid:12345",
	}

	resp, err := server.GetAgentConfig(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, []byte("config content"), resp.Config)
}

func TestGetAgentConfigReturnsError_InvalidArguments(t *testing.T) {
	server := NewAgentServer(&mockController{}, testServerCfg)

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

func TestGetAgentConfigReturnsError_AgentNotFound(t *testing.T) {
	server := NewAgentServer(&mockController{}, testServerCfg)

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

func TestGetServiceInfoReturnsInfo(t *testing.T) {
	server := NewInfoServer(testServerCfg)

	req := &api.GetServiceInfoRequest{}

	resp, err := server.GetServiceInfo(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "test-id", resp.Id)
	assert.Equal(t, "localhost", resp.Hostname)
	assert.Equal(t, "2025-01-01T00:00:00Z", resp.CreatedAt)
}
