/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package manager

import (
	"context"
	"log/slog"
	"net"
	"os"
	"time"

	"github.com/tschaefer/finch/api"
	"github.com/tschaefer/finch/internal/auth"
	"github.com/tschaefer/finch/internal/config"
	"github.com/tschaefer/finch/internal/controller"
	"github.com/tschaefer/finch/internal/database"
	grpcserver "github.com/tschaefer/finch/internal/grpc"
	httpserver "github.com/tschaefer/finch/internal/http"
	"github.com/tschaefer/finch/internal/model"
	"github.com/tschaefer/finch/internal/profiler"
	"github.com/tschaefer/finch/internal/version"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Manager struct {
	config     *config.Config
	database   *database.Database
	model      *model.Model
	controller *controller.Controller
	profiler   *profiler.Profiler
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

func (m *Manager) Run(ctx context.Context, grpcAddr string, httpAddr string, authAddr string) {
	slog.Debug("Running Manager", "grpcAddr", grpcAddr, "httpAddr", httpAddr, "authAddr", authAddr)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	slog.Info("Starting Finch management server", "release", version.Release(), "commit", version.Commit())
	slog.Info("Listening on " + grpcAddr + " (gRPC)")
	slog.Info("Listening on " + httpAddr + " (HTTP)")
	slog.Info("Listening on " + authAddr + " (Auth)")

	grpcServer, err := m.runGRPCServer(grpcAddr)
	if err != nil {
		slog.Error("Failed to start gRPC server", "error", err)
		os.Exit(1)
	}

	httpServer, err := m.runHTTPServer(httpAddr)
	if err != nil {
		slog.Error("Failed to start HTTP server", "error", err)
		os.Exit(1)
	}

	authServer, err := m.runAuthServer(authAddr)
	if err != nil {
		slog.Error("Failed to start Auth server", "error", err)
		os.Exit(1)
	}

	<-ctx.Done()
	slog.Info("Shutting down servers...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := authServer.Stop(shutdownCtx); err != nil {
		slog.Error("Auth server shutdown error", "error", err)
	}

	if err := httpServer.Stop(shutdownCtx); err != nil {
		slog.Error("HTTP server shutdown error", "error", err)
	}

	grpcServer.GracefulStop()
	slog.Info("Servers stopped")
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

	dashboardServer := grpcserver.NewDashboardServer(m.controller)
	api.RegisterDashboardServiceServer(grpcServer, dashboardServer)

	reflection.Register(grpcServer)

	go func() {
		if err := grpcServer.Serve(listen); err != nil && err != grpc.ErrServerStopped {
			slog.Error("Server error: " + err.Error())
			os.Exit(1)
		}
	}()

	return grpcServer, nil
}

func (m *Manager) runHTTPServer(httpAddr string) (*httpserver.Server, error) {
	httpServer := httpserver.NewServer(httpAddr, m.controller, m.config)
	if err := httpServer.Start(); err != nil {
		return nil, err
	}
	return httpServer, nil
}

func (m *Manager) runAuthServer(authAddr string) (*auth.Server, error) {
	authServer := auth.NewServer(authAddr, m.model)
	if err := authServer.Start(); err != nil {
		return nil, err
	}
	return authServer, nil
}
