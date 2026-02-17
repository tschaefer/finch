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

func Test_GenerateDashboardTokenReturnsValidToken(t *testing.T) {
	model := newModel(t)
	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	response, err := ctrl.GenerateDashboardToken(0, RoleOperator, []string{})

	assert.NoError(t, err, "get dashboard token")
	assert.NotNil(t, response, "response should not be nil")
	assert.NotEmpty(t, response.Token, "token string should not be empty")
	assert.True(t, response.ExpiresAt.After(time.Now()), "expiration should be in future")
	assert.NotEmpty(t, response.DashboardURL, "dashboard URL should not be empty")
}

func Test_GenerateDashboardTokenUsesDefaultTimeout(t *testing.T) {
	model := newModel(t)
	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	response, err := ctrl.GenerateDashboardToken(0, RoleOperator, []string{})
	assert.NoError(t, err, "get dashboard token")

	expectedExpiration := time.Now().Add(1800 * time.Second)
	timeDiff := response.ExpiresAt.Sub(expectedExpiration)
	assert.Less(t, timeDiff.Abs(), 1*time.Second, "expiration should be close to default (1800s)")
}

func Test_GenerateDashboardTokenUsesCustomTimeout(t *testing.T) {
	model := newModel(t)
	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	customTimeout := 3600 // 1 hour
	response, err := ctrl.GenerateDashboardToken(customTimeout, RoleOperator, []string{})
	assert.NoError(t, err, "get dashboard token")

	expectedExpiration := time.Now().Add(time.Duration(customTimeout) * time.Second)
	timeDiff := response.ExpiresAt.Sub(expectedExpiration)
	assert.Less(t, timeDiff.Abs(), 1*time.Second, "expiration should be close to custom timeout")
}

func Test_GenerateDashboardTokenContainsCorrectClaims(t *testing.T) {
	model := newModel(t)
	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	response, err := ctrl.GenerateDashboardToken(900, RoleOperator, []string{})
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
	assert.Equal(t, RoleOperator, claims["role"], "role claim")
	assert.Equal(t, "[]", claims["scope"], "scope claim should be JSON array")
	assert.NotNil(t, claims["iat"], "issued at claim should exist")
	assert.NotNil(t, claims["exp"], "expiration claim should exist")

	expClaim := int64(claims["exp"].(float64))
	assert.Equal(t, response.ExpiresAt.Unix(), expClaim, "expiration claim should match response time")
}

func Test_GenerateDashboardTokenSignedWithCorrectSecret(t *testing.T) {
	model := newModel(t)
	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	response, err := ctrl.GenerateDashboardToken(0, RoleOperator, []string{})
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

	response, err := ctrl.GenerateDashboardToken(0, RoleOperator, []string{})
	assert.NoError(t, err, "get dashboard token")

	claims, err := ctrl.ValidateDashboardToken(response.Token)
	assert.NoError(t, err, "validate valid token")
	assert.NotNil(t, claims, "claims should not be nil")
	assert.Equal(t, RoleOperator, claims.Role, "role should match")
	assert.Equal(t, []string{}, claims.Scope, "scope should match")
}

