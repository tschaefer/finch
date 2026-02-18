/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package controller

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/tschaefer/finch/internal/model"
)

const defaultTokenExpiration = 365 * 24 * time.Hour

func (c *Controller) GenerateAgentToken(resourceId string, expiration time.Duration) (string, time.Time, error) {
	slog.Debug("Generating agent token", "resourceId", resourceId, "expiration", expiration)

	if expiration == 0 {
		expiration = defaultTokenExpiration
	}

	now := time.Now()
	expiresAt := now.Add(expiration)
	claims := jwt.MapClaims{
		"iss": "finch",
		"sub": "agent",
		"rid": resourceId,
		"iat": now.Unix(),
		"exp": expiresAt.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(c.config.Secret()))
	return tokenString, expiresAt, err
}

func (c *Controller) ValidateAgentToken(tokenString string) error {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(c.config.Secret()), nil
	})

	if err != nil {
		return fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if claims["iss"] != "finch" || claims["sub"] != "agent" {
			return fmt.Errorf("invalid token claims")
		}

		resourceId, ok := claims["rid"].(string)
		if !ok {
			return fmt.Errorf("missing rid claim")
		}

		agent := &model.Agent{ResourceId: resourceId}
		_, err := c.model.GetAgent(agent)
		if err != nil {
			return fmt.Errorf("unknown agent: %s", resourceId)
		}

		return nil
	}

	return fmt.Errorf("invalid token")
}
