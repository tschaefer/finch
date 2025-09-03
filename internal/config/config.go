/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path"
	"reflect"
)

type Data struct {
	CreatedAt   string `json:"created_at"`
	Database    string `json:"database"`
	Hostname    string `json:"hostname"`
	Id          string `json:"id"`
	Secret      string `json:"secret"`
	Version     string `json:"version"`
	Credentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	} `json:"credentials"`
}

type Config interface {
	CreatedAt() string
	Database() string
	Hostname() string
	Id() string
	Library() string
	Secret() string
	Version() string
	Credentials() (string, string)
}

type config struct {
	data    *Data
	library string
}

func Read(file string) (Config, error) {
	slog.Debug("Reading configuration file", "file", file)

	data, err := unmarshal(file)
	if err != nil {
		return nil, err
	}

	if err := valid(data); err != nil {
		return nil, err
	}

	return &config{
		data:    data,
		library: path.Dir(file),
	}, nil
}

func (c *config) Version() string {
	return c.data.Version
}

func (c *config) Hostname() string {
	return c.data.Hostname
}

func (c *config) Database() string {
	return c.data.Database
}

func (c *config) Id() string {
	return c.data.Id
}

func (c *config) CreatedAt() string {
	return c.data.CreatedAt
}

func (c *config) Library() string {
	return c.library
}

func (c *config) Secret() string {
	return c.data.Secret
}

func (c *config) Credentials() (string, string) {
	return c.data.Credentials.Username, c.data.Credentials.Password
}

func unmarshal(file string) (*Data, error) {
	if !path.IsAbs(file) {
		return nil, fmt.Errorf("configuration file path must be absolute: %s", file)
	}

	raw, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file %s: %v", file, err)
	}
	var data Data
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration file: %v", err)
	}
	return &data, nil
}

func valid(data *Data) error {
	fields := []string{
		"CreatedAt",
		"Database",
		"Hostname",
		"Id",
		"Secret",
		"Version",
	}
	object := reflect.ValueOf(*data)
	for _, field := range fields {
		value := object.FieldByName(field)
		if !value.IsValid() || value.Len() == 0 {
			return fmt.Errorf("invalid configuration data, missing field: %s", field)
		}
	}

	object = reflect.ValueOf(data.Credentials)
	for _, field := range []string{"Username", "Password"} {
		value := object.FieldByName(field)
		if !value.IsValid() || value.Len() == 0 {
			return fmt.Errorf("invalid configuration data, missing field: Credentials.%s", field)
		}
	}

	return nil
}
