/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package controller

import (
	"crypto/rand"
	"fmt"
	"net/url"
	"slices"

	"github.com/google/uuid"
	"github.com/tschaefer/finch/internal/aes"
	"github.com/tschaefer/finch/internal/model"
	"golang.org/x/crypto/bcrypt"
)

func (c *Controller) marshalNewAgent(data *Agent) (*model.Agent, error) {
	if data.Hostname == "" {
		return nil, fmt.Errorf("hostname must not be empty")
	}

	effectiveLogSources, err := c.__parseLogSources(data)
	if err != nil {
		return nil, err
	}

	effectiveMetricsTargets := c.__parseMetricsTargets(data)

	password, hash, err := c.__createCredentials()
	if err != nil {
		return nil, err
	}

	agent := &model.Agent{
		Hostname:       data.Hostname,
		LogSources:     effectiveLogSources,
		Metrics:        data.Metrics,
		MetricsTargets: effectiveMetricsTargets,
		Profiles:       data.Profiles,
		Labels:         data.Labels,
		ResourceId:     fmt.Sprintf("rid:finch:%s:agent:%s", c.config.Id(), uuid.New().String()),
		Username:       rand.Text(),
		Password:       password,
		PasswordHash:   string(hash),
	}

	return agent, nil
}

func (c *Controller) marshalUpdateAgent(existing *model.Agent, data *Agent) (*model.Agent, error) {
	effectiveLogSources, err := c.__parseLogSources(data)
	if err != nil {
		return nil, err
	}

	effectiveMetricsTargets := c.__parseMetricsTargets(data)

	existing.Labels = data.Labels
	existing.LogSources = effectiveLogSources
	existing.Metrics = data.Metrics
	existing.MetricsTargets = effectiveMetricsTargets
	existing.Profiles = data.Profiles

	return existing, nil
}

func (c *Controller) __parseLogSources(data *Agent) ([]string, error) {
	if len(data.LogSources) == 0 {
		return nil, fmt.Errorf("at least one log source must be specified")
	}

	var effectiveLogSources []string
	for _, logSource := range data.LogSources {
		uri, err := url.Parse(logSource)
		if err != nil {
			continue
		}
		if !slices.Contains([]string{"journal", "docker", "file"}, uri.Scheme) {
			continue
		}

		effectiveLogSources = append(effectiveLogSources, uri.String())
	}

	if len(effectiveLogSources) == 0 {
		return nil, fmt.Errorf("no valid log source specified")
	}

	return effectiveLogSources, nil
}

func (c *Controller) __parseMetricsTargets(data *Agent) []string {
	var effectiveMetricsTargets []string
	for _, metricsTarget := range data.MetricsTargets {
		uri, err := url.Parse(metricsTarget)
		if err != nil {
			continue
		}
		if !slices.Contains([]string{"http", "https"}, uri.Scheme) {
			continue
		}
		if uri.Host == "" {
			continue
		}
		effectiveMetricsTargets = append(effectiveMetricsTargets, uri.String())
	}

	return effectiveMetricsTargets
}

func (c *Controller) __createCredentials() (string, string, error) {
	password := rand.Text()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", "", err
	}
	password, err = aes.Encrypt(c.config.Secret(), password)
	if err != nil {
		return "", "", err
	}

	return password, string(hash), nil
}
