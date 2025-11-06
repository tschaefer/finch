/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/tschaefer/finch/internal/controller"
)

func (h *handler) registerAgentHandlers() {
	h.router.HandleFunc("/api/v1/agent", h.ListAgents).Methods(http.MethodGet)
	h.router.HandleFunc("/api/v1/agent", h.CreateAgent).Methods(http.MethodPost)
	h.router.HandleFunc("/api/v1/agent/{rid}", h.GetAgent).Methods(http.MethodGet)
	h.router.HandleFunc("/api/v1/agent/{rid}", h.DeleteAgent).Methods(http.MethodDelete)
	h.router.HandleFunc("/api/v1/agent/{rid}/config", h.GetAgentConfig).Methods(http.MethodGet)
}

func (h *handler) CreateAgent(w http.ResponseWriter, r *http.Request) {
	var p controller.Agent

	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		go h.makeLog(r, http.StatusBadRequest, slog.LevelError, "invalid request body")
		h.makeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	rid, err := h.controller.RegisterAgent(&p)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, controller.ErrAgentAlreadyExists) {
			status = http.StatusConflict
		}
		go h.makeLog(r, status, slog.LevelError, err.Error())
		h.makeError(w, status, err.Error())
		return
	}

	go h.makeLog(r, http.StatusCreated, slog.LevelInfo, fmt.Sprintf("agent %s created", rid))
	h.makeResponse(w, http.StatusCreated, map[string]string{"rid": rid})
}

func (h *handler) DeleteAgent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	rid := vars["rid"]

	if err := h.controller.DeregisterAgent(rid); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, controller.ErrAgentNotFound) {
			status = http.StatusNotFound
		}
		go h.makeLog(r, status, slog.LevelError, err.Error())
		h.makeError(w, status, err.Error())
		return
	}

	go h.makeLog(r, http.StatusNoContent, slog.LevelInfo, fmt.Sprintf("agent %s deleted", rid))
	h.makeResponse(w, http.StatusNoContent, nil)
}

func (h *handler) GetAgent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	rid := vars["rid"]

	agent, err := h.controller.GetAgent(rid)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, controller.ErrAgentNotFound) {
			status = http.StatusNotFound
		}
		go h.makeLog(r, status, slog.LevelError, err.Error())
		h.makeError(w, status, err.Error())
		return
	}

	go h.makeLog(r, http.StatusOK, slog.LevelInfo, fmt.Sprintf("agent %s retrieved", rid))
	h.makeResponse(w, http.StatusOK, agent)
}

func (h *handler) GetAgentConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	rid := vars["rid"]

	config, err := h.controller.CreateAgentConfig(rid)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, controller.ErrAgentNotFound) {
			status = http.StatusNotFound
		}
		go h.makeLog(r, status, slog.LevelError, err.Error())
		h.makeError(w, status, err.Error())
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.cfg\"", rid))
	w.Header().Set("Content-Type", "application/octet-stream")
	go h.makeLog(r, http.StatusOK, slog.LevelInfo, fmt.Sprintf("config for agent %s retrieved", rid))
	http.ServeContent(w, r, fmt.Sprintf("%s.cfg", rid), time.Now(), bytes.NewReader(config))
}

func (h *handler) ListAgents(w http.ResponseWriter, r *http.Request) {
	agents, err := h.controller.ListAgents()
	if err != nil {
		go h.makeLog(r, http.StatusInternalServerError, slog.LevelError, err.Error())
		h.makeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	go h.makeLog(r, http.StatusOK, slog.LevelInfo, "agents listed")
	h.makeResponse(w, http.StatusOK, agents)
}
