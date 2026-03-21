/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package grpc

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/tschaefer/finch/internal/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	AuthHeader = "x-forwarded-tls-client-cert"
	CADirPath  = "%s/traefik/etc/certs.d"
	PEMHeader  = "-----BEGIN CERTIFICATE-----\n"
	PEMFooter  = "\n-----END CERTIFICATE-----\n"
)

type AuthInterceptor struct {
	config *config.Config
}

func NewAuthInterceptor(cfg *config.Config) *AuthInterceptor {
	return &AuthInterceptor{
		config: cfg,
	}
}

func (a *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if err := a.authenticate(ctx); err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

func (a *AuthInterceptor) authenticate(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		slog.Warn("no metadata in context")
		return status.Error(codes.InvalidArgument, "no metadata in context")
	}

	values := md.Get(AuthHeader)
	if len(values) == 0 {
		slog.Warn("no client certificate in metadata")
		return status.Error(codes.Unauthenticated, "permission denied")
	}

	certPem := fmt.Sprintf("%s%s%s", PEMHeader, values[0], PEMFooter)
	caDirPath := fmt.Sprintf(CADirPath, a.config.Library())

	pattern := filepath.Join(caDirPath, "*.pem")
	caFiles, err := filepath.Glob(pattern)
	if err != nil {
		slog.Error("failed to list CA certificates with glob pattern", "pattern", pattern, "error", err)
		return status.Error(codes.Internal, "internal server error")
	}

	if len(caFiles) == 0 {
		slog.Error("no CA certificates found in directory", "directory", caDirPath)
		return status.Error(codes.Internal, "internal server error")
	}

	for _, caFile := range caFiles {
		caPem, err := os.ReadFile(caFile)
		if err != nil {
			slog.Error("failed to read CA certificate", "path", caFile, "error", err)
			continue
		}

		valid, err := a.clientCertIsValid([]byte(certPem), caPem)
		if err != nil {
			continue
		}
		if valid {
			return nil
		}
	}

	slog.Warn("client certificate is not valid")
	return status.Error(codes.Unauthenticated, "permission denied")
}

func (a *AuthInterceptor) parseCertFromPEM(bytes []byte) (*x509.Certificate, error) {
	var block *pem.Block

	for {
		block, bytes = pem.Decode(bytes)
		if block == nil {
			return nil, errors.New("no PEM block found")
		}
		if block.Type == "CERTIFICATE" {
			return x509.ParseCertificate(block.Bytes)
		}
	}
}

func (a *AuthInterceptor) clientCertIsValid(clientPEM, caPEM []byte) (bool, error) {
	clientCert, err := a.parseCertFromPEM(clientPEM)
	if err != nil {
		return false, fmt.Errorf("parse client cert: %w", err)
	}

	caCert, err := a.parseCertFromPEM(caPEM)
	if err != nil {
		return false, fmt.Errorf("parse CA cert: %w", err)
	}

	roots := x509.NewCertPool()
	roots.AddCert(caCert)

	opts := x509.VerifyOptions{
		Roots:       roots,
		CurrentTime: time.Now(),
		KeyUsages:   []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	if _, err := clientCert.Verify(opts); err != nil {
		return false, err
	}

	return true, nil
}
