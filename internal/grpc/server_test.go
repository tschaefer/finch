package grpc

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tschaefer/finch/api"
	"github.com/tschaefer/finch/internal/config"
	"github.com/tschaefer/finch/internal/controller"
	"github.com/tschaefer/finch/internal/database"
	"github.com/tschaefer/finch/internal/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var testServerCfg = config.NewFromData(&config.Data{
	Id:        "test-id",
	Hostname:  "localhost",
	CreatedAt: "2025-01-01T00:00:00Z",
	Database:  "sqlite://:memory:",
	Credentials: config.Credentials{
		Username: "admin",
		Password: "password",
	},
	Secret: "gpFb8WTh5iELimbX3YfuvRYRh2Z2PHa8Lmoog0a25QQ=",
}, "")

func newController(t *testing.T) *controller.Controller {
	db, err := database.New(testServerCfg)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Migrate()
	if err != nil {
		t.Fatal(err)
	}

	model := model.New(db.Connection())

	return controller.New(model, testServerCfg)
}

func TestRegisterAgentReturnsResourceId(t *testing.T) {
	server := NewAgentServer(newController(t), testServerCfg)

	req := &api.RegisterAgentRequest{
		Hostname:   "test-host",
		LogSources: []string{"journal://"},
		Metrics:    true,
	}

	resp, err := server.RegisterAgent(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Contains(t, resp.Rid, "rid:finch:test-id:agent:")
}

func TestRegisterAgentReturnsError_AgentAlreadyExists(t *testing.T) {
	server := NewAgentServer(newController(t), testServerCfg)

	register := func() (*api.RegisterAgentResponse, error) {
		req := &api.RegisterAgentRequest{
			Hostname:   "existing",
			LogSources: []string{"journal://"},
		}
		resp, err := server.RegisterAgent(context.Background(), req)
		return resp, err
	}
	_, err := register()
	assert.NoError(t, err)

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
	server := NewAgentServer(newController(t), testServerCfg)

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
	server := NewAgentServer(newController(t), testServerCfg)

	register := func() (*api.RegisterAgentResponse, error) {
		req := &api.RegisterAgentRequest{
			Hostname:   "existing",
			LogSources: []string{"journal://"},
		}
		resp, err := server.RegisterAgent(context.Background(), req)
		return resp, err
	}
	agent, err := register()
	assert.NoError(t, err)

	req := &api.DeregisterAgentRequest{
		Rid: agent.Rid,
	}

	resp, err := server.DeregisterAgent(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestDeregisterAgentReturnsError_InvalidArguments(t *testing.T) {
	server := NewAgentServer(newController(t), testServerCfg)

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
	server := NewAgentServer(newController(t), testServerCfg)

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
	server := NewAgentServer(newController(t), testServerCfg)

	register := func() (*api.RegisterAgentResponse, error) {
		req := &api.RegisterAgentRequest{
			Hostname:   "node1",
			LogSources: []string{"journal://"},
		}
		resp, err := server.RegisterAgent(context.Background(), req)
		return resp, err
	}
	agent, err := register()
	assert.NoError(t, err)

	req := &api.GetAgentRequest{
		Rid: agent.Rid,
	}

	resp, err := server.GetAgent(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, agent.Rid, resp.ResourceId)
	assert.Equal(t, "node1", resp.Hostname)
}

func TestGetAgentReturnsError_InvalidArguments(t *testing.T) {
	server := NewAgentServer(newController(t), testServerCfg)

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
	server := NewAgentServer(newController(t), testServerCfg)

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
	server := NewAgentServer(newController(t), testServerCfg)

	register := func(n int) error {
		for i := range n {
			req := &api.RegisterAgentRequest{
				Hostname:   fmt.Sprintf("node%d", i),
				LogSources: []string{"journal://"},
			}
			_, err := server.RegisterAgent(context.Background(), req)
			if err != nil {
				return err
			}
		}
		return nil
	}
	err := register(2)
	if err != nil {
		t.Fatal(err)
	}

	req := &api.ListAgentsRequest{}

	resp, err := server.ListAgents(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Agents, 2)
}

func TestGetAgentConfigReturnsConfig(t *testing.T) {
	server := NewAgentServer(newController(t), testServerCfg)

	register := func() (*api.RegisterAgentResponse, error) {
		req := &api.RegisterAgentRequest{
			Hostname:   "node1",
			LogSources: []string{"journal://"},
		}
		resp, err := server.RegisterAgent(context.Background(), req)
		return resp, err
	}
	agent, err := register()
	assert.NoError(t, err)

	req := &api.GetAgentConfigRequest{
		Rid: agent.Rid,
	}

	resp, err := server.GetAgentConfig(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Contains(t, string(resp.Config), agent.Rid)
}

func TestGetAgentConfigReturnsError_InvalidArguments(t *testing.T) {
	server := NewAgentServer(newController(t), testServerCfg)

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
	server := NewAgentServer(newController(t), testServerCfg)

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
