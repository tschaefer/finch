/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package controller

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func Test_GetDashboardTokenReturnsValidToken(t *testing.T) {
	model := newModel(t)
	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	response, err := ctrl.GetDashboardToken(0)

	assert.NoError(t, err, "get dashboard token")
	assert.NotNil(t, response, "response should not be nil")
	assert.NotEmpty(t, response.Token, "token string should not be empty")
	assert.True(t, response.ExpiresAt.After(time.Now()), "expiration should be in future")
	assert.NotEmpty(t, response.DashboardURL, "dashboard URL should not be empty")
}

func Test_GetDashboardTokenUsesDefaultTimeout(t *testing.T) {
	model := newModel(t)
	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	response, err := ctrl.GetDashboardToken(0)
	assert.NoError(t, err, "get dashboard token")

	expectedExpiration := time.Now().Add(1800 * time.Second)
	timeDiff := response.ExpiresAt.Sub(expectedExpiration)
	assert.Less(t, timeDiff.Abs(), 1*time.Second, "expiration should be close to default (1800s)")
}

func Test_GetDashboardTokenUsesCustomTimeout(t *testing.T) {
	model := newModel(t)
	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	customTimeout := 3600 // 1 hour
	response, err := ctrl.GetDashboardToken(customTimeout)
	assert.NoError(t, err, "get dashboard token")

	expectedExpiration := time.Now().Add(time.Duration(customTimeout) * time.Second)
	timeDiff := response.ExpiresAt.Sub(expectedExpiration)
	assert.Less(t, timeDiff.Abs(), 1*time.Second, "expiration should be close to custom timeout")
}

func Test_GetDashboardTokenContainsCorrectClaims(t *testing.T) {
	model := newModel(t)
	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	response, err := ctrl.GetDashboardToken(900)
	assert.NoError(t, err, "get dashboard token")

	token, err := jwt.Parse(response.Token, func(token *jwt.Token) (any, error) {
		return []byte(cfg.Secret()), nil
	})

	assert.NoError(t, err, "parse token")
	assert.True(t, token.Valid, "token should be valid")

	claims, ok := token.Claims.(jwt.MapClaims)
	assert.True(t, ok, "claims should be MapClaims")

	assert.Equal(t, "finch", claims["iss"], "issuer claim")
	assert.Equal(t, "dashboard", claims["sub"], "subject claim")
	assert.NotNil(t, claims["iat"], "issued at claim should exist")
	assert.NotNil(t, claims["exp"], "expiration claim should exist")

	expClaim := int64(claims["exp"].(float64))
	assert.Equal(t, response.ExpiresAt.Unix(), expClaim, "expiration claim should match response time")
}

func Test_GetDashboardTokenSignedWithCorrectSecret(t *testing.T) {
	model := newModel(t)
	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	response, err := ctrl.GetDashboardToken(0)
	assert.NoError(t, err, "get dashboard token")

	token, err := jwt.Parse(response.Token, func(token *jwt.Token) (any, error) {
		return []byte(cfg.Secret()), nil
	})
	assert.NoError(t, err, "parse token with correct secret")
	assert.True(t, token.Valid, "token should be valid with correct secret")

	wrongToken, err := jwt.Parse(response.Token, func(token *jwt.Token) (any, error) {
		return []byte("wrong-secret"), nil
	})
	assert.Error(t, err, "parse token with wrong secret should fail")
	assert.False(t, wrongToken.Valid, "token should be invalid with wrong secret")
}

func Test_ValidateDashboardTokenSucceeds_WithValidToken(t *testing.T) {
	model := newModel(t)
	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	response, err := ctrl.GetDashboardToken(0)
	assert.NoError(t, err, "get dashboard token")

	err = ctrl.ValidateDashboardToken(response.Token)
	assert.NoError(t, err, "validate valid token")
}

func Test_ValidateDashboardTokenReturnsError_WithExpiredToken(t *testing.T) {
	model := newModel(t)
	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	now := time.Now()
	claims := jwt.MapClaims{
		"iss": "finch",
		"sub": "dashboard",
		"iat": now.Add(-2 * time.Hour).Unix(),
		"exp": now.Add(-1 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(cfg.Secret()))
	assert.NoError(t, err, "sign expired token")

	err = ctrl.ValidateDashboardToken(tokenString)
	assert.Error(t, err, "validate expired token should fail")
}

func Test_ValidateDashboardTokenReturnsError_WithInvalidSignature(t *testing.T) {
	model := newModel(t)
	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	now := time.Now()
	claims := jwt.MapClaims{
		"iss": "finch",
		"sub": "dashboard",
		"iat": now.Unix(),
		"exp": now.Add(15 * time.Minute).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte("wrong-secret"))
	assert.NoError(t, err, "sign token with wrong secret")

	err = ctrl.ValidateDashboardToken(tokenString)
	assert.Error(t, err, "validate token with wrong signature should fail")
}

func Test_ValidateDashboardTokenReturnsError_WithMalformedToken(t *testing.T) {
	model := newModel(t)
	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	err := ctrl.ValidateDashboardToken("invalid.token.format")
	assert.Error(t, err, "validate malformed token should fail")
}

func Test_ValidateDashboardTokenReturnsError_WithEmptyToken(t *testing.T) {
	model := newModel(t)
	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	err := ctrl.ValidateDashboardToken("")
	assert.Error(t, err, "validate empty token should fail")
}

func Test_ValidateDashboardTokenReturnsError_WithWrongIssuer(t *testing.T) {
	model := newModel(t)
	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	now := time.Now()
	claims := jwt.MapClaims{
		"iss": "wrong-issuer",
		"sub": "dashboard",
		"iat": now.Unix(),
		"exp": now.Add(15 * time.Minute).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(cfg.Secret()))
	assert.NoError(t, err, "sign token with wrong issuer")

	err = ctrl.ValidateDashboardToken(tokenString)
	assert.Error(t, err, "validate token with wrong issuer should fail")
}

func Test_ValidateDashboardTokenReturnsError_WithWrongSubject(t *testing.T) {
	model := newModel(t)
	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	now := time.Now()
	claims := jwt.MapClaims{
		"iss": "finch",
		"sub": "wrong-subject",
		"iat": now.Unix(),
		"exp": now.Add(15 * time.Minute).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(cfg.Secret()))
	assert.NoError(t, err, "sign token with wrong subject")

	err = ctrl.ValidateDashboardToken(tokenString)
	assert.Error(t, err, "validate token with wrong subject should fail")
}
