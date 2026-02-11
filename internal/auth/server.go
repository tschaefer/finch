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

	"github.com/golang-jwt/jwt/v5"

	"github.com/tschaefer/finch/internal/config"
	"github.com/tschaefer/finch/internal/model"
)

type Server struct {
	model  *model.Model
	config *config.Config
	server *http.Server
}

func NewServer(addr string, model *model.Model, cfg *config.Config) *Server {
	slog.Debug("Initializing Auth Server", "addr", addr)

	mux := http.NewServeMux()

	s := &Server{
		model:  model,
		config: cfg,
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

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(s.config.Secret()), nil
	})

	if err != nil || !token.Valid {
		s.log(r, slog.LevelWarn, "Auth request has invalid or expired token")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		s.log(r, slog.LevelWarn, "Auth request has invalid token claims")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	resourceId, ok := claims["rid"].(string)
	if !ok {
		s.log(r, slog.LevelWarn, "Auth request missing rid claim")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	agent := &model.Agent{ResourceId: resourceId}
	_, err = s.model.GetAgent(agent)
	if err != nil {
		s.log(r, slog.LevelWarn, "Auth request for unknown agent", "rid", resourceId)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	s.log(r, slog.LevelDebug, "Auth request successful", "rid", resourceId)
	w.WriteHeader(http.StatusOK)
}
