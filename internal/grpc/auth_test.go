/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package grpc

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tschaefer/finch/internal/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type testSetup struct {
	library         string
	clientCertBody  string
	invalidCertBody string
}

func generateCA(t *testing.T) (*ecdsa.PrivateKey, *x509.Certificate, []byte) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate CA key: %v", err)
	}

	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{Organization: []string{"Finch"}, CommonName: "Finch Test CA"},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("failed to create CA cert: %v", err)
	}

	cert, err := x509.ParseCertificate(derBytes)
	if err != nil {
		t.Fatalf("failed to parse CA cert: %v", err)
	}

	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	return key, cert, pemBytes
}

func generateClientCert(t *testing.T, caCert *x509.Certificate, caKey *ecdsa.PrivateKey) string {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate client key: %v", err)
	}

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{Organization: []string{"Finch"}, CommonName: "Finch Test Client"},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, &key.PublicKey, caKey)
	if err != nil {
		t.Fatalf("failed to create client cert: %v", err)
	}

	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	body := strings.TrimSpace(string(pemBytes))
	body = strings.TrimPrefix(body, "-----BEGIN CERTIFICATE-----")
	body = strings.TrimSuffix(body, "-----END CERTIFICATE-----")
	return strings.TrimSpace(body)
}

func setup(t *testing.T) *testSetup {
	t.Helper()

	library, err := os.MkdirTemp("", "finch-test-lib-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	caDirPath := fmt.Sprintf(CADirPath, library)
	if err := os.MkdirAll(caDirPath, 0700); err != nil {
		_ = os.RemoveAll(library)
		t.Fatalf("failed to create ca dir: %v", err)
	}

	caKey, caCert, caPEM := generateCA(t)

	caFile := filepath.Join(caDirPath, "rid:finchctl:47110815.pem")
	if err := os.WriteFile(caFile, caPEM, 0600); err != nil {
		_ = os.RemoveAll(library)
		t.Fatalf("failed to write ca pem: %v", err)
	}

	clientCertBody := generateClientCert(t, caCert, caKey)

	otherCAKey, otherCACert, _ := generateCA(t)
	invalidCertBody := generateClientCert(t, otherCACert, otherCAKey)

	return &testSetup{
		library:         library,
		clientCertBody:  clientCertBody,
		invalidCertBody: invalidCertBody,
	}
}

func TestAuthInterceptorSucceeds(t *testing.T) {
	ts := setup(t)
	defer func() {
		_ = os.RemoveAll(ts.library)
	}()

	cfg := config.NewFromData(&config.Data{}, ts.library)
	interceptor := NewAuthInterceptor(cfg)
	unary := interceptor.Unary()

	md := metadata.Pairs(AuthHeader, ts.clientCertBody)
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
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.False(t, handlerCalled, "handler must not be called on auth failure")
}

func TestAuthInterceptorReturnsError_InvalidCert(t *testing.T) {
	ts := setup(t)
	defer func() {
		_ = os.RemoveAll(ts.library)
	}()

	cfg := config.NewFromData(&config.Data{}, ts.library)
	interceptor := NewAuthInterceptor(cfg)
	unary := interceptor.Unary()

	md := metadata.Pairs(AuthHeader, ts.invalidCertBody)
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
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.False(t, handlerCalled, "handler must not be called on auth failure")
}

func TestAuthInterceptorSucceeds_MultipleCAsFirstFailsLaterSucceeds(t *testing.T) {
	ts := setup(t)
	defer func() {
		_ = os.RemoveAll(ts.library)
	}()

	caDirPath := fmt.Sprintf(CADirPath, ts.library)
	badCAFile := filepath.Join(caDirPath, "aaa_bad.pem")
	if err := os.WriteFile(badCAFile, []byte(ts.invalidCertBody), 0600); err != nil {
		t.Fatalf("failed to write bad ca pem: %v", err)
	}

	cfg := config.NewFromData(&config.Data{}, ts.library)
	interceptor := NewAuthInterceptor(cfg)
	unary := interceptor.Unary()

	md := metadata.Pairs(AuthHeader, ts.clientCertBody)
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

func TestAuthInterceptorReturnsError_EmptyCADirectory(t *testing.T) {
	library, err := os.MkdirTemp("", "finch-test-lib-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(library)
	}()

	caDirPath := fmt.Sprintf(CADirPath, library)
	if err := os.MkdirAll(caDirPath, 0700); err != nil {
		t.Fatalf("failed to create ca dir: %v", err)
	}

	caKey, caCert, _ := generateCA(t)
	clientCertBody := generateClientCert(t, caCert, caKey)

	cfg := config.NewFromData(&config.Data{}, library)
	interceptor := NewAuthInterceptor(cfg)
	unary := interceptor.Unary()

	md := metadata.Pairs(AuthHeader, clientCertBody)
	ctx := metadata.NewIncomingContext(context.Background(), md)

	handlerCalled := false
	handler := func(ctx context.Context, req any) (any, error) {
		handlerCalled = true
		return nil, nil
	}

	_, err = unary(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/test"}, handler)
	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	assert.False(t, handlerCalled, "handler must not be called on auth failure")
}
