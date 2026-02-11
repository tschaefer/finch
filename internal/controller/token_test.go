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

func Test_GenerateAgentTokenReturnsValidToken(t *testing.T) {
	model := newModel(t)
	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	resourceId := "rid:finch:test-id:agent:abc-123"
	tokenString, expiresAt, err := ctrl.GenerateAgentToken(resourceId, 0)

	assert.NoError(t, err, "generate token")
	assert.NotEmpty(t, tokenString, "token string should not be empty")
	assert.True(t, expiresAt.After(time.Now()), "expiration should be in future")
}

func Test_GenerateAgentTokenUsesDefaultExpiration(t *testing.T) {
	model := newModel(t)
	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	resourceId := "rid:finch:test-id:agent:abc-123"
	_, expiresAt, err := ctrl.GenerateAgentToken(resourceId, 0)

	assert.NoError(t, err, "generate token")

	expectedExpiration := time.Now().Add(defaultTokenExpiration)
	timeDiff := expiresAt.Sub(expectedExpiration)
	assert.Less(t, timeDiff.Abs(), 1*time.Second, "expiration should be close to default")
}

func Test_GenerateAgentTokenUsesCustomExpiration(t *testing.T) {
	model := newModel(t)
	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	resourceId := "rid:finch:test-id:agent:abc-123"
	customExpiration := 24 * time.Hour
	_, expiresAt, err := ctrl.GenerateAgentToken(resourceId, customExpiration)

	assert.NoError(t, err, "generate token")

	expectedExpiration := time.Now().Add(customExpiration)
	timeDiff := expiresAt.Sub(expectedExpiration)
	assert.Less(t, timeDiff.Abs(), 1*time.Second, "expiration should be close to custom value")
}

func Test_GenerateAgentTokenContainsCorrectClaims(t *testing.T) {
	model := newModel(t)
	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	resourceId := "rid:finch:test-id:agent:abc-123"
	tokenString, expiresAt, err := ctrl.GenerateAgentToken(resourceId, 1*time.Hour)

	assert.NoError(t, err, "generate token")

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		return []byte(cfg.Secret()), nil
	})

	assert.NoError(t, err, "parse token")
	assert.True(t, token.Valid, "token should be valid")

	claims, ok := token.Claims.(jwt.MapClaims)
	assert.True(t, ok, "claims should be MapClaims")

	assert.Equal(t, "finch", claims["iss"], "issuer claim")
	assert.Equal(t, "agent", claims["sub"], "subject claim")
	assert.Equal(t, resourceId, claims["rid"], "resource ID claim")
	assert.NotNil(t, claims["iat"], "issued at claim should exist")
	assert.NotNil(t, claims["exp"], "expiration claim should exist")

	expClaim := int64(claims["exp"].(float64))
	assert.Equal(t, expiresAt.Unix(), expClaim, "expiration claim should match returned time")
}

func Test_GenerateAgentTokenSignedWithCorrectSecret(t *testing.T) {
	model := newModel(t)
	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	resourceId := "rid:finch:test-id:agent:abc-123"
	tokenString, _, err := ctrl.GenerateAgentToken(resourceId, 1*time.Hour)

	assert.NoError(t, err, "generate token")

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		return []byte(cfg.Secret()), nil
	})
	assert.NoError(t, err, "parse token with correct secret")
	assert.True(t, token.Valid, "token should be valid with correct secret")

	wrongToken, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		return []byte("wrong-secret"), nil
	})
	assert.Error(t, err, "parse token with wrong secret should fail")
	assert.False(t, wrongToken.Valid, "token should be invalid with wrong secret")
}

func Test_GenerateAgentTokenUsesHS256Algorithm(t *testing.T) {
	model := newModel(t)
	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	resourceId := "rid:finch:test-id:agent:abc-123"
	tokenString, _, err := ctrl.GenerateAgentToken(resourceId, 1*time.Hour)

	assert.NoError(t, err, "generate token")

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		_, ok := token.Method.(*jwt.SigningMethodHMAC)
		assert.True(t, ok, "signing method should be HMAC")
		assert.Equal(t, "HS256", token.Method.Alg(), "algorithm should be HS256")
		return []byte(cfg.Secret()), nil
	})

	assert.NoError(t, err, "parse token")
	assert.True(t, token.Valid, "token should be valid")
}
