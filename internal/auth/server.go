/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package auth

import (
	"context"
	"encoding/base64"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/tschaefer/finch/internal/model"
)

type Server struct {
	model  *model.Model
	server *http.Server
}

func NewServer(addr string, model *model.Model) *Server {
	slog.Debug("Initializing Auth Server", "addr", addr)

	mux := http.NewServeMux()

	s := &Server{
		model: model,
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
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if !strings.HasPrefix(authHeader, "Basic ") {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	encoded := strings.TrimPrefix(authHeader, "Basic ")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	credentials := string(decoded)
	parts := strings.SplitN(credentials, ":", 2)
	if len(parts) != 2 {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	username := parts[0]
	password := parts[1]

	agent := &model.Agent{Username: username}
	_, err = s.model.GetAgent(agent)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(agent.PasswordHash), []byte(password))
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusOK)
}
