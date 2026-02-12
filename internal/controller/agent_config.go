/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package controller

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/tschaefer/finch/internal/model"
)

const alloyTemplate = `
logging {
	level = "info"
}

loki.write "default" {
	endpoint {
		url = "https://{{ .ServiceName }}/loki/loki/api/v1/push"

		// Token expires: {{ .TokenExpiry }}
		bearer_token = "{{ .Token }}"

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

		// Token expires: {{ .TokenExpiry }}
		bearer_token = "{{ .Token }}"

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

		// Token expires: {{ .TokenExpiry }}
		bearer_token = "{{ .Token }}"

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

type alloyConfigData struct {
	ServiceName string
	Hostname    string
	Token       string
	TokenExpiry string
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
}

func (c *Controller) generateAlloyConfig(agent *model.Agent) (*alloyConfigData, error) {
	token, expiresAt, err := c.GenerateAgentToken(agent.ResourceId, 0)
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

	data := &alloyConfigData{
		Hostname:    agent.Hostname,
		ServiceName: c.config.Hostname(),
		Token:       token,
		TokenExpiry: expiresAt.Format("2006-01-02 15:04:05 MST"),
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

	return data, nil
}
