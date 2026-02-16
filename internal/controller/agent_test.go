/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package controller

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tschaefer/finch/internal/config"
	"github.com/tschaefer/finch/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var cfg = config.NewFromData(&config.Data{
	Secret: "1suNCrW7sWlPbU+YCfdGQI7z3ZMo9Ru2GNV4h69QzaM=",
	Id:     "test-id",
}, "")

func newModel(t *testing.T) *model.Model {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	err = db.AutoMigrate(&model.Agent{})
	if err != nil {
		t.Fatal(err)
	}

	return model.New(db)
}

func Test_RegisterAgentReturnsError_InvalidParameters(t *testing.T) {
	model := newModel(t)

	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	data := Agent{
		Hostname:       "",
		Node:           "unix",
		Labels:         nil,
		LogSources:     nil,
		Metrics:        false,
		MetricsTargets: nil,
		Profiles:       false,
	}

	_, err := ctrl.RegisterAgent(&data)
	expected := "hostname must not be empty"
	assert.EqualError(t, err, expected, "register agent with empty hostname")

	data.Hostname = "test-host"
	_, err = ctrl.RegisterAgent(&data)
	expected = "at least one log source must be specified"
	assert.EqualError(t, err, expected, "register agent with no log sources")

	data.LogSources = []string{"invalid://source"}
	_, err = ctrl.RegisterAgent(&data)
	expected = "no valid log source specified"
	assert.EqualError(t, err, expected, "register agent with invalid log source")

	data.Node = "invalid"
	_, err = ctrl.RegisterAgent(&data)
	expected = "node must be either 'windows' or 'unix'"
	assert.EqualError(t, err, expected, "register agent with invalid node type")
}

func Test_RegisterAgentReturnsResourceId(t *testing.T) {
	model := newModel(t)

	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	data := Agent{
		Hostname:       "test-host",
		Node:           "unix",
		Labels:         []string{"key=value", "env=prod"},
		LogSources:     []string{"file:///var/log/syslog"},
		Metrics:        false,
		MetricsTargets: nil,
		Profiles:       false,
	}

	rid, err := ctrl.RegisterAgent(&data)
	assert.NoError(t, err, "register agent with valid parameters")

	assert.NotEmpty(t, rid, "resource ID not empty")
	parts := strings.Split(rid, ":")
	assert.Len(t, parts, 5, "resource ID format")
	assert.Equal(t, "rid", parts[0], "resource ID prefix")
	assert.Equal(t, "finch", parts[1], "resource ID service")
	assert.Equal(t, "test-id", parts[2], "resource ID identifier")
	assert.Equal(t, "agent", parts[3], "resource ID type")
	assert.Len(t, parts[4], 36, "resource ID UUID length")

}

func Test_DeregisterAgentReturnsError_AgentNotFound(t *testing.T) {
	model := newModel(t)

	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	err := ctrl.DeregisterAgent("non-existent-rid")
	expected := "agent not found"
	assert.EqualError(t, err, expected, "deregister non-existent agent")
}

func Test_DeregisterAgentSucceeds(t *testing.T) {
	model := newModel(t)

	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	data := Agent{
		Hostname:       "test-host",
		Node:           "unix",
		Labels:         []string{"key=value"},
		LogSources:     []string{"file:///var/log/syslog"},
		Metrics:        false,
		MetricsTargets: nil,
		Profiles:       false,
	}

	rid, err := ctrl.RegisterAgent(&data)
	assert.NoError(t, err, "register agent with valid parameters")

	err = ctrl.DeregisterAgent(rid)
	assert.NoError(t, err, "deregister existing agent")
}

func Test_CreateAgentConfigReturnsError_AgentNotFound(t *testing.T) {
	model := newModel(t)

	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	_, err := ctrl.CreateAgentConfig("non-existent-rid")
	expected := "agent not found"
	assert.EqualError(t, err, expected, "create config for non-existent agent")
}

func Test_CreateAgentConfigReturnsConfig(t *testing.T) {
	model := newModel(t)

	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	data := Agent{
		Hostname:       "test-host",
		Node:           "unix",
		Labels:         []string{"key=value", "statement"},
		LogSources:     []string{"file:///var/log/syslog", "journal://", "docker://"},
		Metrics:        false,
		MetricsTargets: []string{"http://localhost:9100/metrics"},
		Profiles:       false,
	}

	rid, err := ctrl.RegisterAgent(&data)
	assert.NoError(t, err, "register agent with valid parameters")

	agentConfig, err := ctrl.CreateAgentConfig(rid)
	assert.NoError(t, err, "create config for existing agent")
	assert.NotEmpty(t, agentConfig, "agent config not empty")
}

