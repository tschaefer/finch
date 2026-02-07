/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/tschaefer/finch/internal/controller"
)

func TestHandleDashboardRendersTemplate(t *testing.T) {
	ctrl := newTestController(t)
	server := NewServer("127.0.0.1:0", ctrl, testCfg)

	resp, err := ctrl.GetDashboardToken(1800)
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.AddCookie(&http.Cookie{
		Name:  "dashboard_token",
		Value: resp.Token,
	})
	rec := httptest.NewRecorder()

	server.server.Handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Finch Dashboard")
	assert.Contains(t, rec.Body.String(), "<!DOCTYPE html>")
}

func TestHandleLoginRendersTemplate(t *testing.T) {
	ctrl := newTestController(t)
	server := NewServer("127.0.0.1:0", ctrl, testCfg)

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rec := httptest.NewRecorder()

	server.server.Handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Finch Dashboard - Login")
	assert.Contains(t, rec.Body.String(), "<!DOCTYPE html>")
}

func TestHandleWebSocketUpgrade(t *testing.T) {
	ctrl := newTestController(t)
	server := NewServer("127.0.0.1:0", ctrl, testCfg)

	resp, err := ctrl.GetDashboardToken(1800)
	assert.NoError(t, err)

	testServer := httptest.NewServer(server.server.Handler)
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http") + "/ws"

	headers := http.Header{}
	headers.Add("Cookie", "dashboard_token="+resp.Token)

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, headers)
	assert.NoError(t, err)
	defer func() {
		_ = ws.Close()
	}()

	var msg WSResponse
	err = ws.ReadJSON(&msg)
	assert.NoError(t, err)
	assert.Contains(t, []string{"info", "stats", "endpoints", "agents"}, msg.Type)
}

func TestHandleWebSocketRejectsWithoutAuth(t *testing.T) {
	ctrl := newTestController(t)
	server := NewServer("127.0.0.1:0", ctrl, testCfg)

	testServer := httptest.NewServer(server.server.Handler)
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http") + "/ws"

	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.Error(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestHandleWebSocketRejectsInvalidToken(t *testing.T) {
	ctrl := newTestController(t)
	server := NewServer("127.0.0.1:0", ctrl, testCfg)

	testServer := httptest.NewServer(server.server.Handler)
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http") + "/ws"

	headers := http.Header{}
	headers.Add("Cookie", "dashboard_token=invalid_token")

	_, resp, err := websocket.DefaultDialer.Dial(wsURL, headers)
	assert.Error(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestWebSocketSendsInfoUpdate(t *testing.T) {
	ctrl := newTestController(t)
	server := NewServer("127.0.0.1:0", ctrl, testCfg)

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() {
			_ = conn.Close()
		}()

		server.sendInfoUpdate(conn)
	}))
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	defer func() {
		_ = ws.Close()
	}()

	var msg WSResponse
	err = ws.ReadJSON(&msg)
	assert.NoError(t, err)
	assert.Equal(t, "info", msg.Type)
	assert.Contains(t, msg.HTML, "localhost")
}

func TestWebSocketSendsStatsUpdate(t *testing.T) {
	ctrl := newTestController(t)
	server := NewServer("127.0.0.1:0", ctrl, testCfg)

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() {
			_ = conn.Close()
		}()

		server.sendStatsUpdate(conn)
	}))
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	defer func() {
		_ = ws.Close()
	}()

	var msg WSResponse
	err = ws.ReadJSON(&msg)
	assert.NoError(t, err)
	assert.Equal(t, "stats", msg.Type)
	assert.Contains(t, msg.HTML, "Total Agents")
}

func TestWebSocketSendsEndpointsUpdate(t *testing.T) {
	ctrl := newTestController(t)
	server := NewServer("127.0.0.1:0", ctrl, testCfg)

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() {
			_ = conn.Close()
		}()

		server.sendEndpointsUpdate(conn)
	}))
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	defer func() {
		_ = ws.Close()
	}()

	var msg WSResponse
	err = ws.ReadJSON(&msg)
	assert.NoError(t, err)
	assert.Equal(t, "endpoints", msg.Type)
	assert.Contains(t, msg.HTML, "loki")
	assert.Contains(t, msg.HTML, "mimir")
	assert.Contains(t, msg.HTML, "pyroscope")
}

func TestWebSocketSendsAgentsUpdate(t *testing.T) {
	ctrl := newTestController(t)
	server := NewServer("127.0.0.1:0", ctrl, testCfg)

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()

		server.sendAgentsUpdate(conn, 1, "")
	}))
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	defer func() {
		_ = ws.Close()
	}()

	var msg WSResponse
	err = ws.ReadJSON(&msg)
	assert.NoError(t, err)
	assert.Equal(t, "agents", msg.Type)
}

