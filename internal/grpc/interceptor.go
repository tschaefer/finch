/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package grpc

import (
	"context"
	"encoding/base64"
	"log/slog"
	"strings"

	"github.com/tschaefer/finch/internal/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type AuthInterceptor struct {
	config config.Config
}

func NewAuthInterceptor(cfg config.Config) *AuthInterceptor {
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
		return status.Error(codes.Unauthenticated, "missing metadata")
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return status.Error(codes.Unauthenticated, "missing authorization header")
	}

	authHeader := values[0]
	if !strings.HasPrefix(authHeader, "Basic ") {
		return status.Error(codes.Unauthenticated, "invalid authorization header")
	}

	encoded := strings.TrimPrefix(authHeader, "Basic ")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return status.Error(codes.Unauthenticated, "invalid authorization header")
	}

	credentials := strings.SplitN(string(decoded), ":", 2)
	if len(credentials) != 2 {
		return status.Error(codes.Unauthenticated, "invalid credentials format")
	}

	username, password := a.config.Credentials()
	if credentials[0] != username || credentials[1] != password {
		slog.Warn("Unauthorized gRPC request", "username", credentials[0])
		return status.Error(codes.Unauthenticated, "invalid credentials")
	}

	return nil
}
