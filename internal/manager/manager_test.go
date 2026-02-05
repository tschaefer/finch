/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package manager

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tschaefer/finch/internal/config"
)

func createConfigFile(t *testing.T, data any) string {
	json, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	cfgFile := tmpDir + "/finch.json"
	err = os.WriteFile(cfgFile, json, 0644)
	if err != nil {
		t.Fatal(err)
	}

	return cfgFile
}

func Test_NewReturnsManager(t *testing.T) {
	data := config.Data{
		CreatedAt: "2025-01-01T00:00:00Z",
		Database:  "sqlite://:memory:",
		Hostname:  "finch.example.com",
		Id:        "8d134b24c2541730",
		Secret:    "C7LVMO6YY0ZfZvlEayQJR0zOE7JF8g+nrYgrcvetIbU=",
		Version:   "1.4.0",
	}
	cfgFile := createConfigFile(t, data)
	m, err := New(cfgFile)
	assert.NoError(t, err)
	assert.NotNil(t, m)
}

func Test_NewReturnsError_MissingConfigFile(t *testing.T) {
	_, err := New("nonexistent_file.json")
	assert.Error(t, err)
}

func Test_NewReturnsError_InvalidConfigFile(t *testing.T) {
	cfgFile := createConfigFile(t, "invalid json")
	m, err := New(cfgFile)
	assert.Nil(t, m)
	assert.Error(t, err)
}

func Test_NewReturnsError_InvalidDatabaseURL(t *testing.T) {
	data := config.Data{
		CreatedAt: "2025-01-01T00:00:00Z",
		Database:  "invalid_db_url",
		Hostname:  "finch.example.com",
		Id:        "8d134b24c2541730",
		Secret:    "C7LVMO6YY0ZfZvlEayQJR0zOE7JF8g+nrYgrcvetIbU=",
		Version:   "1.4.0",
	}
	cfgFile := createConfigFile(t, data)
	m, err := New(cfgFile)
	assert.Nil(t, m)
	assert.Error(t, err)
}

func Test_RunSucceeds(t *testing.T) {
	data := config.Data{
		CreatedAt: "2025-01-01T00:00:00Z",
		Database:  "sqlite://:memory:",
		Hostname:  "finch.example.com",
		Id:        "8d134b24c2541730",
		Secret:    "C7LVMO6YY0ZfZvlEayQJR0zOE7JF8g+nrYgrcvetIbU=",
		Version:   "1.4.0",
	}
	cfgFile := createConfigFile(t, data)
	m, err := New(cfgFile)
	assert.NoError(t, err)
	assert.NotNil(t, m)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var grpcPort, httpPort string
	for _, grpcPort = range []string{"11111", "22222", "33333", "44444", "55555", "66666"} {
		conn, _ := net.Dial("tcp", net.JoinHostPort("127.0.0.1", grpcPort))
		if conn == nil {
			break
		}
		_ = conn.Close()
	}
	for _, httpPort = range []string{"11112", "22223", "33334", "44445", "55556", "66667"} {
		conn, _ := net.Dial("tcp", net.JoinHostPort("127.0.0.1", httpPort))
		if conn == nil {
			break
		}
		_ = conn.Close()
	}

	go m.Run(ctx, net.JoinHostPort("127.0.0.1", grpcPort), net.JoinHostPort("127.0.0.1", httpPort))

	var conn net.Conn
	for range 50 {
		conn, err = net.Dial("tcp", net.JoinHostPort("127.0.0.1", grpcPort))
		if conn != nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	assert.NoError(t, err)
	assert.NotNil(t, conn)
	_ = conn.Close()

	cancel()
	time.Sleep(100 * time.Millisecond)
}
