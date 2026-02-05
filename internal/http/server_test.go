/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tschaefer/finch/internal/config"
	"github.com/tschaefer/finch/internal/controller"
	"github.com/tschaefer/finch/internal/database"
	"github.com/tschaefer/finch/internal/model"
)

var testCfg = config.NewFromData(&config.Data{
	Id:        "test-id",
	Hostname:  "localhost",
	CreatedAt: "2025-01-01T00:00:00Z",
	Database:  "sqlite://:memory:",
	Secret:    "gpFb8WTh5iELimbX3YfuvRYRh2Z2PHa8Lmoog0a25QQ=",
}, "")

func newTestController(t *testing.T) *controller.Controller {
	db, err := database.New(testCfg)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Migrate()
	if err != nil {
		t.Fatal(err)
	}

	model := model.New(db.Connection())

	return controller.New(model, testCfg)
}

func TestNewServerCreatesServerWithRoutes(t *testing.T) {
	ctrl := newTestController(t)
	server := NewServer("127.0.0.1:0", ctrl, testCfg)

	assert.NotNil(t, server)
	assert.NotNil(t, server.server)
	assert.NotNil(t, server.controller)
	assert.NotNil(t, server.config)
	assert.Equal(t, 10*time.Second, server.server.ReadTimeout)
	assert.Equal(t, 10*time.Second, server.server.WriteTimeout)
	assert.Equal(t, 60*time.Second, server.server.IdleTimeout)
}

func TestServerStartAndStop(t *testing.T) {
	ctrl := newTestController(t)
	server := NewServer("127.0.0.1:0", ctrl, testCfg)

	err := server.Start()
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = server.Stop(ctx)
	assert.NoError(t, err)
}

func TestDashboardRouteExists(t *testing.T) {
	ctrl := newTestController(t)
	server := NewServer("127.0.0.1:0", ctrl, testCfg)

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rec := httptest.NewRecorder()

	server.server.Handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/login", rec.Header().Get("Location"))
}

func TestWebSocketRouteExists(t *testing.T) {
	ctrl := newTestController(t)
	server := NewServer("127.0.0.1:0", ctrl, testCfg)

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	rec := httptest.NewRecorder()

	server.server.Handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/login", rec.Header().Get("Location"))
}

func TestNonExistentRouteReturnsNotFound(t *testing.T) {
	ctrl := newTestController(t)
	server := NewServer("127.0.0.1:0", ctrl, testCfg)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	rec := httptest.NewRecorder()

	server.server.Handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestSecurityHeadersAreSet(t *testing.T) {
	ctrl := newTestController(t)
	server := NewServer("127.0.0.1:0", ctrl, testCfg)

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rec := httptest.NewRecorder()

	server.server.Handler.ServeHTTP(rec, req)

	assert.Contains(t, rec.Header().Get("Content-Security-Policy"), "default-src 'self'")
	assert.Contains(t, rec.Header().Get("Content-Security-Policy"), "script-src 'self' 'unsafe-inline'")
	assert.Contains(t, rec.Header().Get("Content-Security-Policy"), "connect-src 'self' ws: wss:")
	assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "1; mode=block", rec.Header().Get("X-XSS-Protection"))
	assert.Equal(t, "strict-origin-when-cross-origin", rec.Header().Get("Referrer-Policy"))
}

func TestHSTSHeaderSetOnlyForHTTPS(t *testing.T) {
	ctrl := newTestController(t)
	server := NewServer("127.0.0.1:0", ctrl, testCfg)

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rec := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(rec, req)
	assert.Empty(t, rec.Header().Get("Strict-Transport-Security"))

	req = httptest.NewRequest(http.MethodGet, "/login", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rec = httptest.NewRecorder()
	server.server.Handler.ServeHTTP(rec, req)
	assert.Equal(t, "max-age=31536000; includeSubDomains", rec.Header().Get("Strict-Transport-Security"))
}