func Test_GetAgentReturnsError_AgentNotFound(t *testing.T) {
	model := newModel(t)

	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	_, err := ctrl.GetAgent("non-existent-rid")
	expected := "agent not found"
	assert.EqualError(t, err, expected, "get non-existent agent")
}

func Test_GetAgentReturnsAgent(t *testing.T) {
	model := newModel(t)

	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	data := Agent{
		Hostname:       "test-host",
		Node:           "unix",
		Labels:         []string{"key=value"},
		LogSources:     []string{"file:///var/log/syslog"},
		Metrics:        false,
		MetricsTargets: nil,
		Profiles:       false,
	}

	rid, err := ctrl.RegisterAgent(&data)
	assert.NoError(t, err, "register agent with valid parameters")

	agent, err := ctrl.GetAgent(rid)
	assert.NoError(t, err, "get existing agent")
	assert.Equal(t, "test-host", agent.Hostname, "agent hostname")
}

func Test_ListAgentsReturnsEmptyList_AgentsNotFound(t *testing.T) {
	model := newModel(t)

	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	agents, err := ctrl.ListAgents()
	assert.NoError(t, err, "list agents")
	assert.Len(t, agents, 0, "agent list")
}

func Test_ListAgentsReturnsAgents(t *testing.T) {
	model := newModel(t)

	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	data := Agent{
		Hostname:       "test-host-1",
		Node:           "unix",
		Labels:         []string{"key=value"},
		LogSources:     []string{"file:///var/log/syslog"},
		Metrics:        false,
		MetricsTargets: nil,
		Profiles:       false,
	}

	_, err := ctrl.RegisterAgent(&data)
	assert.NoError(t, err, "register first agent")

	data = Agent{
		Hostname:       "test-host-2",
		Node:           "windows",
		Labels:         []string{"env=dev"},
		LogSources:     []string{"event://System"},
		Metrics:        false,
		MetricsTargets: nil,
		Profiles:       false,
	}

	_, err = ctrl.RegisterAgent(&data)
	assert.NoError(t, err, "register second agent")

	agents, err := ctrl.ListAgents()
	assert.NoError(t, err, "list agents")
	assert.Len(t, agents, 2, "agent list")
	assert.Equal(t, "test-host-1", agents[0]["hostname"], "first agent hostname")
	assert.Equal(t, "test-host-2", agents[1]["hostname"], "second agent hostname")
}

func Test_UpdateAgentReturnsError_AgentNotFound(t *testing.T) {
	model := newModel(t)

	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	data := Agent{
		Hostname:       "non-existent-rid",
		Node:           "unix",
		Labels:         []string{"key=value"},
		LogSources:     []string{"file:///var/log/syslog"},
		Metrics:        false,
		MetricsTargets: nil,
		Profiles:       false,
	}

	err := ctrl.UpdateAgent("non-existent-rid", &data)
	expected := "agent not found"
	assert.EqualError(t, err, expected, "update non-existent agent")
}

func Test_UpdateAgentSucceeds(t *testing.T) {
	model := newModel(t)

	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	data := Agent{
		Hostname:       "test-host-update",
		Node:           "unix",
		Labels:         []string{"key=value"},
		LogSources:     []string{"file:///var/log/syslog"},
		Metrics:        false,
		MetricsTargets: nil,
		Profiles:       false,
	}

	rid, err := ctrl.RegisterAgent(&data)
	assert.NoError(t, err, "register agent with valid parameters")

	updatedData := Agent{
		Labels:         []string{"env=staging"},
		Node:           "unix",
		LogSources:     []string{"journal://"},
		Metrics:        true,
		MetricsTargets: []string{"http://localhost:9100/metrics"},
		Profiles:       true,
	}

	err = ctrl.UpdateAgent(rid, &updatedData)
	assert.NoError(t, err, "update existing agent")

	agent, err := ctrl.GetAgent(rid)
	assert.NoError(t, err, "get updated agent")
	assert.Equal(t, []string{"env=staging"}, agent.Labels, "updated labels")
	assert.Equal(t, []string{"journal:"}, agent.LogSources, "updated log sources")
	assert.True(t, agent.Metrics, "updated metrics flag")
	assert.Equal(t, []string{"http://localhost:9100/metrics"}, agent.MetricsTargets, "updated metrics targets")
	assert.True(t, agent.Profiles, "updated profiles flag")
}
