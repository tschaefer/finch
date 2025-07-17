/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type Agent struct {
	ID           uint       `gorm:"primarykey" json:"-"`
	CreatedAt    time.Time  `json:"-"`
	UpdatedAt    time.Time  `json:"-"`
	Active       bool       `gorm:"not null;default:true" json:"active"`
	Hostname     string     `gorm:"not null;unique" json:"hostname"`
	LastSeen     *time.Time `gorm:"default:NULL" json:"last_seen"`
	LogSources   []string   `gorm:"not null;serializer:json" json:"log_sources"`
	Password     string     `gorm:"not null" json:"-"`
	PasswordHash string     `gorm:"not null" json:"-"`
	RegisteredAt time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"registered_at"`
	ResourceId   string     `gorm:"not null;unique;uniqueIndex:uidx_agents_resource_id" json:"resource_id"`
	Tags         []string   `gorm:"serializer:json" json:"tags"`
	Username     string     `gorm:"not null" json:"-"`
}

type ModelAgent interface {
	CreateAgent(agent *Agent) (*Agent, error)
	DeleteAgent(agent *Agent) error
	GetAgent(agent *Agent) (*Agent, error)
	ListAgents(agent *[]Agent) (*[]Agent, error)
}

var (
	ErrAgentNotFound = errors.New("agent not found")
)

func (m *model) CreateAgent(agent *Agent) (*Agent, error) {
	if err := m.db.Create(agent).Error; err != nil {
		return nil, err
	}

	return agent, nil
}

func (m *model) DeleteAgent(agent *Agent) error {
	return m.db.Delete(agent).Error
}

func (m *model) GetAgent(agent *Agent) (*Agent, error) {
	if err := m.db.Where(agent).First(agent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAgentNotFound
		}
		return nil, err
	}

	return agent, nil
}

func (m *model) ListAgents(agents *[]Agent) (*[]Agent, error) {
	if err := m.db.Find(agents).Error; err != nil {
		return nil, err
	}

	return agents, nil
}
