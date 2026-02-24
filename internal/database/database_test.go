/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package database

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tschaefer/finch/internal/config"
	"github.com/tschaefer/finch/internal/model"
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
		Database: "sqlite:///:memory:",
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
		Database: "sqlite:///:memory:",
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
		"profiles",
		"registered_at",
		"resource_id",
		"updated_at",
		"node",
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

func Test_MigrateSucceeds_RenamingTagsToLabels(t *testing.T) {
	cfg := config.NewFromData(&config.Data{
		Database: "sqlite:///:memory:",
	}, "")

	db, err := New(cfg)
	assert.NoError(t, err, "new database instance")

	err = db.Migrate()
	assert.NoError(t, err, "migrate database")

	db.Connection().Exec("ALTER TABLE agents RENAME COLUMN labels TO tags;")

	err = db.Migrate()
	assert.NoError(t, err, "migrate database with existing column 'tags'")

	exists := db.Connection().Migrator().HasColumn(&model.Agent{}, "labels")
	assert.True(t, exists, "column 'labels' should exist after migration")
}

func Test_MigrateSucceeds_RemovesCredentialsColumns(t *testing.T) {
	cfg := config.NewFromData(&config.Data{
		Database: "sqlite:///:memory:",
	}, "")

	db, err := New(cfg)
	assert.NoError(t, err, "new database instance")

	err = db.Migrate()
	assert.NoError(t, err, "first migration")

	columns := []string{"password", "password_hash", "username"}
	for _, column := range columns {
		result := db.Connection().Exec("ALTER TABLE agents ADD COLUMN " + column + " TEXT;")
		assert.NoError(t, result.Error, "add column "+column)
	}

	for _, column := range columns {
		has := db.Connection().Migrator().HasColumn(&model.Agent{}, column)
		assert.True(t, has, "column '"+column+"' should exist before second migration")
	}

	err = db.Migrate()
	assert.NoError(t, err, "second migration with credentials columns")

	for _, column := range columns {
		has := db.Connection().Migrator().HasColumn(&model.Agent{}, column)
		assert.False(t, has, "column '"+column+"' should be removed by second migration")
	}
}

func Test_PingSucceeds(t *testing.T) {
	cfg := config.NewFromData(&config.Data{
		Database: "sqlite:///:memory:",
	}, "")

	db, err := New(cfg)
	assert.NoError(t, err, "new database instance")

	err = db.Migrate()
	assert.NoError(t, err, "migrate database")

	err = db.Ping(context.Background())
	assert.NoError(t, err, "ping database")
}

func Test_PingReturnsError_WritePermissionDenied(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "testdb-*.db")
	assert.NoError(t, err, "create temporary file")
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	cfg := config.NewFromData(&config.Data{
		Database: "sqlite:///" + tmpFile.Name(),
	}, "")

	db, err := New(cfg)
	assert.NoError(t, err, "new database instance")

	err = os.Chmod(tmpFile.Name(), 0400)
	assert.NoError(t, err, "change file permissions")

	err = db.Ping(context.Background())
	assert.Error(t, err, "ping database with read-only permissions")
}
