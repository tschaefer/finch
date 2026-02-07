/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package http

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tschaefer/finch/internal/version"
)

//go:embed templates/*
var templateFiles embed.FS

var templates *template.Template

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WSMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

type WSResponse struct {
	Type string `json:"type"`
	HTML string `json:"html"`
}

func init() {
	var err error
	templates, err = template.ParseFS(templateFiles, "templates/*.html")
	if err != nil {
		panic(fmt.Sprintf("Failed to parse templates: %v", err))
	}
}

type AgentData struct {
	ResourceID     string
	Hostname       string
	Labels         []string
	LogSources     []string
	Metrics        bool
	MetricsTargets []string
	Profiles       bool
	RegisteredAt   string
	Active         bool
}

type AgentListData struct {
	Agents      []AgentData
	Page        int
	TotalPages  int
	TotalAgents int
	PrevPage    int
	NextPage    int
	Search      string
}

type StatsData struct {
	TotalAgents     int
	MetricsEnabled  int
	ProfilesEnabled int
}

type ServiceInfoData struct {
	Hostname string
	Release  string
	Commit   string
}

type CredentialsData struct {
	Username   string
	Password   string
	ResourceID string
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var token, errorMsg string
	switch r.Method {
	case http.MethodGet:
		token = r.URL.Query().Get("token")
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		token = r.FormValue("token")
		errorMsg = r.FormValue("error")
	}
	token = strings.TrimSpace(token)

	if token == "" {
		data := map[string]string{}
		if errorMsg == "expired" {
			data["Error"] = "expired"
		}
		if err := templates.ExecuteTemplate(w, "login.html", data); err != nil {
			slog.Error("Failed to render login", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	if err := s.controller.ValidateDashboardToken(token); err != nil {
		data := map[string]string{"Error": "invalid"}
		if err := templates.ExecuteTemplate(w, "login.html", data); err != nil {
			slog.Error("Failed to render login", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	cookie := &http.Cookie{
		Name:     "dashboard_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   0,
	}
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		cookie.Secure = true
	}
	http.SetCookie(w, cookie)

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	cookie := &http.Cookie{
		Name:     "dashboard_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	}
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		cookie.Secure = true
	}
	http.SetCookie(w, cookie)

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if err := templates.ExecuteTemplate(w, "dashboard.html", nil); err != nil {
		slog.Error("Failed to render dashboard", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("dashboard_token")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	token := cookie.Value

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("Failed to upgrade WebSocket", "error", err)
		return
	}
	defer func() {
		_ = conn.Close()
	}()

	currentPage := 1
	currentSearch := ""

	s.sendInfoUpdate(conn)
	s.sendStatsUpdate(conn)
	s.sendEndpointsUpdate(conn)
	s.sendAgentsUpdate(conn, currentPage, currentSearch)

	agentEvents := s.controller.SubscribeAgentEvents()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			var msg WSMessage
			if err := conn.ReadJSON(&msg); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					slog.Error("WebSocket read error", "error", err)
				}
				return
			}
			if msg.Type == "get_agents" {
				var params struct {
					Page   int    `json:"page"`
					Search string `json:"search"`
				}
				if err := json.Unmarshal(msg.Data, &params); err == nil {
					if params.Page >= 1 {
						currentPage = params.Page
					}
					currentSearch = params.Search
				}
			}

			s.handleWSMessage(conn, msg)
		}
	}()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.controller.ValidateDashboardToken(token); err != nil {
				_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Token expired"))
				return
			}
		case <-agentEvents:
			s.sendAgentsUpdate(conn, currentPage, currentSearch)
			s.sendStatsUpdate(conn)
		case <-done:
			return
		}
	}
}

func (s *Server) handleWSMessage(conn *websocket.Conn, msg WSMessage) {
	switch msg.Type {
	case "get_agents":
		var params struct {
			Page   int    `json:"page"`
			Search string `json:"search"`
		}
		if err := json.Unmarshal(msg.Data, &params); err == nil {
			if params.Page < 1 {
				params.Page = 1
			}
			s.sendAgentsUpdate(conn, params.Page, params.Search)
		}
	case "get_credentials":
		var params struct {
			RID string `json:"rid"`
		}
		if err := json.Unmarshal(msg.Data, &params); err == nil {
			s.sendCredentials(conn, params.RID)
		}
	case "download_config":
		var params struct {
			RID string `json:"rid"`
		}
		if err := json.Unmarshal(msg.Data, &params); err == nil {
			s.sendConfig(conn, params.RID)
		}
	}
}

