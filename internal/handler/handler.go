/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/tschaefer/finch/internal/config"
	"github.com/tschaefer/finch/internal/controller"
)

type Handler interface {
	Router() *mux.Router
}

type handler struct {
	controller controller.Controller
	config     config.Config
	router     *mux.Router
}

func New(ctrl controller.Controller, cfg config.Config) Handler {
	slog.Debug("Initializing Handler", "ctrl", fmt.Sprintf("%+v", ctrl), "cfg", fmt.Sprintf("%+v", cfg))

	router := mux.NewRouter()

	return &handler{
		controller: ctrl,
		router:     router,
		config:     cfg,
	}
}

func (h *handler) Router() *mux.Router {
	h.router.NotFoundHandler = http.HandlerFunc(h.notFound)
	h.router.MethodNotAllowedHandler = http.HandlerFunc(h.methodNotAllowed)
	h.router.Use(h.basicAuth)
	h.registerAgentHandlers()

	return h.router
}

func (h *handler) notFound(w http.ResponseWriter, r *http.Request) {
	slog.Debug("Route not found", "path", r.URL.Path)

	h.makeLog(r, http.StatusNotFound, slog.LevelWarn, "route not found")
	h.makeError(w, http.StatusNotFound, "route not found")
}

func (h *handler) methodNotAllowed(w http.ResponseWriter, r *http.Request) {
	h.makeLog(r, http.StatusMethodNotAllowed, slog.LevelWarn, "method not allowed")
	h.makeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func (h *handler) makeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	json.NewEncoder(w).Encode(map[string]string{"detail": message})
}

func (h *handler) makeResponse(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

func (h *handler) basicAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		u, p := h.config.Credentials()

		if !ok || username != u || password != p {
			h.makeLog(r, http.StatusUnauthorized, slog.LevelWarn, "unauthorized")
			h.makeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *handler) makeLog(r *http.Request, status int, level slog.Level, msg string) {
	args := []any{
		slog.String("RemoteAddr", r.RemoteAddr),
		slog.String("UserAgent", r.UserAgent()),
		slog.Int("Status", status),
		slog.String("RequestMethod", r.Method),
		slog.String("RequestPath", r.RequestURI),
	}

	switch level {
	case slog.LevelInfo:
		slog.Info(msg, args...)
	case slog.LevelWarn:
		slog.Warn(msg, args...)
	case slog.LevelError:
		slog.Error(msg, args...)
	default:
		slog.Info(msg, args...)
	}
}
