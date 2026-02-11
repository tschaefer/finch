/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package controller

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const defaultTokenExpiration = 365 * 24 * time.Hour

func (c *Controller) GenerateAgentToken(resourceId string, expiration time.Duration) (string, time.Time, error) {
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
