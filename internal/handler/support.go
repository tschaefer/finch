/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package handler

import (
	"bytes"
	_ "embed"
	"log/slog"
	"net/http"
	"time"

	"github.com/tschaefer/finch/internal/version"
)

//go:embed openapi.yaml
var openapiSpec []byte

func (h *handler) registerSupportHandlers() {
	h.router.HandleFunc("/api/v1/info", h.GetServiceInfo).Methods(http.MethodGet)
	h.router.HandleFunc("/openapi.yaml", h.GetOpenAPISpec).Methods(http.MethodGet)
}

func (h *handler) GetOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Disposition", "attachment; filename=\"openapi.yaml\"")
	w.Header().Set("Content-Type", "application/octet-stream")

	go h.makeLog(r, http.StatusOK, slog.LevelInfo, "OpenAPI spec retrieved")
	http.ServeContent(w, r, "openapi.yaml", time.Now(), bytes.NewReader(openapiSpec))
}

func (h *handler) GetServiceInfo(w http.ResponseWriter, r *http.Request) {
	info := map[string]string{
		"id":         h.config.Id(),
		"hostname":   h.config.Hostname(),
		"created_at": h.config.CreatedAt(),
		"release":    version.Release(),
		"commit":     version.Commit(),
	}

	go h.makeLog(r, http.StatusOK, slog.LevelInfo, "version info")
	h.makeResponse(w, http.StatusOK, info)
}
