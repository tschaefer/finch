package model

import (
	"testing"
	"time"

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

func Test_CreateAgentSucceeds(t *testing.T) {
	db := mockDatabase()
	m := New(db)

	agent := &Agent{
		Active:       true,
		Hostname:     "test-agent",
		LastSeen:     nil,
		LogSources:   []string{"source1", "source2"},
		Password:     "password",
		PasswordHash: "hashed_password",
		RegisteredAt: time.Now(),
		ResourceId:   "resource-123",
		Tags:         []string{"tag1", "tag2"},
		Username:     "testuser",
	}
	createdAgent, err := m.CreateAgent(agent)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if createdAgent.Hostname != agent.Hostname {
		t.Errorf("expected hostname %s, got %s", agent.Hostname, createdAgent.Hostname)
	}

	if len(createdAgent.LogSources) != len(agent.LogSources) {
		t.Errorf("expected %d log sources, got %d", len(agent.LogSources), len(createdAgent.LogSources))
	}

	if createdAgent.ResourceId != agent.ResourceId {
		t.Errorf("expected resource ID %s, got %s", agent.ResourceId, createdAgent.ResourceId)
	}

	if createdAgent.Username != agent.Username {
		t.Errorf("expected username %s, got %s", agent.Username, createdAgent.Username)
	}

	if createdAgent.Active != agent.Active {
		t.Errorf("expected active status %v, got %v", agent.Active, createdAgent.Active)
	}

	if createdAgent.Password != agent.Password {
		t.Errorf("expected password %s, got %s", agent.Password, createdAgent.Password)
	}

	if createdAgent.PasswordHash != agent.PasswordHash {
		t.Errorf("expected password hash %s, got %s", agent.PasswordHash, createdAgent.PasswordHash)
	}

	if createdAgent.RegisteredAt.IsZero() {
		t.Error("expected registered_at to be set, got zero value")
	}

	if len(createdAgent.Tags) != len(agent.Tags) {
		t.Errorf("expected %d tags, got %d", len(agent.Tags), len(createdAgent.Tags))
	}

	if createdAgent.LastSeen != nil {
		t.Error("expected last_seen to be nil, got non-nil value")
	}

	if createdAgent.ID == 0 {
		t.Error("expected agent ID to be set, got zero value")
	}
}
