/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package manager

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/tschaefer/finch/internal/config"
	"github.com/tschaefer/finch/internal/controller"
	"github.com/tschaefer/finch/internal/database"
	"github.com/tschaefer/finch/internal/handler"
	"github.com/tschaefer/finch/internal/model"
	"github.com/tschaefer/finch/internal/profiler"
	"github.com/tschaefer/finch/internal/version"
)

type Manager interface {
	Run(listenAddr string)
}

type manager struct {
	config     config.Config
	database   database.Database
	model      model.Model
	controller controller.Controller
	profiler   profiler.Profiler
}

func New(cfgFile string) (Manager, error) {
	slog.Debug("Initializing Manager", "cfgFile", cfgFile)

	cfg, err := config.Read(cfgFile)
	if err != nil {
		return nil, err
	}

	profiler := profiler.New(cfg, false)
	if err := profiler.Start(); err != nil {
		slog.Warn("Failed to start Pyroscope profiler", "error", err)
	}

	db, err := database.New(cfg)
	if err != nil {
		return nil, err
	}

	if err := db.Migrate(); err != nil {
		return nil, err
	}

	model := model.New(db.Connection())
	ctrl := controller.New(model, cfg)

	return &manager{
		config:     cfg,
		database:   db,
		model:      model,
		controller: ctrl,
		profiler:   profiler,
	}, nil
}

func (m *manager) Run(listenAddr string) {
	slog.Debug("Running Manager", "listenAddr", listenAddr)

	router := handler.New(m.controller, m.config).Router()

	server := &http.Server{
		Addr:         listenAddr,
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      router,
	}

	slog.Info("Starting Finch management server.", "release", version.Release(), "commit", version.Commit())
	slog.Info("Listening on " + listenAddr)
	if err := server.ListenAndServe(); err != nil {
		slog.Error("Error starting server: " + err.Error())
		os.Exit(1)
	}
}
