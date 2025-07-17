/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package controller

import (
	"strings"
	"testing"

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

func Test_RegisterAgentReturnsError_BadParameters(t *testing.T) {
	model := mockModel()

	ctrl := New(model, &mockedConfig)
	if ctrl == nil {
		t.Fatal("Expected controller to be created, but got nil")
	}

	_, err := ctrl.RegisterAgent("", nil, nil)
	if err == nil {
		t.Error("Expected error when registering agent with empty hostname, but got nil")
	}
	expected := "hostname must not be empty"
	if err.Error() != expected {
		t.Errorf("Expected error message '%s', but got '%s'", expected, err.Error())
	}

	_, err = ctrl.RegisterAgent("test-host", nil, nil)
	if err == nil {
		t.Error("Expected error when registering agent with no log sources, but got nil")
	}
	expected = "at least one log source must be specified"
	if err.Error() != expected {
		t.Errorf("Expected error message '%s', but got '%s'", expected, err.Error())
	}

	_, err = ctrl.RegisterAgent("test-host", nil, []string{"invalid://source"})
	if err == nil {
		t.Error("Expected error when registering agent with invalid log source, but got nil")
	}
	expected = "no valid log source specified"
	if err.Error() != expected {
		t.Errorf("Expected error message '%s', but got '%s'", expected, err.Error())
	}
}

func Test_RegisterAgentReturnsError_InvalidSecret(t *testing.T) {
	model := mockModel()
	config := mockedConfig
	config.secret = "invalid-secret"

	ctrl := New(model, &config)
	if ctrl == nil {
		t.Fatal("Expected controller to be created, but got nil")
	}

	_, err := ctrl.RegisterAgent("test-host", []string{"tag1"}, []string{"journal://"})
	if err == nil {
		t.Error("Expected error when registering agent with invalid secret, but got nil")
	}
}

func Test_RegisterAgentReturnsResourceId(t *testing.T) {
	model := mockModel()

	ctrl := New(model, &mockedConfig)
	if ctrl == nil {
		t.Fatal("Expected controller to be created, but got nil")
	}

	rid, err := ctrl.RegisterAgent("test-host", []string{"tag1", "tag2"}, []string{"file:///var/log/syslog"})
	if err != nil {
		t.Fatalf("Expected no error when registering agent, but got: %v", err)
	}
	if rid == "" {
		t.Error("Expected non-empty resource ID, but got empty string")
	}

	if !strings.HasPrefix(rid, "rid:") {
		t.Errorf("Expected resource ID to start with 'rid', but got '%s'", rid)
	}
}

func Test_DeregisterAgentReturnsError_NotFound(t *testing.T) {
	model := mockModel()

	ctrl := New(model, &mockedConfig)
	if ctrl == nil {
		t.Fatal("Expected controller to be created, but got nil")
	}

	err := ctrl.DeregisterAgent("non-existent-rid")
	if err == nil {
		t.Error("Expected error when deregistering non-existent agent, but got nil")
	}
	expected := "agent not found"
	if err.Error() != expected {
		t.Errorf("Expected error message '%s', but got '%s'", expected, err.Error())
	}
}

func Test_DeregisterAgentReturnsNil(t *testing.T) {
	model := mockModel()

	ctrl := New(model, &mockedConfig)
	if ctrl == nil {
		t.Fatal("Expected controller to be created, but got nil")
	}

	rid, err := ctrl.RegisterAgent("test-host", []string{"tag1"}, []string{"file:///var/log/syslog"})
	if err != nil {
		t.Fatalf("Expected no error when registering agent, but got: %v", err)
	}

	err = ctrl.DeregisterAgent(rid)
	if err != nil {
		t.Fatalf("Expected no error when deregistering agent, but got: %v", err)
	}
}

func Test_CreateAgentConfigReturnsError_NotFound(t *testing.T) {
	model := mockModel()

	ctrl := New(model, &mockedConfig)
	if ctrl == nil {
		t.Fatal("Expected controller to be created, but got nil")
	}

	_, err := ctrl.CreateAgentConfig("non-existent-rid")
	if err == nil {
		t.Error("Expected error when creating agent config for non-existent agent, but got nil")
	}
	expected := "agent not found"
	if err.Error() != expected {
		t.Errorf("Expected error message '%s', but got '%s'", expected, err.Error())
	}
}

func Test_CreateAgentConfigReturnsConfig(t *testing.T) {
	model := mockModel()

	ctrl := New(model, &mockedConfig)
	if ctrl == nil {
		t.Fatal("Expected controller to be created, but got nil")
	}

	rid, err := ctrl.RegisterAgent("test-host", []string{"tag1"}, []string{"file:///var/log/syslog"})
	if err != nil {
		t.Fatalf("Expected no error when registering agent, but got: %v", err)
	}

	config, err := ctrl.CreateAgentConfig(rid)
	if err != nil {
		t.Fatalf("Expected no error when creating agent config, but got: %v", err)
	}
	if len(config) == 0 {
		t.Error("Expected non-empty agent config, but got empty byte slice")
	}
}

func Test_GetAgentReturnsError_NotFound(t *testing.T) {
	model := mockModel()

	ctrl := New(model, &mockedConfig)
	if ctrl == nil {
		t.Fatal("Expected controller to be created, but got nil")
	}

	_, err := ctrl.GetAgent("non-existent-rid")
	if err == nil {
		t.Error("Expected error when getting non-existent agent, but got nil")
	}
	expected := "agent not found"
	if err.Error() != expected {
		t.Errorf("Expected error message '%s', but got '%s'", expected, err.Error())
	}
}

func Test_GetAgentReturnsAgent(t *testing.T) {
	model := mockModel()

	ctrl := New(model, &mockedConfig)
	if ctrl == nil {
		t.Fatal("Expected controller to be created, but got nil")
	}

	rid, err := ctrl.RegisterAgent("test-host", []string{"tag1"}, []string{"file:///var/log/syslog"})
	if err != nil {
		t.Fatalf("Expected no error when registering agent, but got: %v", err)
	}

	agent, err := ctrl.GetAgent(rid)
	if err != nil {
		t.Fatalf("Expected no error when getting agent, but got: %v", err)
	}
	if agent.Hostname != "test-host" {
		t.Errorf("Expected agent hostname 'test-host', but got '%s'", agent.Hostname)
	}
}

func Test_ListAgentsReturnsEmptyList(t *testing.T) {
	model := mockModel()

	ctrl := New(model, &mockedConfig)
	if ctrl == nil {
		t.Fatal("Expected controller to be created, but got nil")
	}

	agents, err := ctrl.ListAgents()
	if err != nil {
		t.Fatalf("Expected no error when listing agents, but got: %v", err)
	}
	if len(agents) != 0 {
		t.Errorf("Expected empty agent list, but got %d agents", len(agents))
	}
}

func Test_ListAgentsReturnsAgents(t *testing.T) {
	model := mockModel()

	ctrl := New(model, &mockedConfig)
	if ctrl == nil {
		t.Fatal("Expected controller to be created, but got nil")
	}

	_, err := ctrl.RegisterAgent("test-host-1", []string{"tag1"}, []string{"file:///var/log/syslog"})
	if err != nil {
		t.Fatalf("Expected no error when registering agent, but got: %v", err)
	}
	_, err = ctrl.RegisterAgent("test-host-2", []string{"tag2"}, []string{"file:///var/log/syslog"})
	if err != nil {
		t.Fatalf("Expected no error when registering agent, but got: %v", err)
	}

	agents, err := ctrl.ListAgents()
	if err != nil {
		t.Fatalf("Expected no error when listing agents, but got: %v", err)
	}
	if len(agents) != 2 {
		t.Errorf("Expected 2 agents, but got %d agents", len(agents))
	}
	if agents[0]["hostname"] != "test-host-1" && agents[1]["hostname"] != "test-host-2" {
		t.Errorf("Expected agents with hostnames 'test-host-1' and 'test-host-2', but got '%s' and '%s'", agents[0]["hostname"], agents[1]["hostname"])
	}
}

// TODO: Test generateCredentialsFile