func TestWebSocketHandlesGetAgentsMessage(t *testing.T) {
	ctrl := newTestController(t)
	server := NewServer("127.0.0.1:0", ctrl, testCfg)

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() {
			_ = conn.Close()
		}()

		msg := WSMessage{
			Type: "get_agents",
			Data: json.RawMessage(`{"page": 1, "search": ""}`),
		}
		server.handleWSMessage(conn, msg)
	}))
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	defer func() { _ = ws.Close() }()

	var msg WSResponse
	err = ws.ReadJSON(&msg)
	assert.NoError(t, err)
	assert.Equal(t, "agents", msg.Type)
}

func TestWebSocketHandlesDownloadConfigMessage(t *testing.T) {
	ctrl := newTestController(t)
	server := NewServer("127.0.0.1:0", ctrl, testCfg)

	agentData := &controller.Agent{
		Hostname:   "test-host",
		LogSources: []string{"journal://"},
	}
	rid, err := ctrl.RegisterAgent(agentData)
	assert.NoError(t, err)

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() {
			_ = conn.Close()
		}()

		msg := WSMessage{
			Type: "download_config",
			Data: json.RawMessage(`{"rid": "` + rid + `"}`),
		}
		server.handleWSMessage(conn, msg)
	}))
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	defer func() {
		_ = ws.Close()
	}()

	var msg map[string]string
	err = ws.ReadJSON(&msg)
	assert.NoError(t, err)
	assert.Equal(t, "config", msg["type"])
	assert.NotEmpty(t, msg["content"])
}

func TestWebSocketHandlesGetCredentialsMessage(t *testing.T) {
	ctrl := newTestController(t)
	server := NewServer("127.0.0.1:0", ctrl, testCfg)

	agentData := &controller.Agent{
		Hostname:   "test-host",
		LogSources: []string{"journal://"},
	}
	rid, err := ctrl.RegisterAgent(agentData)
	assert.NoError(t, err)

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() {
			_ = conn.Close()
		}()

		msg := WSMessage{
			Type: "get_credentials",
			Data: json.RawMessage(`{"rid": "` + rid + `"}`),
		}
		server.handleWSMessage(conn, msg)
	}))
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	defer func() {
		_ = ws.Close()
	}()

	var msg WSResponse
	err = ws.ReadJSON(&msg)
	assert.NoError(t, err)
	assert.Equal(t, "credentials", msg.Type)
	assert.Contains(t, msg.HTML, "Username")
	assert.Contains(t, msg.HTML, "Password")
}

func TestAgentListDataPagination(t *testing.T) {
	ctrl := newTestController(t)
	server := NewServer("127.0.0.1:0", ctrl, testCfg)

	for i := range 11 {
		agentData := &controller.Agent{
			Hostname:   "test-host-" + string(rune('a'+i)),
			LogSources: []string{"journal://"},
		}
		_, err := ctrl.RegisterAgent(agentData)
		assert.NoError(t, err)
	}

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() {
			_ = conn.Close()
		}()

		server.sendAgentsUpdate(conn, 1, "")
	}))
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	defer func() {
		_ = ws.Close()
	}()

	var msg WSResponse
	err = ws.ReadJSON(&msg)
	assert.NoError(t, err)
	assert.Equal(t, "agents", msg.Type)
}

func TestAgentListDataSearch(t *testing.T) {
	ctrl := newTestController(t)
	server := NewServer("127.0.0.1:0", ctrl, testCfg)

	prodAgent := &controller.Agent{
		Hostname:   "prod-server",
		LogSources: []string{"journal://"},
	}
	_, err := ctrl.RegisterAgent(prodAgent)
	assert.NoError(t, err)

	devAgent := &controller.Agent{
		Hostname:   "dev-server",
		LogSources: []string{"journal://"},
	}
	_, err = ctrl.RegisterAgent(devAgent)
	assert.NoError(t, err)

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() {
			_ = conn.Close()
		}()

		server.sendAgentsUpdate(conn, 1, "prod")
	}))
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	defer func() {
		_ = ws.Close()
	}()

	var msg WSResponse
	err = ws.ReadJSON(&msg)
	assert.NoError(t, err)
	assert.Equal(t, "agents", msg.Type)
	assert.Contains(t, msg.HTML, "prod-server")
	assert.NotContains(t, msg.HTML, "dev-server")
}
