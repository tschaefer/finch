/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package controller

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/url"
	"os"
	"slices"
	"text/template"
	"time"

	"github.com/gofrs/flock"
	"github.com/google/uuid"
	"github.com/tschaefer/finch/internal/aes"
	"github.com/tschaefer/finch/internal/model"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrAgentNotFound      = errors.New("agent not found")
	ErrAgentAlreadyExists = errors.New("agent already exists")
)

type ControllerAgent interface {
	RegisterAgent(hostname string, tags, logSources []string, metrics bool, profiles bool) (string, error)
	DeregisterAgent(rid string) error
	CreateAgentConfig(rid string) ([]byte, error)
	ListAgents() ([]map[string]string, error)
	GetAgent(rid string) (*model.Agent, error)
}

const alloyTemplate = `
logging {
	level = "info"
}

loki.write "default" {
	endpoint {
		url = "https://{{ .ServiceName }}/loki/loki/api/v1/push"

		basic_auth {
			username = "{{ .Username }}"
			password = "{{ .Password }}"
		}

		tls_config {
			insecure_skip_verify = true
		}
	}
	external_labels = {
		"host" = "{{ .Hostname }}",
		"rid" = "{{ .ResourceId }}",
	}
}

{{ if .LogSources.Journal -}}
loki.relabel "journal" {
	forward_to = []

	rule {
		source_labels = ["__journal__systemd_unit"]
		target_label  = "unit"
	}

	rule {
		source_labels = ["__journal__boot_id"]
		target_label  = "boot_id"
	}

	rule {
		source_labels = ["__journal__transport"]
		target_label  = "transport"
	}

	rule {
		source_labels = ["__journal_priority_keyword"]
		target_label  = "level"
	}
}

loki.source.journal "journal" {
	format_as_json = true
	max_age        = "12h"
	relabel_rules  = loki.relabel.journal.rules
	forward_to     = [loki.write.default.receiver]
}
{{ end -}}

{{ if .LogSources.Docker }}
discovery.docker "linux" {
	host = "unix:///var/run/docker.sock"
}

loki.relabel "docker" {
	forward_to = []

	rule {
		source_labels = ["__meta_docker_container_name"]
		regex         = "/(.*)"
		target_label  = "service_name"
	}
}

loki.source.docker "docker" {
	host    = "unix:///var/run/docker.sock"
	targets = discovery.docker.linux.targets
	labels  = {
		"platform" = "docker",
	}
	relabel_rules = loki.relabel.docker.rules
	forward_to    = [loki.write.default.receiver]
}
{{ end -}}

{{ if .LogSources.Files }}

loki.process "files" {
	stage.static_labels {
		values = {
			service_name = "loki.source.file.files",
		}
	}
	forward_to = [loki.write.default.receiver]
}

local.file_match "files" {
	path_targets = [
			{{- range .LogSources.Files }}
			{{ . }},
			{{- end }}
		]
}

loki.source.file "file" {
	targets    = local.file_match.files.targets
	forward_to = [loki.process.files.receiver]
}
{{ end -}}

{{ if .Metrics }}

prometheus.remote_write "default" {
	endpoint {
		url = "https://{{ .ServiceName }}/mimir/api/v1/push"

		basic_auth {
			username = "{{ .Username }}"
			password = "{{ .Password }}"
		}

		tls_config {
			insecure_skip_verify = true
		}
	}
	external_labels = {
		"host" = "{{ .Hostname }}",
		"rid" = "{{ .ResourceId }}",
	}
}

prometheus.exporter.unix "node" {
	include_exporter_metrics = true
	enable_collectors = [
		"systemd",
	]
}

prometheus.scrape "node" {
	targets         = prometheus.exporter.unix.node.targets
	forward_to      = [prometheus.remote_write.default.receiver]
	scrape_interval = "15s"
}

prometheus.receive_http "default" {
	http {
		listen_address = "127.0.0.1"
		listen_port = 9091
	}
	forward_to = [prometheus.remote_write.default.receiver]
}
{{ end -}}

{{ if .Profiles }}
pyroscope.receive_http "default" {
	http {
		listen_address = "127.0.0.1"
		listen_port = 4040
	}
	forward_to = [pyroscope.write.backend.receiver]
}

pyroscope.write "backend" {
	endpoint {
		url = "https://{{ .ServiceName }}/pyroscope"

		basic_auth {
			username = "{{ .Username }}"
			password = "{{ .Password }}"
		}

		tls_config {
			insecure_skip_verify = true
		}
	}
	external_labels = {
		"host" = "{{ .Hostname }}",
		"rid" = "{{ .ResourceId }}",
	}
}
{{ end -}}
`

