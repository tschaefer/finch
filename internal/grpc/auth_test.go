/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package grpc

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tschaefer/finch/internal/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	TestCA = `-----BEGIN CERTIFICATE-----
MIIBtjCCAVugAwIBAgIRAMDZb0jTSAU/EaHYww5oUecwCgYIKoZIzj0EAwIwOjEO
MAwGA1UEChMFRmluY2gxKDAmBgNVBAMTH0ZpbmNoIENBIC0gZmluY2gudC5jb3Jl
c2VjLnpvbmUwHhcNMjYwMTE5MTQyMzI0WhcNMjYwNDE5MTQyMzI0WjA6MQ4wDAYD
VQQKEwVGaW5jaDEoMCYGA1UEAxMfRmluY2ggQ0EgLSBmaW5jaC50LmNvcmVzZWMu
em9uZTBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABPDdUu9IQQponAiYg5A+tvvV
bFi4uIQpZmJ4AHLy712wKSf8+okDPPya55c0Kjy2VNiE5oARH3x7ltitIZnXDiej
QjBAMA4GA1UdDwEB/wQEAwIBhjAPBgNVHRMBAf8EBTADAQH/MB0GA1UdDgQWBBQ5
GDc2QmnrfRD32xYGHwwNlZ3sXDAKBggqhkjOPQQDAgNJADBGAiEA3HiFDzExl8Bk
uIsJUwo3c/2BgAoK78gVjsFMXLEhirUCIQDXXb/WhMeZROt+ZfEWFQgQU/ikjAPo
gjI5V+hs6TQSgQ==
-----END CERTIFICATE-----`
	TestCert = `
MIIBvzCCAWWgAwIBAgIRAKSS5NAflDBIU+OmV3iiy5EwCgYIKo
ZIzj0EAwIwOjEOMAwGA1UEChMFRmluY2gxKDAmBgNVBAMTH0ZpbmNoIENBIC0gZmluY2gudC
5jb3Jlc2VjLnpvbmUwHhcNMjYwMTE5MTQyMzI0WhcNMjYwNDE5MTQyMzI0WjA+MQ4wDAYDVQ
QKEwVGaW5jaDEsMCoGA1UEAxMjRmluY2ggQ2xpZW50IC0gZmluY2gudC5jb3Jlc2VjLnpvbm
UwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAASN6FdhaAQhAcLi4J7PeMzSQyndbaJJ8itOdT
x6DYT4uBZXe8rb6Gn4Vz/q49zZXyMQaanoMJij3+OhpXWIiviQo0gwRjAOBgNVHQ8BAf8EBA
MCBaAwEwYDVR0lBAwwCgYIKwYBBQUHAwIwHwYDVR0jBBgwFoAUORg3NkJp630Q99sWBh8MDZ
Wd7FwwCgYIKoZIzj0EAwIDSAAwRQIgE+Ewhuzh7Srikx7H9N9Qlu/U1OG5WYNWPy6OFkQPGQ
sCIQCn8o4SVaHuxhCpZIdu7uRz7hdZyxRyHxoHZIWKvWKa6g==
`
	TestInvalidCert = `
MIICkzCCAjmgAwIBAgIUf1frg4J/1f3Ad4lsPgmpuMzguoUwCgYIKoZIzj0EAwIw
gZ4xCzAJBgNVBAYTAkRFMQ8wDQYDVQQIDAZCYXllcm4xEzARBgNVBAcMCk3Dg8K8
bmNoZW4xEDAOBgNVBAoMB1ByaXZhdGUxEDAOBgNVBAsMB1ByaXZhdGUxKDAmBgNV
BAMMH0FsbCBZb3VyIENlcnRzIEFyZSBCZWxvbmcgVG8gVXMxGzAZBgkqhkiG9w0B
CQEWDHRsc0BhY21lLmNvbTAeFw0yNjAxMjAxODM3NDRaFw0yNzAxMjAxODM3NDRa
MIGeMQswCQYDVQQGEwJERTEPMA0GA1UECAwGQmF5ZXJuMRMwEQYDVQQHDApNw4PC
vG5jaGVuMRAwDgYDVQQKDAdQcml2YXRlMRAwDgYDVQQLDAdQcml2YXRlMSgwJgYD
VQQDDB9BbGwgWW91ciBDZXJ0cyBBcmUgQmVsb25nIFRvIFVzMRswGQYJKoZIhvcN
AQkBFgx0bHNAYWNtZS5jb20wWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAQheKac
QUfK8+RWiyEVBYzFbACmy0c7uggs91lTBzH/F3MYCbXyWZCcf1PYt/0CgPRoApAY
TvlNdnqi+A2cnjHIo1MwUTAdBgNVHQ4EFgQU4+3avq11eWPL8ymUB/LQQxzS7tgw
HwYDVR0jBBgwFoAU4+3avq11eWPL8ymUB/LQQxzS7tgwDwYDVR0TAQH/BAUwAwEB
/zAKBggqhkjOPQQDAgNIADBFAiEAsVkwAfe4TfUO0lS0V1ua3vgrjHOCsx488sn3
tM1nHb4CIFN6LWw/Th/A3uxLQIlTpO8wQCEx5u3g4eQQm6vSq53W
`
)

