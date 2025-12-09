/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockConfig struct {
	version   string
	hostname  string
	database  string
	profiler  string
	id        string
	createdAt string
	library   string
	secret    string
	username  string
	password  string
}

func (m *mockConfig) Version() string               { return m.version }
func (m *mockConfig) Hostname() string              { return m.hostname }
func (m *mockConfig) Database() string              { return m.database }
func (m *mockConfig) Profiler() string              { return m.profiler }
func (m *mockConfig) Id() string                    { return m.id }
func (m *mockConfig) CreatedAt() string             { return m.createdAt }
func (m *mockConfig) Library() string               { return m.library }
func (m *mockConfig) Secret() string                { return m.secret }
func (m *mockConfig) Credentials() (string, string) { return m.username, m.password }

func Test_NewReturnsError_InvalidUrlSchema(t *testing.T) {
	mockedConfig := mockConfig{
		database: "psql://user:pass@localhost/dbname",
	}

	_, err := New(&mockedConfig)
	wanted := "unsupported database scheme: psql"
	assert.EqualError(t, err, wanted, "error message")
}

func Test_NewReturnsError_PathNotExist(t *testing.T) {
	mockedConfig := mockConfig{
		database: "sqlite:///nonexistent/path/to/database.db",
	}

	_, err := New(&mockedConfig)
	wanted := "unable to open database file: no such file or directory"
	assert.EqualError(t, err, wanted, "error message")
}

func Test_ConnectionReturnsGormDB(t *testing.T) {
	mockedConfig := mockConfig{
		database: "sqlite://:memory:",
	}

	db, err := New(&mockedConfig)
	assert.NoError(t, err, "new database instance")
	assert.NotNil(t, db, "database instance")

	connection := db.Connection()
	assert.NotNil(t, connection, "database connection")
	assert.IsType(t, connection, connection, "database connection type")
}

func Test_MigrateSucceeds(t *testing.T) {
	mockedConfig := mockConfig{
		database: "sqlite://:memory:",
	}

	db, err := New(&mockedConfig)
	assert.NoError(t, err, "new database instance")

	err = db.Migrate()
	assert.NoError(t, err, "migrate database")

	connection := db.Connection()
	type result struct {
		Name string
	}
	var results []result
	err = connection.Raw("PRAGMA table_info(agents);").Scan(&results).Error
	assert.NoError(t, err, "query table info")

	columns := []string{
		"active",
		"created_at",
		"hostname",
		"id",
		"labels",
		"last_seen",
		"log_sources",
		"metrics",
		"metrics_targets",
		"password",
		"password_hash",
		"profiles",
		"registered_at",
		"resource_id",
		"updated_at",
		"username",
	}

	assert.GreaterOrEqual(t, len(results), len(columns), "agents table should have at least the expected number of columns")
	for _, column := range columns {
		found := false
		for _, row := range results {
			if row.Name == column {
				found = true
				break
			}
		}
		assert.True(t, found, "column "+column+" should exist in agents table")
	}
}
