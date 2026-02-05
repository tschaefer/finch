/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package model

import "gorm.io/gorm"

type AgentEvent struct {
	Type string
}

type Model struct {
	db               *gorm.DB
	agentEventChan   chan AgentEvent
	agentSubscribers []chan AgentEvent
}

func New(db *gorm.DB) *Model {
	return &Model{
		db:               db,
		agentEventChan:   make(chan AgentEvent, 100),
		agentSubscribers: make([]chan AgentEvent, 0),
	}
}

func (m *Model) SubscribeAgentEvents() <-chan AgentEvent {
	ch := make(chan AgentEvent, 10)
	m.agentSubscribers = append(m.agentSubscribers, ch)
	return ch
}

func (m *Model) notifyAgentEvent(eventType string) {
	event := AgentEvent{Type: eventType}
	select {
	case m.agentEventChan <- event:
	default:
	}

	for _, sub := range m.agentSubscribers {
		select {
		case sub <- event:
		default:
		}
	}
}
