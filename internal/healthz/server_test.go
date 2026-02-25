/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package healthz

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tschaefer/finch/internal/config"
	"github.com/tschaefer/finch/internal/database"
	"github.com/tschaefer/finch/internal/version"
)

var testCfg = config.NewFromData(&config.Data{
	Id:        "test-id",
	Hostname:  "localhost",
	CreatedAt: "2025-01-01T00:00:00Z",
	Database:  "sqlite:///:memory:",
	Secret:    "gpFb8WTh5iELimbX3YfuvRYRh2Z2PHa8Lmoog0a25QQ=",
}, "")

func newTestDB(t *testing.T) *database.Database {
	t.Helper()
	db, err := database.New(testCfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(); err != nil {
		t.Fatal(err)
	}
	return db
}

func Test_HealthzHandler_Healthy(t *testing.T) {
	s := NewServer("127.0.0.1:0", newTestDB(t))

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	s.handleHealthz(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, version.Commit(), rec.Header().Get("X-Finch-Commit"))
	assert.Equal(t, version.Release(), rec.Header().Get("X-Finch-Release"))
}

func Test_HealthzHandler_Unhealthy(t *testing.T) {
	db := newTestDB(t)

	sqlDB, err := db.Connection().DB()
	if err != nil {
		t.Fatal(err)
	}
	_ = sqlDB.Close()

	s := NewServer("127.0.0.1:0", db)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	s.handleHealthz(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, version.Commit(), rec.Header().Get("X-Finch-Commit"))
	assert.Equal(t, version.Release(), rec.Header().Get("X-Finch-Release"))
}

func Test_HealthzHandler_MethodNotAllowed(t *testing.T) {
	s := NewServer("127.0.0.1:0", newTestDB(t))

	req := httptest.NewRequest(http.MethodPost, "/healthz", nil)
	rec := httptest.NewRecorder()

	s.handleHealthz(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}