func Test_ValidateDashboardTokenReturnsError_WithExpiredToken(t *testing.T) {
	model := newModel(t)
	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	now := time.Now()
	claims := jwt.MapClaims{
		"iss":   "finch",
		"sub":   "dashboard",
		"role":  RoleOperator,
		"scope": "[]",
		"iat":   now.Add(-2 * time.Hour).Unix(),
		"exp":   now.Add(-1 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(cfg.Secret()))
	assert.NoError(t, err, "sign expired token")

	_, err = ctrl.ValidateDashboardToken(tokenString)
	assert.Error(t, err, "validate expired token should fail")
}

func Test_ValidateDashboardTokenReturnsError_WithInvalidSignature(t *testing.T) {
	model := newModel(t)
	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	now := time.Now()
	claims := jwt.MapClaims{
		"iss":   "finch",
		"sub":   "dashboard",
		"role":  RoleOperator,
		"scope": "[]",
		"iat":   now.Unix(),
		"exp":   now.Add(15 * time.Minute).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte("wrong-secret"))
	assert.NoError(t, err, "sign token with wrong secret")

	_, err = ctrl.ValidateDashboardToken(tokenString)
	assert.Error(t, err, "validate token with wrong signature should fail")
}

func Test_ValidateDashboardTokenReturnsError_WithMalformedToken(t *testing.T) {
	model := newModel(t)
	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	_, err := ctrl.ValidateDashboardToken("invalid.token.format")
	assert.Error(t, err, "validate malformed token should fail")
}

func Test_ValidateDashboardTokenReturnsError_WithEmptyToken(t *testing.T) {
	model := newModel(t)
	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	_, err := ctrl.ValidateDashboardToken("")
	assert.Error(t, err, "validate empty token should fail")
}

func Test_ValidateDashboardTokenReturnsError_WithWrongIssuer(t *testing.T) {
	model := newModel(t)
	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	now := time.Now()
	claims := jwt.MapClaims{
		"iss":   "wrong-issuer",
		"sub":   "dashboard",
		"role":  RoleOperator,
		"scope": "[]",
		"iat":   now.Unix(),
		"exp":   now.Add(15 * time.Minute).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(cfg.Secret()))
	assert.NoError(t, err, "sign token with wrong issuer")

	_, err = ctrl.ValidateDashboardToken(tokenString)
	assert.Error(t, err, "validate token with wrong issuer should fail")
}

func Test_ValidateDashboardTokenReturnsError_WithWrongSubject(t *testing.T) {
	model := newModel(t)
	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	now := time.Now()
	claims := jwt.MapClaims{
		"iss":   "finch",
		"sub":   "wrong-subject",
		"role":  RoleOperator,
		"scope": "[]",
		"iat":   now.Unix(),
		"exp":   now.Add(15 * time.Minute).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(cfg.Secret()))
	assert.NoError(t, err, "sign token with wrong subject")

	_, err = ctrl.ValidateDashboardToken(tokenString)
	assert.Error(t, err, "validate token with wrong subject should fail")
}

func Test_GenerateDashboardTokenReturnsError_WithInvalidRole(t *testing.T) {
	model := newModel(t)
	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	_, err := ctrl.GenerateDashboardToken(1800, "invalid-role", []string{})
	assert.Error(t, err, "get dashboard token with invalid role should fail")
	assert.Equal(t, ErrInvalidRole, err, "error should be ErrInvalidRole")
}

func Test_CanViewTokens_AdminRole(t *testing.T) {
	model := newModel(t)
	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	claims := &DashboardClaims{Role: RoleAdmin, Scope: []string{}}
	assert.True(t, ctrl.CanViewTokens(claims), "admin should be able to view tokens")
}

func Test_CanViewTokens_OperatorRole(t *testing.T) {
	model := newModel(t)
	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	claims := &DashboardClaims{Role: RoleOperator, Scope: []string{}}
	assert.True(t, ctrl.CanViewTokens(claims), "operator should be able to view tokens")
}

func Test_CanViewTokens_ViewerRole(t *testing.T) {
	model := newModel(t)
	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	claims := &DashboardClaims{Role: RoleViewer, Scope: []string{}}
	assert.False(t, ctrl.CanViewTokens(claims), "viewer should not be able to view tokens")
}

func Test_CanDownloadConfig_AdminRole(t *testing.T) {
	model := newModel(t)
	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	claims := &DashboardClaims{Role: RoleAdmin, Scope: []string{}}
	assert.True(t, ctrl.CanDownloadConfig(claims), "admin should be able to download config")
}

func Test_CanDownloadConfig_OperatorRole(t *testing.T) {
	model := newModel(t)
	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	claims := &DashboardClaims{Role: RoleOperator, Scope: []string{}}
	assert.False(t, ctrl.CanDownloadConfig(claims), "operator should not be able to download config")
}

func Test_CanDownloadConfig_ViewerRole(t *testing.T) {
	model := newModel(t)
	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	claims := &DashboardClaims{Role: RoleViewer, Scope: []string{}}
	assert.False(t, ctrl.CanDownloadConfig(claims), "viewer should not be able to download config")
}

func Test_CanAccessAgent_WithEmptyScope(t *testing.T) {
	model := newModel(t)
	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	claims := &DashboardClaims{Role: RoleAdmin, Scope: []string{}}
	assert.True(t, ctrl.CanAccessAgent(claims, "rid-123", "host.example.com"), "should access any agent with empty scope")
}

func Test_CanAccessAgent_WithSpecificRID(t *testing.T) {
	model := newModel(t)
	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	claims := &DashboardClaims{Role: RoleAdmin, Scope: []string{"rid-123", "rid-456"}}
	assert.True(t, ctrl.CanAccessAgent(claims, "rid-123", "host.example.com"), "should access agent with matching RID")
	assert.False(t, ctrl.CanAccessAgent(claims, "rid-789", "other.example.com"), "should not access agent without matching RID")
}

func Test_CanAccessAgent_WithSpecificHostname(t *testing.T) {
	model := newModel(t)
	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	claims := &DashboardClaims{Role: RoleAdmin, Scope: []string{"host1.example.com", "host2.example.com"}}
	assert.True(t, ctrl.CanAccessAgent(claims, "rid-123", "host1.example.com"), "should access agent with matching hostname")
	assert.False(t, ctrl.CanAccessAgent(claims, "rid-456", "host3.example.com"), "should not access agent without matching hostname")
}

func Test_CanAccessAgent_WithHostnameAll(t *testing.T) {
	model := newModel(t)
	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	claims := &DashboardClaims{Role: RoleAdmin, Scope: []string{"all"}}
	assert.True(t, ctrl.CanAccessAgent(claims, "rid-123", "all"), "should access agent with hostname 'all'")
	assert.False(t, ctrl.CanAccessAgent(claims, "rid-456", "other.example.com"), "should not access agent without matching hostname")
}

func Test_GenerateDashboardTokenEncodesMultipleScopes(t *testing.T) {
	model := newModel(t)
	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	response, err := ctrl.GenerateDashboardToken(900, RoleAdmin, []string{"host1", "host2", "host3"})
	assert.NoError(t, err, "get dashboard token")

	token, err := jwt.Parse(response.Token, func(token *jwt.Token) (any, error) {
		return []byte(cfg.Secret()), nil
	})

	assert.NoError(t, err, "parse token")
	assert.True(t, token.Valid, "token should be valid")

	claims, ok := token.Claims.(jwt.MapClaims)
	assert.True(t, ok, "claims should be MapClaims")
	assert.Equal(t, "[\"host1\",\"host2\",\"host3\"]", claims["scope"], "scope should be JSON array")
}

func Test_GenerateDashboardTokenWithEmptyScopeArray(t *testing.T) {
	model := newModel(t)
	ctrl := New(model, cfg)
	assert.NotNil(t, ctrl, "create controller")

	response, err := ctrl.GenerateDashboardToken(900, RoleViewer, []string{})
	assert.NoError(t, err, "get dashboard token")

	token, err := jwt.Parse(response.Token, func(token *jwt.Token) (any, error) {
		return []byte(cfg.Secret()), nil
	})

	assert.NoError(t, err, "parse token")
	claims, ok := token.Claims.(jwt.MapClaims)
	assert.True(t, ok, "claims should be MapClaims")
	assert.Equal(t, "[]", claims["scope"], "empty scope array should be JSON empty array in JWT")
}
