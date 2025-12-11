/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tschaefer/finch/internal/config"
)

func Test_NewReturnsError_InvalidUrlSchema(t *testing.T) {
	cfg := config.NewFromData(&config.Data{
		Database: "psql://user:pass@localhost/dbname",
	}, "")

	_, err := New(cfg)
	wanted := "unsupported database scheme: psql"
	assert.EqualError(t, err, wanted, "error message")
}

func Test_NewReturnsError_PathNotExist(t *testing.T) {
	cfg := config.NewFromData(&config.Data{
		Database: "sqlite:///nonexistent/path/to/database.db",
	}, "")

	_, err := New(cfg)
	wanted := "unable to open database file: no such file or directory"
	assert.EqualError(t, err, wanted, "error message")
}

func Test_ConnectionReturnsGormDB(t *testing.T) {
	cfg := config.NewFromData(&config.Data{
		Database: "sqlite://:memory:",
	}, "")

	db, err := New(cfg)
	assert.NoError(t, err, "new database instance")
	assert.NotNil(t, db, "database instance")

	connection := db.Connection()
	assert.NotNil(t, connection, "database connection")
	assert.IsType(t, connection, connection, "database connection type")
}

func Test_MigrateSucceeds(t *testing.T) {
	cfg := config.NewFromData(&config.Data{
		Database: "sqlite://:memory:",
	}, "")

	db, err := New(cfg)
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

	assert.Equal(t, len(results), len(columns), "agents table should have correct number of columns")
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
