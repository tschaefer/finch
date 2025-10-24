/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package controller

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tschaefer/finch/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

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
	id:        "test-id",
	createdAt: "2025-01-01T00:00:00Z",
	library:   "test-library",
	secret:    "1suNCrW7sWlPbU+YCfdGQI7z3ZMo9Ru2GNV4h69QzaM=",
	username:  "test-user",
	password:  "test-password",
}

func Test_RegisterAgentReturnsError_BadParameters(t *testing.T) {
	model := mockModel()

	ctrl := New(model, &mockedConfig)
	assert.NotNil(t, ctrl, "create controller")

	_, err := ctrl.RegisterAgent("", nil, nil, false, false)
	expected := "hostname must not be empty"
	assert.EqualError(t, err, expected, "register agent with empty hostname")

	_, err = ctrl.RegisterAgent("test-host", nil, nil, false, false)
	expected = "at least one log source must be specified"
	assert.EqualError(t, err, expected, "register agent with no log sources")

	_, err = ctrl.RegisterAgent("test-host", nil, []string{"invalid://source"}, false, false)
	expected = "no valid log source specified"
	assert.EqualError(t, err, expected, "register agent with invalid log source")
}

func Test_RegisterAgentReturnsError_InvalidSecret(t *testing.T) {
	model := mockModel()
	config := mockedConfig
	config.secret = "invalid-secret"

	ctrl := New(model, &config)
	assert.NotNil(t, ctrl, "create controller")

	_, err := ctrl.RegisterAgent("test-host", []string{"tag1"}, []string{"journal://"}, false, false)
	assert.Error(t, err, "register agent with invalid config secret")
}

func Test_RegisterAgentReturnsResourceId(t *testing.T) {
	model := mockModel()

	ctrl := New(model, &mockedConfig)
	assert.NotNil(t, ctrl, "create controller")

	rid, err := ctrl.RegisterAgent("test-host", []string{"tag1", "tag2"}, []string{"file:///var/log/syslog"}, false, false)
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

func Test_DeregisterAgentReturnsError_NotFound(t *testing.T) {
	model := mockModel()

	ctrl := New(model, &mockedConfig)
	assert.NotNil(t, ctrl, "create controller")

	err := ctrl.DeregisterAgent("non-existent-rid")
	expected := "agent not found"
	assert.EqualError(t, err, expected, "deregister non-existent agent")
}

func Test_DeregisterAgentReturnsNil(t *testing.T) {
	model := mockModel()

	ctrl := New(model, &mockedConfig)
	assert.NotNil(t, ctrl, "create controller")

	rid, err := ctrl.RegisterAgent("test-host", []string{"tag1"}, []string{"file:///var/log/syslog"}, false, false)
	assert.NoError(t, err, "register agent with valid parameters")

	err = ctrl.DeregisterAgent(rid)
	assert.NoError(t, err, "deregister existing agent")
}

func Test_CreateAgentConfigReturnsError_NotFound(t *testing.T) {
	model := mockModel()

	ctrl := New(model, &mockedConfig)
	assert.NotNil(t, ctrl, "create controller")

	_, err := ctrl.CreateAgentConfig("non-existent-rid")
	expected := "agent not found"
	assert.EqualError(t, err, expected, "create config for non-existent agent")
}

func Test_CreateAgentConfigReturnsConfig(t *testing.T) {
	model := mockModel()

	ctrl := New(model, &mockedConfig)
	assert.NotNil(t, ctrl, "create controller")

	rid, err := ctrl.RegisterAgent("test-host", []string{"tag1"}, []string{"file:///var/log/syslog"}, false, false)
	assert.NoError(t, err, "register agent with valid parameters")

	config, err := ctrl.CreateAgentConfig(rid)
	assert.NoError(t, err, "create config for existing agent")
	assert.NotEmpty(t, config, "agent config not empty")
}

func Test_GetAgentReturnsError_NotFound(t *testing.T) {
	model := mockModel()

	ctrl := New(model, &mockedConfig)
	assert.NotNil(t, ctrl, "create controller")

	_, err := ctrl.GetAgent("non-existent-rid")
	expected := "agent not found"
	assert.EqualError(t, err, expected, "get non-existent agent")
}

func Test_GetAgentReturnsAgent(t *testing.T) {
	model := mockModel()

	ctrl := New(model, &mockedConfig)
	assert.NotNil(t, ctrl, "create controller")

	rid, err := ctrl.RegisterAgent("test-host", []string{"tag1"}, []string{"file:///var/log/syslog"}, false, false)
	assert.NoError(t, err, "register agent with valid parameters")

	agent, err := ctrl.GetAgent(rid)
	assert.NoError(t, err, "get existing agent")
	assert.Equal(t, "test-host", agent.Hostname, "agent hostname")
}

func Test_ListAgentsReturnsEmptyList(t *testing.T) {
	model := mockModel()

	ctrl := New(model, &mockedConfig)
	assert.NotNil(t, ctrl, "create controller")

	agents, err := ctrl.ListAgents()
	assert.NoError(t, err, "list agents")
	assert.Len(t, agents, 0, "agent list")
}

func Test_ListAgentsReturnsAgents(t *testing.T) {
	model := mockModel()

	ctrl := New(model, &mockedConfig)
	assert.NotNil(t, ctrl, "create controller")

	_, err := ctrl.RegisterAgent("test-host-1", []string{"tag1"}, []string{"file:///var/log/syslog"}, false, false)
	assert.NoError(t, err, "register first agent")

	_, err = ctrl.RegisterAgent("test-host-2", []string{"tag2"}, []string{"file:///var/log/syslog"}, false, false)
	assert.NoError(t, err, "register second agent")

	agents, err := ctrl.ListAgents()
	assert.NoError(t, err, "list agents")
	assert.Len(t, agents, 2, "agent list")
	assert.Equal(t, "test-host-1", agents[0]["hostname"], "first agent hostname")
	assert.Equal(t, "test-host-2", agents[1]["hostname"], "second agent hostname")
}
