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

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	slog.Info("Starting Finch management server", "release", version.Release(), "commit", version.Commit())
	slog.Info("Listening on " + listenAddr)

	go m.runGRPCServer(listenAddr)

	<-stop
	slog.Info("Shutting down server...")
}

func (m *manager) runGRPCServer(listenAddr string) {
	listen, err := net.Listen("tcp", listenAddr)
	if err != nil {
		slog.Error("Failed to listen for gRPC: " + err.Error())
		os.Exit(1)
	}

	authInterceptor := grpcserver.NewAuthInterceptor(m.config)
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(authInterceptor.Unary()),
	)

	agentServer := grpcserver.NewAgentServer(m.controller, m.config)
	api.RegisterAgentServiceServer(grpcServer, agentServer)

	infoServer := grpcserver.NewInfoServer(m.config)
	api.RegisterInfoServiceServer(grpcServer, infoServer)

	reflection.Register(grpcServer)

	if err := grpcServer.Serve(listen); err != nil {
		slog.Error("gRPC server error: " + err.Error())
		os.Exit(1)
	}
}