const lokiUsersTemplate = `---
http:
  middlewares:
    loki-auth:
      basicAuth:
        users:
{{- if . }}
{{- range . }}
          - "{{ .Username }}:{{ .PasswordHash }}"
{{- end }}
{{- else }}
          - ""
{{- end }}
`

func (c *controller) RegisterAgent(hostname string, tags, logSources []string, Metrics bool, Profiles bool) (string, error) {
	slog.Debug("Register Agent", "hostname", hostname, "tags", tags, "logSources", logSources, "metrics", Metrics, "profiles", Profiles)

	agent, err := c.marshalAgent(hostname, tags, logSources, Metrics, Profiles)
	if err != nil {
		return "", err
	}

	exists, err := c.model.GetAgent(&model.Agent{Hostname: hostname})
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

func (c *controller) DeregisterAgent(rid string) error {
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

func (c *controller) CreateAgentConfig(rid string) ([]byte, error) {
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

	password, err := aes.Decrypt(c.config.Secret(), agent.Password)
	if err != nil {
		return nil, err
	}

	data := struct {
		ServiceName string
		Hostname    string
		Username    string
		Password    string
		ResourceId  string
		LogSources  struct {
			Journal bool
			Docker  bool
			Files   []string
		}
		Metrics  bool
		Profiles bool
	}{
		Hostname:    agent.Hostname,
		ServiceName: c.config.Hostname(),
		Username:    agent.Username,
		Password:    password,
		ResourceId:  agent.ResourceId,
		LogSources: struct {
			Journal bool
			Docker  bool
			Files   []string
		}{
			Journal: false,
			Docker:  false,
			Files:   make([]string, 0),
		},
		Metrics:  agent.Metrics,
		Profiles: agent.Profiles,
	}

	files := make([]string, 0)
	for _, source := range agent.LogSources {
		uri, err := url.Parse(source)
		if err != nil {
			continue
		}
		switch uri.Scheme {
		case "journal":
			data.LogSources.Journal = true
		case "docker":
			data.LogSources.Docker = true
		case "file":
			files = append(files, fmt.Sprintf("{__path__ = \"%s\"}", uri.Path))
			data.LogSources.Files = files // fmt.Sprintf("[%s,]", strings.Join(files, ", "))
		default:
			continue
		}
	}

	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (c *controller) ListAgents() ([]map[string]string, error) {
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

func (c *controller) GetAgent(rid string) (*model.Agent, error) {
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

func (c *controller) marshalAgent(hostname string, tags, logSources []string, metrics bool, profiles bool) (*model.Agent, error) {
	if hostname == "" {
		return nil, fmt.Errorf("hostname must not be empty")
	}

	if len(logSources) == 0 {
		return nil, fmt.Errorf("at least one log source must be specified")
	}

	var effectiveLogSources []string
	for _, logSource := range logSources {
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

	password := rand.Text()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	password, err = aes.Encrypt(c.config.Secret(), password)
	if err != nil {
		return nil, err
	}

	agent := &model.Agent{
		Hostname:     hostname,
		LogSources:   effectiveLogSources,
		Metrics:      metrics,
		Profiles:     profiles,
		Tags:         tags,
		ResourceId:   fmt.Sprintf("rid:finch:%s:agent:%s", c.config.Id(), uuid.New().String()),
		Username:     rand.Text(),
		Password:     password,
		PasswordHash: string(hash),
	}

	return agent, nil
}

func (c *controller) generateCredentialsFile() error {
	confDir := fmt.Sprintf("%s/traefik/etc/conf.d", c.config.Library())
	usersFile := fmt.Sprintf("%s/loki-users.yaml", confDir)

	f, err := os.CreateTemp(confDir, "loki-users")
	if err != nil {
		return err
	}
	defer func() {
		_ = os.Remove(f.Name())
	}()

	agents := []model.Agent{}
	_, err = c.model.ListAgents(&agents)
	if err != nil {
		return err
	}

	tmpl, err := template.New("loki-users.yaml").Parse(lokiUsersTemplate)
	if err != nil {
		return nil
	}

	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, agents); err != nil {
		return err
	}

	_, err = fmt.Fprint(f, buf.String())
	if err != nil {
		return err
	}

	fileLock := flock.New(usersFile)
	var locked bool
	for range 25 {
		locked, _ = fileLock.TryLock()
		if locked {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !locked {
		return errors.New("failed to acquire file lock")
	}
	defer func() {
		_ = fileLock.Unlock()
	}()

	if err := os.Rename(f.Name(), usersFile); err != nil {
		return err
	}

	if err := os.Chmod(usersFile, 0600); err != nil {
		return err
	}

	return nil
}
