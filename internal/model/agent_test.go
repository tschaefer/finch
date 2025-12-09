package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func mockDatabase() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	err = db.AutoMigrate(&Agent{})
	if err != nil {
		panic(err)
	}

	return db
}

func Test_CreateAgentReturnsAgent(t *testing.T) {
	db := mockDatabase()
	m := New(db)
	assert.NotNil(t, m, "create model")

	data := &Agent{
		Active:       true,
		Hostname:     "test-agent",
		LastSeen:     nil,
		LogSources:   []string{"source1", "source2"},
		Password:     "password",
		PasswordHash: "hashed_password",
		RegisteredAt: time.Now(),
		ResourceId:   "resource-123",
		Labels:       []string{"key=value", "env=prod"},
		Username:     "testuser",
	}
	agent, err := m.CreateAgent(data)
	assert.NoError(t, err, "create agent")

	assert.Equal(t, data.Hostname, agent.Hostname, "agent hostname")
	assert.Equal(t, data.LogSources, agent.LogSources, "agent log sources")
	assert.Equal(t, data.ResourceId, agent.ResourceId, "agent resource ID")
	assert.Equal(t, data.Labels, agent.Labels, "agent labels")
	assert.Equal(t, data.Username, agent.Username, "agent username")
	assert.Equal(t, data.Active, agent.Active, "agent active status")
	assert.Equal(t, data.Password, agent.Password, "agent password")
	assert.Equal(t, data.PasswordHash, agent.PasswordHash, "agent password hash")
	assert.NotZero(t, agent.RegisteredAt, "agent registered at")
	assert.Nil(t, agent.LastSeen, "agent last seen")
	assert.NotZero(t, agent.ID, "agent ID")
}
