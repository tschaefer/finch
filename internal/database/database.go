/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package database

import (
	"fmt"
	"log/slog"
	"net/url"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/tschaefer/finch/internal/config"
	"github.com/tschaefer/finch/internal/model"
)

type Database interface {
	Connection() *gorm.DB
	Migrate() error
}

type database struct {
	connection *gorm.DB
}

func New(config config.Config) (Database, error) {
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
	default:
		return nil, fmt.Errorf("unsupported database scheme: %s", uri.Scheme)
	}
	if err != nil {
		return nil, err
	}

	return &database{
		connection: connection,
	}, nil
}

func (d *database) Connection() *gorm.DB {
	slog.Debug("Retrieving database connection")

	return d.connection
}

func (d *database) Migrate() error {
	slog.Debug("Migrating database schema")

	return d.connection.AutoMigrate(&model.Agent{})
}
