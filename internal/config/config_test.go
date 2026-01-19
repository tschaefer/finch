/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package config

import (
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
	_, err := NewFromString(`- invalid json`, "")
	assert.Error(t, err, "read config string")

	wanted := "failed to unmarshal configuration file"
	assert.Contains(t, err.Error(), wanted, "error message")
}

func Test_ReadReturnsError_MissingField(t *testing.T) {
	_, err := NewFromString(`{"version": "1.0", "hostname": "localhost"}`, "")
	assert.Error(t, err, "read config string")

	wanted := "invalid configuration data, missing field: CreatedAt"
	assert.Contains(t, err.Error(), wanted, "error message")
}

func Test_ReadReturnsConfig(t *testing.T) {
	cfg, err := NewFromString(`{
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
	}`, "")
	assert.NoError(t, err, "read config string")

	assert.Equal(t, "1.0", cfg.Version(), "version")
	assert.Equal(t, "localhost", cfg.Hostname(), "hostname")
	assert.Equal(t, "testdb", cfg.Database(), "database")
	assert.Equal(t, "12345", cfg.Id(), "id")
	assert.Equal(t, "secret", cfg.Secret(), "secret")
}
