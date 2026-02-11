/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package auth

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tschaefer/finch/internal/model"
)

func setupTestModel(t *testing.T) *model.Model {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(&model.Agent{})
	assert.NoError(t, err)

	return model.New(db)
}

func TestHandleAuth_ValidCredentials(t *testing.T) {
	m := setupTestModel(t)

	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	assert.NoError(t, err)

	agent := &model.Agent{
		Hostname:     "test-host",
		Username:     "testuser",
		PasswordHash: string(hash),
		ResourceId:   "test-rid",
	}
	_, err = m.CreateAgent(agent)
	assert.NoError(t, err)

	server := NewServer(":0", m)

	req := httptest.NewRequest(http.MethodGet, "/auth", nil)
	credentials := base64.StdEncoding.EncodeToString([]byte("testuser:password123"))
	req.Header.Set("Authorization", "Basic "+credentials)

	w := httptest.NewRecorder()
	server.handleAuth(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAuth_InvalidPassword(t *testing.T) {
	m := setupTestModel(t)

	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	assert.NoError(t, err)

	agent := &model.Agent{
		Hostname:     "test-host",
		Username:     "testuser",
		PasswordHash: string(hash),
		ResourceId:   "test-rid",
	}
	_, err = m.CreateAgent(agent)
	assert.NoError(t, err)

	server := NewServer(":0", m)

	req := httptest.NewRequest(http.MethodGet, "/auth", nil)
	credentials := base64.StdEncoding.EncodeToString([]byte("testuser:wrongpassword"))
	req.Header.Set("Authorization", "Basic "+credentials)

	w := httptest.NewRecorder()
	server.handleAuth(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandleAuth_UnknownUser(t *testing.T) {
	m := setupTestModel(t)
	server := NewServer(":0", m)

	req := httptest.NewRequest(http.MethodGet, "/auth", nil)
	credentials := base64.StdEncoding.EncodeToString([]byte("unknownuser:password123"))
	req.Header.Set("Authorization", "Basic "+credentials)

	w := httptest.NewRecorder()
	server.handleAuth(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandleAuth_MissingAuthHeader(t *testing.T) {
	m := setupTestModel(t)
	server := NewServer(":0", m)

	req := httptest.NewRequest(http.MethodGet, "/auth", nil)
	w := httptest.NewRecorder()
	server.handleAuth(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandleAuth_MalformedAuthHeader(t *testing.T) {
	m := setupTestModel(t)
	server := NewServer(":0", m)

	req := httptest.NewRequest(http.MethodGet, "/auth", nil)
	req.Header.Set("Authorization", "Basic not-valid-base64!!!")
	w := httptest.NewRecorder()
	server.handleAuth(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandleAuth_InvalidCredentialsFormat(t *testing.T) {
	m := setupTestModel(t)
	server := NewServer(":0", m)

	req := httptest.NewRequest(http.MethodGet, "/auth", nil)
	credentials := base64.StdEncoding.EncodeToString([]byte("no-colon-separator"))
	req.Header.Set("Authorization", "Basic "+credentials)

	w := httptest.NewRecorder()
	server.handleAuth(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandleAuth_NonBasicAuthScheme(t *testing.T) {
	m := setupTestModel(t)
	server := NewServer(":0", m)

	req := httptest.NewRequest(http.MethodGet, "/auth", nil)
	req.Header.Set("Authorization", "Bearer some-token")
	w := httptest.NewRecorder()
	server.handleAuth(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
