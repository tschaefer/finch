/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package http

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"

	"github.com/tschaefer/finch/internal/config"
	"github.com/tschaefer/finch/internal/controller"
)

type Server struct {
	controller *controller.Controller
	config     *config.Config
	ws         *websocket.Upgrader
	server     *http.Server
}

func NewServer(addr string, ctrl *controller.Controller, cfg *config.Config) *Server {
	slog.Debug("Initializing HTTP Server", "addr", addr)

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

	s.ws = &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				slog.Error("WebSocket connection missing Origin header")
				return false
			}

			uri, err := url.Parse(origin)
			if err != nil {
				slog.Error("Invalid WebSocket Origin header", "origin", origin, "error", err)
				return false
			}

			return uri.Hostname() == s.config.Hostname()
		},
	}

	secureLogin := s.responseHeaders(http.HandlerFunc(s.handleLogin))
	mux.Handle("/login", secureLogin)

	secureLogout := s.responseHeaders(s.authMiddleware(http.HandlerFunc(s.handleLogout)))
	mux.Handle("/logout", secureLogout)

	secureDashboard := s.responseHeaders(s.authMiddleware(http.HandlerFunc(s.handleDashboard)))
	mux.Handle("/dashboard", secureDashboard)

	secureWS := s.responseHeaders(s.authMiddleware(http.HandlerFunc(s.handleWebSocket)))
	mux.Handle("/ws", secureWS)

	return s
}

func (s *Server) Start() error {
	listen, err := net.Listen("tcp", s.server.Addr)
	if err != nil {
		return err
	}

	go func() {
		if err := s.server.Serve(listen); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server error", "error", err)
		}
	}()

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
