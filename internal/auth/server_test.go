/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tschaefer/finch/internal/config"
	"github.com/tschaefer/finch/internal/model"
)

func setupTestServer(t *testing.T) (*Server, *model.Model, *config.Config) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(&model.Agent{})
	assert.NoError(t, err)

	m := model.New(db)

	cfg := config.NewFromData(&config.Data{
		Secret: "test-secret-key-32-bytes-long!",
	}, "/tmp")

	server := NewServer(":0", m, cfg)
	return server, m, cfg
}

func generateTestToken(cfg *config.Config, resourceId string, expiration time.Duration) string {
	now := time.Now()
	claims := jwt.MapClaims{
		"rid": resourceId,
		"iat": now.Unix(),
		"exp": now.Add(expiration).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(cfg.Secret()))
	return tokenString
}

func TestHandleAuth_ValidToken(t *testing.T) {
	server, m, cfg := setupTestServer(t)

	agent := &model.Agent{
		Hostname:   "test-host",
		ResourceId: "rid:test:123",
	}
	_, err := m.CreateAgent(agent)
	assert.NoError(t, err)

	token := generateTestToken(cfg, agent.ResourceId, 1*time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/auth", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	server.handleAuth(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAuth_ExpiredToken(t *testing.T) {
	server, m, cfg := setupTestServer(t)

	agent := &model.Agent{
		Hostname:   "test-host",
		ResourceId: "rid:test:123",
	}
	_, err := m.CreateAgent(agent)
	assert.NoError(t, err)

	token := generateTestToken(cfg, agent.ResourceId, -1*time.Hour) // Expired

	req := httptest.NewRequest(http.MethodGet, "/auth", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	server.handleAuth(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandleAuth_UnknownAgent(t *testing.T) {
	server, _, cfg := setupTestServer(t)

	token := generateTestToken(cfg, "rid:unknown:999", 1*time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/auth", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	server.handleAuth(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandleAuth_MissingAuthHeader(t *testing.T) {
	server, _, _ := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/auth", nil)
	w := httptest.NewRecorder()
	server.handleAuth(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandleAuth_InvalidTokenFormat(t *testing.T) {
	server, _, _ := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/auth", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	server.handleAuth(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandleAuth_MissingBearerPrefix(t *testing.T) {
	server, _, cfg := setupTestServer(t)

	token := generateTestToken(cfg, "rid:test:123", 1*time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/auth", nil)
	req.Header.Set("Authorization", token) // Missing "Bearer " prefix
	w := httptest.NewRecorder()
	server.handleAuth(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandleAuth_WrongSignature(t *testing.T) {
	server, m, _ := setupTestServer(t)

	agent := &model.Agent{
		Hostname:   "test-host",
		ResourceId: "rid:test:123",
	}
	_, err := m.CreateAgent(agent)
	assert.NoError(t, err)

	wrongCfg := config.NewFromData(&config.Data{
		Secret: "wrong-secret-key-32-bytes-long",
	}, "/tmp")

	token := generateTestToken(wrongCfg, agent.ResourceId, 1*time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/auth", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	server.handleAuth(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
