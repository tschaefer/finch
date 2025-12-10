/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package config

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ReadReturnsError_NotExistingFile(t *testing.T) {
	_, err := NewFromFile("/path/not/found/finch.json")
	assert.Error(t, err, "read config file")

	wanted := "no such file or directory"
	assert.Contains(t, err.Error(), wanted, "error message")
}

func Test_ReadReturnsError_RelativeFilePath(t *testing.T) {
	_, err := NewFromFile("relative/path/finch.json")
	assert.Error(t, err, "read config file")

	wanted := "configuration file path must be absolute:"
	assert.Contains(t, err.Error(), wanted, "error message")
}

func Test_ReadReturnsError_InvalidJSON(t *testing.T) {
	f, _ := os.CreateTemp("", "invalid.json")
	defer func() {
		_ = os.Remove(f.Name())
	}()
	_, _ = fmt.Fprintf(f, `- invalid json`)

	_, err := NewFromFile(f.Name())
	assert.Error(t, err, "read config file")

	wanted := "failed to unmarshal configuration file"
	assert.Contains(t, err.Error(), wanted, "error message")
}

func Test_ReadReturnsError_MissingField(t *testing.T) {
	f, _ := os.CreateTemp("", "missing_field.json")
	defer func() {
		_ = os.Remove(f.Name())
	}()
	_, _ = fmt.Fprintf(f, `{"version": "1.0", "hostname": "localhost"}`)

	_, err := NewFromFile(f.Name())
	assert.Error(t, err, "read config file")

	wanted := "invalid configuration data, missing field: CreatedAt"
	assert.Contains(t, err.Error(), wanted, "error message")
}

func Test_ReadReturnsError_MissingCredentialsField(t *testing.T) {
	f, _ := os.CreateTemp("", "missing_credentials.json")
	defer func() {
		_ = os.Remove(f.Name())
	}()
	_, _ = fmt.Fprintf(f, `{"version": "1.0", "hostname": "localhost", "created_at": "2023-10-01T00:00:00Z", "id": "12345", "database": "testdb", "secret": "secret", "credentials": { "username": "user"}}`)

	_, err := NewFromFile(f.Name())
	assert.Error(t, err, "read config file")

	wanted := "invalid configuration data, missing field: Credentials.Password"
	assert.Contains(t, err.Error(), wanted, "error message")
}

func Test_ReadReturnsConfig(t *testing.T) {
	f, _ := os.CreateTemp("", "valid_config.json")
	defer func() {
		_ = os.Remove(f.Name())
	}()
	_, _ = fmt.Fprintf(f, `{
		"created_at": "2023-10-01T00:00:00Z",
		"database": "testdb",
		"hostname": "localhost",
		"id": "12345",
		"secret": "secret",
		"version": "1.0",
		"credentials": {
			"username": "user",
			"password": "pass"
		}
	}`)

	config, err := NewFromFile(f.Name())
	assert.NoError(t, err, "read config file")

	assert.Equal(t, "1.0", config.Version(), "version")
	assert.Equal(t, "localhost", config.Hostname(), "hostname")
	assert.Equal(t, "testdb", config.Database(), "database")
	assert.Equal(t, "12345", config.Id(), "id")
	assert.Equal(t, "secret", config.Secret(), "secret")
	username, password := config.Credentials()
	assert.Equal(t, "user", username, "credentials username")
	assert.Equal(t, "pass", password, "credentials password")
}
