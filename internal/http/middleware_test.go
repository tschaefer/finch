/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuthMiddlewareRedirectsToLogin_WhenNoToken(t *testing.T) {
	ctrl := newTestController(t)
	server := NewServer("127.0.0.1:0", ctrl, testCfg)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rec := httptest.NewRecorder()

	middleware := server.authMiddleware(handler)
	middleware.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/login", rec.Header().Get("Location"))
}

func TestAuthMiddlewareRedirectsToLogin_WhenInvalidCookie(t *testing.T) {
	ctrl := newTestController(t)
	server := NewServer("127.0.0.1:0", ctrl, testCfg)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.AddCookie(&http.Cookie{
		Name:  "dashboard_token",
		Value: "invalid",
	})
	rec := httptest.NewRecorder()

	middleware := server.authMiddleware(handler)
	middleware.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/login", rec.Header().Get("Location"))
}

func TestAuthMiddlewareSucceeds_WhenValidCookie(t *testing.T) {
	ctrl := newTestController(t)
	server := NewServer("127.0.0.1:0", ctrl, testCfg)

	resp, err := ctrl.GetDashboardToken(1800)
	assert.NoError(t, err)

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.AddCookie(&http.Cookie{
		Name:  "dashboard_token",
		Value: resp.Token,
	})
	rec := httptest.NewRecorder()

	middleware := server.authMiddleware(handler)
	middleware.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, handlerCalled, "handler should have been called")
}
