/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package auth

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/tschaefer/finch/internal/config"
	"github.com/tschaefer/finch/internal/controller"
)

type Server struct {
	controller *controller.Controller
	config     *config.Config
	server     *http.Server
}

func NewServer(addr string, ctrl *controller.Controller, cfg *config.Config) *Server {
	slog.Debug("Initializing Auth Server", "addr", addr)

	mux := http.NewServeMux()

	s := &Server{
		controller: ctrl,
		config:     cfg,
		server: &http.Server{
			Addr:         addr,
			Handler:      mux,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}

	mux.HandleFunc("/auth", s.handleAuth)

	return s
}

func (s *Server) Start() error {
	listen, err := net.Listen("tcp", s.server.Addr)
	if err != nil {
		return err
	}

	go func() {
		if err := s.server.Serve(listen); err != nil && err != http.ErrServerClosed {
			slog.Error("Auth server error", "error", err)
		}
	}()

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *Server) handleAuth(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		s.log(r, slog.LevelWarn, "Auth request missing Authorization header")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		s.log(r, slog.LevelWarn, "Auth request has invalid Authorization header format")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	err := s.controller.ValidateAgentToken(tokenString)
	if err != nil {
		s.log(r, slog.LevelWarn, "Auth request failed validation", "error", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	s.log(r, slog.LevelDebug, "Auth request succeeded")
	w.WriteHeader(http.StatusOK)
}
