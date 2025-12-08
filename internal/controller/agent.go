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
	"strings"
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

type Agent struct {
	Hostname       string   `json:"hostname"`
	Labels         []string `json:"labels"`
	LogSources     []string `json:"log_sources"`
	Metrics        bool     `json:"metrics"`
	MetricsTargets []string `json:"metrics_targets"`
	Profiles       bool     `json:"profiles"`
}

type ControllerAgent interface {
	RegisterAgent(agent *Agent) (string, error)
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
		{{- range .Labels }}
		{{ . }},
		{{- end }}
	}
}

loki.source.api "default" {
	http {
		listen_address = "127.0.0.1"
		listen_port = "3100"
	}
	forward_to = [loki.write.default.receiver]
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
		{{- range .Labels }}
		{{ . }},
		{{- end }}
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

{{ if .MetricsTargets }}
{{ range $index, $source := .MetricsTargets }}
prometheus.scrape "custom_{{ $index }}" {
	targets    = [{"__address__" = "{{ $source.Address }}"}]
	forward_to = [prometheus.remote_write.default.receiver]
	scrape_interval = "15s"
	{{ if $source.MetricsPath }}
	metrics_path = "{{ $source.MetricsPath }}"
	{{- end }}
}
{{ end -}}
{{ end -}}
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
		{{- range .Labels }}
		{{ . }},
		{{- end }}
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

func (c *controller) RegisterAgent(data *Agent) (string, error) {
	slog.Debug("Register Agent", "data", fmt.Sprintf("%+v", data))

	agent, err := c.marshalAgent(data)
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

	labels := make([]string, 0)
	for _, label := range agent.Labels {
		if strings.Contains(label, "=") {
			parts := strings.SplitN(label, "=", 2)
			labels = append(labels, fmt.Sprintf("\"%s\" = \"%s\"", parts[0], parts[1]))
		} else {
			labels = append(labels, fmt.Sprintf("\"%s\" = \"true\"", label))
		}
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
		Metrics        bool
		MetricsTargets []struct {
			Address     string
			MetricsPath string
		}
		Profiles bool
		Labels   []string
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
		Metrics: agent.Metrics,
		MetricsTargets: make([]struct {
			Address     string
			MetricsPath string
		}, 0),
		Profiles: agent.Profiles,
		Labels:   labels,
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
			data.LogSources.Files = files
		default:
			continue
		}
	}

	for _, source := range agent.MetricsTargets {
		uri, err := url.Parse(source)
		if err != nil {
			continue
		}
		entry := struct {
			Address     string
			MetricsPath string
		}{
			Address:     uri.Host,
			MetricsPath: uri.Path,
		}
		data.MetricsTargets = append(data.MetricsTargets, entry)
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

func (c *controller) marshalAgent(data *Agent) (*model.Agent, error) {
	if data.Hostname == "" {
		return nil, fmt.Errorf("hostname must not be empty")
	}

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
