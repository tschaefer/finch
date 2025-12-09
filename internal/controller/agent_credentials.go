/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package controller

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"text/template"
	"time"

	"github.com/gofrs/flock"
	"github.com/tschaefer/finch/internal/model"
)

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
		return err
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
