/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package config

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func Test_ReadReturnsError_NotExistingFile(t *testing.T) {
	_, err := Read("/path/not/found/finch.json")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	wanted := "no such file or directory"
	if !strings.HasSuffix(err.Error(), wanted) {
		t.Fatalf("wanted '%s' error, got '%s'", wanted, err.Error())
	}
}

func Test_ReadReturnsError_RelativeFilePath(t *testing.T) {
	_, err := Read("relative/path/finch.json")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	wanted := "configuration file path must be absolute:"
	if !strings.HasPrefix(err.Error(), wanted) {
		t.Fatalf("wanted '%s' error, got '%s'", wanted, err.Error())
	}
}

func Test_ReadReturnsError_InvalidJSON(t *testing.T) {
	f, _ := os.CreateTemp("", "invalid.json")
	defer func() {
		_ = os.Remove(f.Name())
	}()
	_, _ = fmt.Fprintf(f, `- invalid json`)

	_, err := Read(f.Name())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	wanted := "failed to unmarshal configuration file"
	if !strings.HasPrefix(err.Error(), wanted) {
		t.Fatalf("wanted '%s' error, got '%s'", wanted, err.Error())
	}
}

func Test_ReadReturnsError_MissingField(t *testing.T) {
	f, _ := os.CreateTemp("", "missing_field.json")
	defer func() {
		_ = os.Remove(f.Name())
	}()
	_, _ = fmt.Fprintf(f, `{"version": "1.0", "hostname": "localhost"}`)

	_, err := Read(f.Name())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	wanted := "invalid configuration data, missing field: CreatedAt"
	if !strings.Contains(err.Error(), wanted) {
		t.Fatalf("wanted '%s' error, got '%s'", wanted, err.Error())
	}
}

func Test_ReadReturnsError_MissingCredentialsField(t *testing.T) {
	f, _ := os.CreateTemp("", "missing_credentials.json")
	defer func() {
		_ = os.Remove(f.Name())
	}()
	_, _ = fmt.Fprintf(f, `{"version": "1.0", "hostname": "localhost", "created_at": "2023-10-01T00:00:00Z", "id": "12345", "database": "testdb", "secret": "secret", "credentials": { "username": "user"}}`)

	_, err := Read(f.Name())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	wanted := "invalid configuration data, missing field: Credentials.Password"
	if !strings.Contains(err.Error(), wanted) {
		t.Fatalf("wanted '%s' error, got '%s'", wanted, err.Error())
	}
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

	config, err := Read(f.Name())
	if err != nil {
		t.Fatalf("expected no error, got '%s'", err)
	}

	if config.Version() != "1.0" {
		t.Errorf("expected version '1.0', got '%s'", config.Version())
	}
	if config.Hostname() != "localhost" {
		t.Errorf("expected hostname 'localhost', got '%s'", config.Hostname())
	}
	if config.Database() != "testdb" {
		t.Errorf("expected database 'testdb', got '%s'", config.Database())
	}
	if config.Id() != "12345" {
		t.Errorf("expected id '12345', got '%s'", config.Id())
	}
	if config.Secret() != "secret" {
		t.Errorf("expected secret 'secret', got '%s'", config.Secret())
	}
	username, password := config.Credentials()
	if username != "user" || password != "pass" {
		t.Errorf("expected credentials (user, pass), got (%s, %s)", username, password)
	}
}
