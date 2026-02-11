/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package database

import (
	"fmt"
	"log/slog"
	"net/url"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/tschaefer/finch/internal/config"
	"github.com/tschaefer/finch/internal/model"
)

type Database struct {
	connection *gorm.DB
}

func New(config *config.Config) (*Database, error) {
	slog.Debug("Initializing database", "config", fmt.Sprintf("%+v", config))

	uri, err := url.Parse(config.Database())
	if err != nil {
		return nil, err
	}

	var connection *gorm.DB
	dbcfg := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}

	if uri.Path == "" && uri.Host == "" {
		return nil, fmt.Errorf("no database target specified in URI: %s", uri.String())
	}

	switch uri.Scheme {
	case "sqlite":
		var path string

		if uri.Path != "" {
			path = uri.Path
		} else if uri.Host == ":memory:" {
			path = uri.Host
		} else {
			path = fmt.Sprintf("%s/%s", config.Library(), uri.Host)
		}

		connection, err = gorm.Open(sqlite.Open(path), dbcfg)
	case "postgres", "postgresql":
		connection, err = gorm.Open(postgres.Open(uri.String()), dbcfg)
	default:
		return nil, fmt.Errorf("unsupported database scheme: %s", uri.Scheme)
	}
	if err != nil {
		return nil, err
	}

	return &Database{
		connection: connection,
	}, nil
}

func (d *Database) Connection() *gorm.DB {
	slog.Debug("Retrieving database connection")

	return d.connection
}

func (d *Database) Migrate() error {
	slog.Debug("Migrating database schema")

	if d.connection.Migrator().HasColumn(&model.Agent{}, "tags") {
		sql := "ALTER TABLE agents RENAME COLUMN tags TO labels"
		if err := d.connection.Exec(sql).Error; err != nil {
			return err
		}
	}

	columnsToRemove := []string{"password", "password_hash", "username"}
	for _, column := range columnsToRemove {
		if d.connection.Migrator().HasColumn(&model.Agent{}, column) {
			sql := fmt.Sprintf("ALTER TABLE agents DROP COLUMN %s", column)
			if err := d.connection.Exec(sql).Error; err != nil {
				return err
			}
		}
	}

	if err := d.connection.AutoMigrate(&model.Agent{}); err != nil {
		return err
	}

	return nil
}
