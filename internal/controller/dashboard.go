/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package controller

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
)

type DashboardTokenResponse struct {
	Token        string
	ExpiresAt    time.Time
	DashboardURL string
}

func (c *Controller) GetDashboardToken(sessionTimeout int) (*DashboardTokenResponse, error) {
	if sessionTimeout <= 0 {
		sessionTimeout = 1800
	}

	expiresAt := time.Now().Add(time.Duration(sessionTimeout) * time.Second)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss": "finch",
		"sub": "dashboard",
		"exp": expiresAt.Unix(),
		"iat": time.Now().Unix(),
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

func (c *Controller) ValidateDashboardToken(tokenString string) error {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(c.config.Secret()), nil
	})

	if err != nil {
		return ErrInvalidToken
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if claims["iss"] != "finch" || claims["sub"] != "dashboard" {
			return ErrInvalidToken
		}

		return nil
	}

	return ErrInvalidToken
}