func setup(t *testing.T) string {
	library, err := os.MkdirTemp("", "finch-test-lib-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	capath := fmt.Sprintf(CAPath, library)
	if err := os.MkdirAll(filepath.Dir(capath), 0700); err != nil {
		t.Fatalf("failed to create ca dir: %v", err)
	}

	if err := os.WriteFile(capath, []byte(TestCA), 0600); err != nil {
		t.Fatalf("failed to write ca pem: %v", err)
	}

	return library
}

func TestAuthInterceptorSucceeds(t *testing.T) {
	library := setup(t)
	defer func() {
		_ = os.RemoveAll(library)
	}()

	cfg := config.NewFromData(&config.Data{}, library)
	interceptor := NewAuthInterceptor(cfg)
	unary := interceptor.Unary()

	md := metadata.Pairs(AuthHeader, strings.TrimSpace(TestCert))
	ctx := metadata.NewIncomingContext(context.Background(), md)

	called := false
	handler := func(ctx context.Context, req any) (any, error) {
		called = true
		return "ok", nil
	}

	resp, err := unary(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/test"}, handler)
	assert.NoError(t, err)
	assert.Equal(t, "ok", resp)
	assert.True(t, called, "handler should have been called")
}

func TestAuthInterceptorReturnsError_MissingMetadata(t *testing.T) {
	interceptor := NewAuthInterceptor(&config.Config{})
	unary := interceptor.Unary()

	ctx := context.Background()

	handlerCalled := false
	handler := func(ctx context.Context, req any) (any, error) {
		handlerCalled = true
		return nil, nil
	}

	_, err := unary(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/test"}, handler)
	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, "Code(418)", st.Code().String())
	assert.False(t, handlerCalled, "handler must not be called on auth failure")
}

func TestAuthInterceptorReturnsError_InvalidCert(t *testing.T) {
	library := setup(t)
	defer func() {
		_ = os.RemoveAll(library)
	}()

	cfg := config.NewFromData(&config.Data{}, library)
	interceptor := NewAuthInterceptor(cfg)
	unary := interceptor.Unary()

	md := metadata.Pairs(AuthHeader, strings.TrimSpace(TestInvalidCert))
	ctx := metadata.NewIncomingContext(context.Background(), md)

	handlerCalled := false
	handler := func(ctx context.Context, req any) (any, error) {
		handlerCalled = true
		return nil, nil
	}

	_, err := unary(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/test"}, handler)
	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, "Code(418)", st.Code().String())
	assert.False(t, handlerCalled, "handler must not be called on auth failure")
}
