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

	grpcListener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err, "allocate gRPC port")
	grpcAddr := grpcListener.Addr().String()
	_ = grpcListener.Close()

	httpListener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err, "allocate HTTP port")
	httpAddr := httpListener.Addr().String()
	_ = httpListener.Close()

	authListener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err, "allocate auth port")
	authAddr := authListener.Addr().String()
	_ = authListener.Close()

	go m.Run(ctx, grpcAddr, httpAddr, authAddr)

	var conn net.Conn
	for range 50 {
		conn, err = net.Dial("tcp", grpcAddr)
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
