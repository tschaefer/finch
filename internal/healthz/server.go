/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package healthz

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/tschaefer/finch/internal/database"
	"github.com/tschaefer/finch/internal/version"
)

type Server struct {
	server *http.Server
	db     *database.Database
}

func NewServer(addr string, db *database.Database) *Server {
	slog.Debug("Initializing Healthz Server", "addr", addr)

	s := &Server{db: db}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)

	s.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	return s
}

func (s *Server) Start() error {
	listen, err := net.Listen("tcp", s.server.Addr)
	if err != nil {
		return err
	}

	go func() {
		if err := s.server.Serve(listen); err != nil && err != http.ErrServerClosed {
			slog.Error("healthz server error", "error", err)
		}
	}()

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("X-Finch-Commit", version.Commit())
	w.Header().Set("X-Finch-Release", version.Release())

	ctx, cancel := context.WithTimeout(r.Context(), time.Second)
	defer cancel()

	if err := s.db.Ping(ctx); err != nil {
		slog.Error("database ping failed", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
