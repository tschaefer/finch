/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package http

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type logEntry struct {
	Level      string `json:"level"`
	Msg        string `json:"msg"`
	RemoteAddr string `json:"remote_addr"`
	UserAgent  string `json:"user_agent"`
	Rid        string `json:"rid"`
}

func Test_Log_ExtractsRemoteAddrFromXForwardedFor(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	server := &Server{}
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.100")
	req.Header.Set("User-Agent", "test-agent")

	server.log(req, slog.LevelInfo, "test message", "rid", "test-123")

	var entry logEntry
	err := json.Unmarshal(buf.Bytes(), &entry)
	assert.NoError(t, err, "parse log entry")
	assert.Equal(t, "INFO", entry.Level)
	assert.Equal(t, "test message", entry.Msg)
	assert.Equal(t, "192.168.1.100", entry.RemoteAddr)
	assert.Equal(t, "test-agent", entry.UserAgent)
	assert.Equal(t, "test-123", entry.Rid)
}

func Test_Log_ExtractsRemoteAddrFromXRealIp(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	server := &Server{}
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Real-Ip", "10.0.0.50")
	req.Header.Set("User-Agent", "another-agent")

	server.log(req, slog.LevelInfo, "test message")

	var entry logEntry
	err := json.Unmarshal(buf.Bytes(), &entry)
	assert.NoError(t, err, "parse log entry")
	assert.Equal(t, "10.0.0.50", entry.RemoteAddr)
	assert.Equal(t, "another-agent", entry.UserAgent)
}

func Test_Log_PrefersXForwardedForOverXRealIp(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	server := &Server{}
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.100")
	req.Header.Set("X-Real-Ip", "10.0.0.50")

	server.log(req, slog.LevelInfo, "test message")

	var entry logEntry
	err := json.Unmarshal(buf.Bytes(), &entry)
	assert.NoError(t, err, "parse log entry")
	assert.Equal(t, "192.168.1.100", entry.RemoteAddr)
}

func Test_Log_ExtractsRemoteAddrFromRemoteAddrAndStripsPort(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	server := &Server{}
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.100:54321"

	server.log(req, slog.LevelInfo, "test message")

	var entry logEntry
	err := json.Unmarshal(buf.Bytes(), &entry)
	assert.NoError(t, err, "parse log entry")
	assert.Equal(t, "192.168.1.100", entry.RemoteAddr)
}

func Test_Log_HandlesIPv6WithPort(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	server := &Server{}
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "[::1]:54321"

	server.log(req, slog.LevelInfo, "test message")

	var entry logEntry
	err := json.Unmarshal(buf.Bytes(), &entry)
	assert.NoError(t, err, "parse log entry")
	assert.Equal(t, "[::1]", entry.RemoteAddr)
}

func Test_Log_HandlesEmptyUserAgent(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	server := &Server{}
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.100:54321"

	server.log(req, slog.LevelInfo, "test message")

	var entry logEntry
	err := json.Unmarshal(buf.Bytes(), &entry)
	assert.NoError(t, err, "parse log entry")
	assert.Equal(t, "", entry.UserAgent)
}

func Test_Log_RespectsLogLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	server := &Server{}
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.100:54321"

	server.log(req, slog.LevelWarn, "warning message")

	var entry logEntry
	err := json.Unmarshal(buf.Bytes(), &entry)
	assert.NoError(t, err, "parse log entry")
	assert.Equal(t, "WARN", entry.Level)
	assert.Equal(t, "warning message", entry.Msg)
}

func Test_Log_AppendsAdditionalArgs(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	server := &Server{}
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.100:54321"

	server.log(req, slog.LevelInfo, "test message", "key1", "value1", "key2", 42)

	var entry map[string]any
	err := json.Unmarshal(buf.Bytes(), &entry)
	assert.NoError(t, err, "parse log entry")
	assert.Equal(t, "value1", entry["key1"])
	assert.Equal(t, float64(42), entry["key2"])
}
