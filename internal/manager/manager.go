/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package manager

import (
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/tschaefer/finch/api"
	"github.com/tschaefer/finch/internal/config"
	"github.com/tschaefer/finch/internal/controller"
	"github.com/tschaefer/finch/internal/database"
	grpcserver "github.com/tschaefer/finch/internal/grpc"
	"github.com/tschaefer/finch/internal/model"
	"github.com/tschaefer/finch/internal/profiler"
	"github.com/tschaefer/finch/internal/version"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Manager struct {
	config     *config.Config
	database   database.Database
	model      model.Model
	controller controller.Controller
	profiler   profiler.Profiler
}

func New(cfgFile string) (*Manager, error) {
	slog.Debug("Initializing Manager", "cfgFile", cfgFile)

	cfg, err := config.NewFromFile(cfgFile)
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

	return &Manager{
		config:     cfg,
		database:   db,
		model:      model,
		controller: ctrl,
		profiler:   profiler,
	}, nil
}

func (m *Manager) Run(listenAddr string) {
	slog.Debug("Running Manager", "listenAddr", listenAddr)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	slog.Info("Starting Finch management server", "release", version.Release(), "commit", version.Commit())
	slog.Info("Listening on " + listenAddr)

	grpcServer, err := m.runGRPCServer(listenAddr)
	if err != nil {
		slog.Error("Failed to start server", "error", err)
		os.Exit(1)
	}

	<-stop
	slog.Info("Shutting down server...")

	grpcServer.GracefulStop()
	slog.Info("Server stopped")
}

func (m *Manager) runGRPCServer(listenAddr string) (*grpc.Server, error) {
	listen, err := net.Listen("tcp", listenAddr)
	if err != nil {
		slog.Error("Failed to listen: " + err.Error())
		return nil, err
	}

	authInterceptor := grpcserver.NewAuthInterceptor(m.config)
	headersInterceptor := grpcserver.NewHeadersInterceptor()
	loggingInterceptor := grpcserver.NewLoggingInterceptor()
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			loggingInterceptor.Unary(),
			authInterceptor.Unary(),
			headersInterceptor.Unary(),
		),
	)

	agentServer := grpcserver.NewAgentServer(m.controller, m.config)
	api.RegisterAgentServiceServer(grpcServer, agentServer)

	infoServer := grpcserver.NewInfoServer(m.config)
	api.RegisterInfoServiceServer(grpcServer, infoServer)

	reflection.Register(grpcServer)

	go func() {
		if err := grpcServer.Serve(listen); err != nil && err != grpc.ErrServerStopped {
			slog.Error("Server error: " + err.Error())
			os.Exit(1)
		}
	}()

	return grpcServer, nil
}
