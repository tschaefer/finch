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
	"slices"
)

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Data struct {
	CreatedAt   string      `json:"created_at"`
	Database    string      `json:"database"`
	Profiler    string      `json:"profiler"`
	Hostname    string      `json:"hostname"`
	Id          string      `json:"id"`
	Secret      string      `json:"secret"`
	Version     string      `json:"version"`
	Credentials Credentials `json:"credentials"`
}

type Config struct {
	data    *Data
	library string
}

func NewFromFile(file string) (*Config, error) {
	slog.Debug("Reading configuration file", "file", file)

	if !path.IsAbs(file) {
		return nil, fmt.Errorf("configuration file path must be absolute: %s", file)
	}

	raw, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file %s: %v", file, err)
	}

	return NewFromString(string(raw), path.Dir(file))
}

func NewFromString(jsonString string, library string) (*Config, error) {
	slog.Debug("Parsing configuration from string")

	var data Data
	if err := json.Unmarshal([]byte(jsonString), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration file: %v", err)
	}

	if err := valid(&data); err != nil {
		return nil, err
	}

	slog.Debug("Configuration data", "data", filterSecrets(&data))

	return &Config{
		data:    &data,
		library: library,
	}, nil
}

func NewFromData(data *Data, library string) *Config {
	slog.Debug("Creating configuration from data", "data", filterSecrets(data), "library", library)

	return &Config{
		data:    data,
		library: library,
	}
}

func (c *Config) Version() string {
	return c.data.Version
}

func (c *Config) Hostname() string {
	return c.data.Hostname
}

func (c *Config) Database() string {
	return c.data.Database
}

func (c *Config) Profiler() string {
	return c.data.Profiler
}

func (c *Config) Id() string {
	return c.data.Id
}

func (c *Config) CreatedAt() string {
	return c.data.CreatedAt
}

func (c *Config) Library() string {
	return c.library
}

func (c *Config) Secret() string {
	return c.data.Secret
}

func (c *Config) Credentials() (string, string) {
	return c.data.Credentials.Username, c.data.Credentials.Password
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

func filterSecrets(p any) map[string]string {
	secrets := []string{"Secret", "Credentials"}

	result := make(map[string]string)
	object := reflect.ValueOf(p).Elem()
	typ := object.Type()

	for i := range object.NumField() {
		field := typ.Field(i)
		if slices.Contains(secrets, field.Name) {
			result[field.Name] = "REDACTED"
		} else {
			result[field.Name] = fmt.Sprintf("%v", object.Field(i).Interface())
		}
	}

	return result
}
