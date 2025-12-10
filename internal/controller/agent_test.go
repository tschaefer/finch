/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package controller

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func mockModel() model.Model {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		panic(err)
	}
	err = db.AutoMigrate(&model.Agent{})
	if err != nil {
		panic(err)
	}

	return model.New(db)
}

func Test_RegisterAgentReturnsError_InvalidParameters(t *testing.T) {
	model := mockModel()

	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	data := Agent{
		Hostname:       "",
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
}

func Test_RegisterAgentReturnsError_InvalidServiceSecret(t *testing.T) {
	model := mockModel()
	cfg := config.NewFromData(&config.Data{Secret: "invalid-secret"}, "")

	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	data := Agent{
		Hostname:       "test-host",
		Labels:         []string{"key=value"},
		LogSources:     []string{"journal://"},
		Metrics:        false,
		MetricsTargets: nil,
		Profiles:       false,
	}

	_, err := ctrl.RegisterAgent(&data)
	assert.Error(t, err, "register agent with invalid config secret")
}

func Test_RegisterAgentReturnsResourceId(t *testing.T) {
	model := mockModel()

	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	data := Agent{
		Hostname:       "test-host",
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
	model := mockModel()

	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	err := ctrl.DeregisterAgent("non-existent-rid")
	expected := "agent not found"
	assert.EqualError(t, err, expected, "deregister non-existent agent")
}

func Test_DeregisterAgentSucceeds(t *testing.T) {
	model := mockModel()

	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	data := Agent{
		Hostname:       "test-host",
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
	model := mockModel()

	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	_, err := ctrl.CreateAgentConfig("non-existent-rid")
	expected := "agent not found"
	assert.EqualError(t, err, expected, "create config for non-existent agent")
}

func Test_CreateAgentConfigReturnsConfig(t *testing.T) {
	model := mockModel()

	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	data := Agent{
		Hostname:       "test-host",
		Labels:         []string{"key=value", "statement"},
		LogSources:     []string{"file:///var/log/syslog", "journal://", "docker://"},
		Metrics:        false,
		MetricsTargets: []string{"http://localhost:9100/metrics"},
		Profiles:       false,
	}

	rid, err := ctrl.RegisterAgent(&data)
	assert.NoError(t, err, "register agent with valid parameters")

	cfg, err := ctrl.CreateAgentConfig(rid)
	assert.NoError(t, err, "create config for existing agent")
	assert.NotEmpty(t, cfg, "agent config not empty")
}

func Test_GetAgentReturnsError_AgentNotFound(t *testing.T) {
	model := mockModel()

	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	_, err := ctrl.GetAgent("non-existent-rid")
	expected := "agent not found"
	assert.EqualError(t, err, expected, "get non-existent agent")
}

func Test_GetAgentReturnsAgent(t *testing.T) {
	model := mockModel()

	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	data := Agent{
		Hostname:       "test-host",
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
	model := mockModel()
	config := config.NewFromData(&config.Data{Secret: "1suNCrW7sWlPbU+YCfdGQI7z3ZMo9Ru2GNV4h69QzaM="}, "")

	ctrl := New(model, config)
	assert.NotNil(t, ctrl, "create controller")

	agents, err := ctrl.ListAgents()
	assert.NoError(t, err, "list agents")
	assert.Len(t, agents, 0, "agent list")
}

func Test_ListAgentsReturnsAgents(t *testing.T) {
	model := mockModel()

	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	data := Agent{
		Hostname:       "test-host-1",
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
		Labels:         []string{"env=dev"},
		LogSources:     []string{"file:///var/log/syslog"},
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

func Test_RegisterAgentGeneratesCredentialsFile(t *testing.T) {
	model := mockModel()

	tmp := t.TempDir()
	confDir := filepath.Join(tmp, "traefik", "etc", "conf.d")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		t.Fatalf("failed to create conf dir: %v", err)
	}

	cfg := config.NewFromData(&config.Data{
		Secret: "1suNCrW7sWlPbU+YCfdGQI7z3ZMo9Ru2GNV4h69QzaM=",
	}, tmp)

	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	data := Agent{
		Hostname:       "test-host-credentials",
		Labels:         []string{"key=value"},
		LogSources:     []string{"file:///var/log/syslog"},
		Metrics:        false,
		MetricsTargets: nil,
		Profiles:       false,
	}

	rid, err := ctrl.RegisterAgent(&data)
	assert.NoError(t, err, "register agent with valid parameters")

	usersFile := filepath.Join(confDir, "loki-users.yaml")

	var content []byte
	found := false
	for range 50 {
		if _, err := os.Stat(usersFile); err == nil {
			content, err = os.ReadFile(usersFile)
			if err != nil {
				t.Fatalf("failed reading credentials file: %v", err)
			}
			found = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !found {
		t.Fatalf("credentials file not created: %s", usersFile)
	}

	stored, err := ctrl.GetAgent(rid)
	assert.NoError(t, err)
	assert.Contains(t, string(content), stored.Username, "credentials file should contain username")
	assert.Contains(t, string(content), stored.PasswordHash, "credentials file should contain password hash")
}

func Test_DeregisterAgentUpdatesCredentialsFile(t *testing.T) {
	model := mockModel()

	tmp := t.TempDir()
	confDir := filepath.Join(tmp, "traefik", "etc", "conf.d")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		t.Fatalf("failed to create conf dir: %v", err)
	}

	cfg := config.NewFromData(&config.Data{
		Secret: "1suNCrW7sWlPbU+YCfdGQI7z3ZMo9Ru2GNV4h69QzaM=",
	}, tmp)

	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	data := Agent{
		Hostname:       "test-host-deregister",
		Labels:         []string{"key=value"},
		LogSources:     []string{"file:///var/log/syslog"},
		Metrics:        false,
		MetricsTargets: nil,
		Profiles:       false,
	}

	rid, err := ctrl.RegisterAgent(&data)
	assert.NoError(t, err, "register agent with valid parameters")

	usersFile := filepath.Join(confDir, "loki-users.yaml")

	var content []byte
	found := false
	for range 50 {
		if _, err := os.Stat(usersFile); err == nil {
			content, err = os.ReadFile(usersFile)
			if err != nil {
				t.Fatalf("failed reading credentials file: %v", err)
			}
			found = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !found {
		t.Fatalf("credentials file not created: %s", usersFile)
	}

	stored, err := ctrl.GetAgent(rid)
	assert.NoError(t, err)
	assert.Contains(t, string(content), stored.Username, "credentials file should contain username before deregister")

	err = ctrl.DeregisterAgent(rid)
	assert.NoError(t, err)

	updated := false
	for range 50 {
		if _, err := os.Stat(usersFile); err == nil {
			content, err = os.ReadFile(usersFile)
			if err != nil {
				t.Fatalf("failed reading credentials file after deregister: %v", err)
			}
			if !strings.Contains(string(content), stored.Username) {
				updated = true
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !updated {
		t.Fatalf("credentials file not updated after deregister; still contains username: %s", stored.Username)
	}
}
