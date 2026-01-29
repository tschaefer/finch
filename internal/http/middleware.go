/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package http

import (
	"net/http"
	"strings"
)

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' 'unsafe-inline'; "+
				"style-src 'self' 'unsafe-inline'; "+
				"connect-src 'self' ws: wss:; "+
				"img-src 'self' data:; "+
				"font-src 'self'; "+
				"frame-ancestors 'none'; "+
				"base-uri 'self'; "+
				"form-action 'self'")

		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		w.Header().Set("Permissions-Policy",
			"camera=(), "+
				"microphone=(), "+
				"geolocation=(), "+
				"payment=()")

		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var token string

		auth := r.Header.Get("Authorization")
		if auth != "" {
			const prefix = "Bearer "
			if !strings.HasPrefix(auth, prefix) {
				http.Error(w, "Invalid authorization header", http.StatusUnauthorized)
				return
			}
			token = strings.TrimPrefix(auth, prefix)
		} else {
			cookie, err := r.Cookie("dashboard_token")
			if err == nil {
				token = cookie.Value
			} else {
				token = r.URL.Query().Get("token")
			}
		}

		if token == "" {
			if r.URL.Path != "/login" {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			next.ServeHTTP(w, r)
			return
		}

		if err := s.controller.ValidateDashboardToken(token); err != nil {
			http.SetCookie(w, &http.Cookie{
				Name:   "dashboard_token",
				Value:  "",
				Path:   "/",
				MaxAge: -1,
			})

			if r.URL.Path == "/ws" {
				http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			} else {
				http.Redirect(w, r, "/login?error=expired", http.StatusSeeOther)
			}
			return
		}

		next.ServeHTTP(w, r)
	})
}