func (s *Server) sendAgentsUpdate(conn *websocket.Conn, page int, search string) {
	agentList, err := s.controller.ListAgents()
	if err != nil {
		slog.Error("Failed to list agents", "error", err)
		return
	}

	filtered := []AgentData{}
	for _, a := range agentList {
		agent, err := s.controller.GetAgent(a["rid"])
		if err != nil {
			continue
		}

		agentData := AgentData{
			ResourceID:     agent.ResourceId,
			Hostname:       agent.Hostname,
			Labels:         agent.Labels,
			LogSources:     agent.LogSources,
			Metrics:        agent.Metrics,
			MetricsTargets: agent.MetricsTargets,
			Profiles:       agent.Profiles,
			RegisteredAt:   agent.RegisteredAt.Format("2006-01-02 15:04:05"),
			Active:         true,
		}

		if search == "" {
			filtered = append(filtered, agentData)
			continue
		}

		searchLower := strings.ToLower(search)
		hostnameMatch := strings.Contains(strings.ToLower(agent.Hostname), searchLower)
		ridMatch := strings.Contains(strings.ToLower(agent.ResourceId), searchLower)
		labelsMatch := false
		for _, label := range agent.Labels {
			if strings.Contains(strings.ToLower(label), searchLower) {
				labelsMatch = true
				break
			}
		}
		if hostnameMatch || labelsMatch || ridMatch {
			filtered = append(filtered, agentData)
		}
	}

	perPage := 5
	totalAgents := len(filtered)
	totalPages := int(math.Ceil(float64(totalAgents) / float64(perPage)))

	if totalPages == 0 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}

	start := (page - 1) * perPage
	end := start + perPage
	end = min(end, totalAgents)

	pageAgents := []AgentData{}
	if totalAgents > 0 {
		pageAgents = filtered[start:end]
	}

	data := AgentListData{
		Agents:      pageAgents,
		Page:        page,
		TotalPages:  totalPages,
		TotalAgents: totalAgents,
		PrevPage:    page - 1,
		NextPage:    page + 1,
		Search:      search,
	}

	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "agents.html", data); err != nil {
		slog.Error("Failed to render agents template", "error", err)
		return
	}

	response := WSResponse{
		Type: "agents",
		HTML: buf.String(),
	}
	conn.WriteJSON(response)
}

func (s *Server) sendStatsUpdate(conn *websocket.Conn) {
	agentList, err := s.controller.ListAgents()
	if err != nil {
		slog.Error("Failed to list agents", "error", err)
		return
	}

	stats := StatsData{
		TotalAgents: len(agentList),
	}

	for _, a := range agentList {
		agent, err := s.controller.GetAgent(a["rid"])
		if err != nil {
			continue
		}
		if agent.Metrics {
			stats.MetricsEnabled++
		}
		if agent.Profiles {
			stats.ProfilesEnabled++
		}
	}

	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "stats.html", stats); err != nil {
		slog.Error("Failed to render stats template", "error", err)
		return
	}

	response := WSResponse{
		Type: "stats",
		HTML: buf.String(),
	}
	conn.WriteJSON(response)
}

func (s *Server) sendInfoUpdate(conn *websocket.Conn) {
	hostname := s.config.Hostname()

	commit := version.GitCommit
	if len(commit) > 7 {
		commit = commit[:7]
	}

	data := ServiceInfoData{
		Hostname: hostname,
		Release:  version.Version,
		Commit:   commit,
	}

	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "info.html", data); err != nil {
		slog.Error("Failed to render info template", "error", err)
		return
	}

	response := WSResponse{
		Type: "info",
		HTML: buf.String(),
	}
	conn.WriteJSON(response)
}

func (s *Server) sendEndpointsUpdate(conn *websocket.Conn) {
	data := ServiceInfoData{
		Hostname: s.config.Hostname(),
	}

	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "endpoints.html", data); err != nil {
		slog.Error("Failed to render endpoints template", "error", err)
		return
	}

	response := WSResponse{
		Type: "endpoints",
		HTML: buf.String(),
	}
	conn.WriteJSON(response)
}

func (s *Server) sendCredentials(conn *websocket.Conn, rid string) {
	agent, err := s.controller.GetAgent(rid)
	if err != nil {
		slog.Error("Failed to get agent", "rid", rid, "error", err)
		return
	}

	data := CredentialsData{
		Username:   agent.Username,
		Password:   agent.Password,
		ResourceID: rid,
	}

	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "credentials.html", data); err != nil {
		slog.Error("Failed to render credentials template", "error", err)
		return
	}

	response := WSResponse{
		Type: "credentials",
		HTML: buf.String(),
	}
	conn.WriteJSON(response)
}

func (s *Server) sendConfig(conn *websocket.Conn, rid string) {
	config, err := s.controller.CreateAgentConfig(rid)
	if err != nil {
		slog.Error("Failed to create agent config", "rid", rid, "error", err)
		response := map[string]string{
			"type":  "config_error",
			"error": "Failed to generate config",
		}
		conn.WriteJSON(response)
		return
	}

	response := map[string]string{
		"type":     "config",
		"rid":      rid,
		"filename": fmt.Sprintf("alloy-%s.cfg", rid),
		"content":  string(config),
	}
	conn.WriteJSON(response)
}
