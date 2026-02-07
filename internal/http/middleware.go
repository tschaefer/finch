/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package http

import (
	"net/http"
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

		w.Header().Set("Permissions-Policy",
			"camera=(), "+
				"microphone=(), "+
				"geolocation=(), "+
				"payment=()")

		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("dashboard_token")
		if err != nil {
			if r.URL.Path != "/login" {
				if r.URL.Path == "/ws" {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			next.ServeHTTP(w, r)
			return
		}

		if err := s.controller.ValidateDashboardToken(cookie.Value); err != nil {
			clearCookie := &http.Cookie{
				Name:     "dashboard_token",
				Value:    "",
				Path:     "/",
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
				MaxAge:   -1,
			}
			if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
				clearCookie.Secure = true
			}
			http.SetCookie(w, clearCookie)

			if r.URL.Path == "/ws" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}
