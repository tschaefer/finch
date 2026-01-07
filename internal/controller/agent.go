/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package controller

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"text/template"

	"github.com/tschaefer/finch/internal/aes"
	"github.com/tschaefer/finch/internal/model"
)

var (
	ErrAgentNotFound      = errors.New("agent not found")
	ErrAgentAlreadyExists = errors.New("agent already exists")
)

type Agent struct {
	Hostname       string   `json:"hostname"`
	Labels         []string `json:"labels"`
	LogSources     []string `json:"log_sources"`
	Metrics        bool     `json:"metrics"`
	MetricsTargets []string `json:"metrics_targets"`
	Profiles       bool     `json:"profiles"`
}

func (c *Controller) RegisterAgent(data *Agent) (string, error) {
	slog.Debug("Register Agent", "data", fmt.Sprintf("%+v", data))

	agent, err := c.marshalNewAgent(data)
	if err != nil {
		return "", err
	}

	exists, err := c.model.GetAgent(&model.Agent{Hostname: data.Hostname})
	if err != nil && !errors.Is(err, model.ErrAgentNotFound) {
		return "", err
	}
	if exists != nil {
		return "", ErrAgentAlreadyExists
	}

	_, err = c.model.CreateAgent(agent)
	if err != nil {
		return "", err
	}

	go func() {
		if err := c.generateCredentialsFile(); err != nil {
			log.Printf("failed to generate credentials file: %v", err)
		}
	}()

	return agent.ResourceId, nil
}

func (c *Controller) DeregisterAgent(rid string) error {
	slog.Debug("Deregister Agent", "rid", rid)

	agent, err := c.model.GetAgent(&model.Agent{ResourceId: rid})
	if err != nil {
		if errors.Is(err, model.ErrAgentNotFound) {
			return ErrAgentNotFound
		}
		return err
	}

	if err := c.model.DeleteAgent(agent); err != nil {
		return err
	}

	go func() {
		if err := c.generateCredentialsFile(); err != nil {
			log.Printf("failed to generate credentials file: %v", err)
		}
	}()

	return nil
}

func (c *Controller) CreateAgentConfig(rid string) ([]byte, error) {
	slog.Debug("Create Agent Config", "rid", rid)

	agent, err := c.model.GetAgent(&model.Agent{ResourceId: rid})
	if err != nil {
		if errors.Is(err, model.ErrAgentNotFound) {
			return nil, ErrAgentNotFound
		}
		return nil, err
	}

	tmpl, err := template.New("alloy.cfg").Parse(alloyTemplate)
	if err != nil {
		return nil, err
	}

	data, err := c.generateAlloyConfig(agent)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (c *Controller) ListAgents() ([]map[string]string, error) {
	slog.Debug("List Agents")

	agents := []model.Agent{}
	_, err := c.model.ListAgents(&agents)
	if err != nil {
		return nil, err
	}

	list := make([]map[string]string, 0, len(agents))
	for _, agent := range agents {
		entry := map[string]string{
			"rid":      agent.ResourceId,
			"hostname": agent.Hostname,
		}
		list = append(list, entry)
	}

	return list, nil
}

func (c *Controller) GetAgent(rid string) (*model.Agent, error) {
	slog.Debug("Get Agent", "rid", rid)

	agent, err := c.model.GetAgent(&model.Agent{ResourceId: rid})
	if err != nil {
		if errors.Is(err, model.ErrAgentNotFound) {
			return nil, ErrAgentNotFound
		}
		return nil, err
	}

	password, err := aes.Decrypt(c.config.Secret(), agent.Password)
	if err != nil {
		return nil, err
	}
	agent.Password = password

	return agent, nil
}

func (c *Controller) UpdateAgent(rid string, data *Agent) error {
	slog.Debug("Update Agent", "rid", rid, "data", fmt.Sprintf("%+v", data))

	agent, err := c.model.GetAgent(&model.Agent{ResourceId: rid})
	if err != nil {
		if errors.Is(err, model.ErrAgentNotFound) {
			return ErrAgentNotFound
		}
		return err
	}

	updated, err := c.marshalUpdateAgent(agent, data)
	if err != nil {
		return err
	}

	_, err = c.model.UpdateAgent(updated)
	if err != nil {
		return err
	}

	return nil
}
