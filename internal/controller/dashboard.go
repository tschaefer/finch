/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrInvalidRole  = errors.New("invalid role")
)

const (
	RoleAdmin    = "admin"
	RoleOperator = "operator"
	RoleViewer   = "viewer"
)

type DashboardTokenResponse struct {
	Token        string
	ExpiresAt    time.Time
	DashboardURL string
}

type DashboardClaims struct {
	Role  string
	Scope []string
}

func (c *Controller) GenerateDashboardToken(sessionTimeout int, role string, scope []string) (*DashboardTokenResponse, error) {
	slog.Debug("Generating dashboard token", "sessionTimeout", sessionTimeout, "role", role, "scope", scope)

	if sessionTimeout <= 0 {
		sessionTimeout = 1800
	}

	if role != RoleAdmin && role != RoleOperator && role != RoleViewer {
		return nil, ErrInvalidRole
	}

	scopeJSON, err := json.Marshal(scope)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal scope: %w", err)
	}

	expiresAt := time.Now().Add(time.Duration(sessionTimeout) * time.Second)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss":   "finch",
		"sub":   "dashboard",
		"exp":   expiresAt.Unix(),
		"iat":   time.Now().Unix(),
		"role":  role,
		"scope": string(scopeJSON),
	})

	tokenString, err := token.SignedString([]byte(c.config.Secret()))
	if err != nil {
		return nil, err
	}

	dashboardURL := fmt.Sprintf("https://%s/login", c.config.Hostname())

	return &DashboardTokenResponse{
		Token:        tokenString,
		ExpiresAt:    expiresAt,
		DashboardURL: dashboardURL,
	}, nil
}

func (c *Controller) ValidateDashboardToken(tokenString string) (*DashboardClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(c.config.Secret()), nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if claims["iss"] != "finch" || claims["sub"] != "dashboard" {
			return nil, ErrInvalidToken
		}

		role, _ := claims["role"].(string)
		scopeStr, _ := claims["scope"].(string)

		var scope []string
		if scopeStr != "" {
			if err := json.Unmarshal([]byte(scopeStr), &scope); err != nil {
				return nil, fmt.Errorf("failed to unmarshal scope: %w", err)
			}
		}

		if role != RoleAdmin && role != RoleOperator && role != RoleViewer {
			return nil, ErrInvalidRole
		}

		return &DashboardClaims{
			Role:  role,
			Scope: scope,
		}, nil
	}

	return nil, ErrInvalidToken
}

func (c *Controller) CanViewTokens(claims *DashboardClaims) bool {
	return claims.Role == RoleAdmin || claims.Role == RoleOperator
}

func (c *Controller) CanDownloadConfig(claims *DashboardClaims) bool {
	return claims.Role == RoleAdmin
}

func (c *Controller) CanAccessAgent(claims *DashboardClaims, agentRID, agentHostname string) bool {
	if len(claims.Scope) == 0 {
		return true
	}

	for _, s := range claims.Scope {
		if s == agentRID || s == agentHostname {
			return true
		}
	}
	return false
}
