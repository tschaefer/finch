/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newDatabase(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	err = db.AutoMigrate(&Agent{})
	if err != nil {
		t.Fatal(err)
	}

	return db
}

func Test_CreateAgentReturnsAgent(t *testing.T) {
	db := newDatabase(t)
	m := New(db)
	assert.NotNil(t, m, "create model")

	data := &Agent{
		Active:       true,
		Hostname:     "test-agent",
		LastSeen:     nil,
		LogSources:   []string{"source1", "source2"},
		RegisteredAt: time.Now(),
		ResourceId:   "resource-123",
		Labels:       []string{"key=value", "env=prod"},
	}
	agent, err := m.CreateAgent(data)
	assert.NoError(t, err, "create agent")

	assert.Equal(t, data.Hostname, agent.Hostname, "agent hostname")
	assert.Equal(t, data.LogSources, agent.LogSources, "agent log sources")
	assert.Equal(t, data.ResourceId, agent.ResourceId, "agent resource ID")
	assert.Equal(t, data.Labels, agent.Labels, "agent labels")
	assert.Equal(t, data.Active, agent.Active, "agent active status")
	assert.NotZero(t, agent.RegisteredAt, "agent registered at")
	assert.Nil(t, agent.LastSeen, "agent last seen")
	assert.NotZero(t, agent.ID, "agent ID")
}

func Test_GetAgentReturnsAgent(t *testing.T) {
	db := newDatabase(t)
	m := New(db)
	assert.NotNil(t, m, "create model")

	data := &Agent{
		Active:       true,
		Hostname:     "test-agent",
		LastSeen:     nil,
		LogSources:   []string{"source1", "source2"},
		RegisteredAt: time.Now(),
		ResourceId:   "resource-123",
		Labels:       []string{"key=value", "env=prod"},
	}
	createdAgent, err := m.CreateAgent(data)
	assert.NoError(t, err, "create agent")

	toGet := &Agent{
		ID: createdAgent.ID,
	}
	retrievedAgent, err := m.GetAgent(toGet)
	assert.NoError(t, err, "get agent")

	assert.Equal(t, createdAgent.Hostname, retrievedAgent.Hostname, "agent hostname")
	assert.Equal(t, createdAgent.LogSources, retrievedAgent.LogSources, "agent log sources")
	assert.Equal(t, createdAgent.ResourceId, retrievedAgent.ResourceId, "agent resource ID")
	assert.Equal(t, createdAgent.Labels, retrievedAgent.Labels, "agent labels")
	assert.Equal(t, createdAgent.Active, retrievedAgent.Active, "agent active status")
	assert.NotZero(t, retrievedAgent.RegisteredAt, "agent registered at")
	assert.Nil(t, retrievedAgent.LastSeen, "agent last seen")
	assert.Equal(t, createdAgent.ID, retrievedAgent.ID, "agent ID")
}

func Test_GetAgentReturnsError_AgentNotFound(t *testing.T) {
	db := newDatabase(t)
	m := New(db)
	assert.NotNil(t, m, "create model")

	toGet := &Agent{
		ID: 9999,
	}
	_, err := m.GetAgent(toGet)
	assert.ErrorIs(t, err, ErrAgentNotFound, "get non-existent agent")
}

func Test_DeleteAgentRemovesAgent(t *testing.T) {
	db := newDatabase(t)
	m := New(db)
	assert.NotNil(t, m, "create model")

	data := &Agent{
		Active:       true,
		Hostname:     "test-agent",
		LastSeen:     nil,
		LogSources:   []string{"source1", "source2"},
		RegisteredAt: time.Now(),
		ResourceId:   "resource-123",
		Labels:       []string{"key=value", "env=prod"},
	}
	createdAgent, err := m.CreateAgent(data)
	assert.NoError(t, err, "create agent")

	err = m.DeleteAgent(createdAgent)
	assert.NoError(t, err, "delete agent")

	toGet := &Agent{
		ID: createdAgent.ID,
	}
	_, err = m.GetAgent(toGet)
	assert.ErrorIs(t, err, ErrAgentNotFound, "get deleted agent")
}

func Test_ListAgentsReturnsAllAgents(t *testing.T) {
	db := newDatabase(t)
	m := New(db)
	assert.NotNil(t, m, "create model")

	agentsData := []Agent{
		{
			Active:       true,
			Hostname:     "agent-1",
			LastSeen:     nil,
			LogSources:   []string{"source1"},
			RegisteredAt: time.Now(),
			ResourceId:   "resource-1",
			Labels:       []string{"key1=value1"},
		},
		{
			Active:       false,
			Hostname:     "agent-2",
			LastSeen:     nil,
			LogSources:   []string{"source2"},
			RegisteredAt: time.Now(),
			ResourceId:   "resource-2",
			Labels:       []string{"key2=value2"},
		},
	}

	for i := range agentsData {
		_, err := m.CreateAgent(&agentsData[i])
		assert.NoError(t, err, "create agent")
	}

	var agents []Agent
	listedAgents, err := m.ListAgents(&agents)
	assert.NoError(t, err, "list agents")

	assert.Len(t, *listedAgents, len(agentsData), "number of listed agents")
}

func Test_UpdateAgentModifiesAgent(t *testing.T) {
	db := newDatabase(t)
	m := New(db)
	assert.NotNil(t, m, "create model")

	data := &Agent{
		Active:       true,
		Hostname:     "test-agent",
		LastSeen:     nil,
		LogSources:   []string{"source1", "source2"},
		RegisteredAt: time.Now(),
		ResourceId:   "resource-123",
		Labels:       []string{"key=value", "env=prod"},
	}
	createdAgent, err := m.CreateAgent(data)
	assert.NoError(t, err, "create agent")

	createdAgent.LogSources = []string{"source3"}
	createdAgent.ResourceId = "resource-456"
	createdAgent.Labels = []string{"key=newvalue"}

	updatedAgent, err := m.UpdateAgent(createdAgent)
	assert.NoError(t, err, "update agent")

	assert.Equal(t, createdAgent.LogSources, updatedAgent.LogSources, "agent log sources")
	assert.Equal(t, createdAgent.ResourceId, updatedAgent.ResourceId, "agent resource ID")
	assert.Equal(t, createdAgent.Labels, updatedAgent.Labels, "agent labels")
}
