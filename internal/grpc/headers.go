/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package grpc

import (
	"context"
	"log/slog"

	"github.com/tschaefer/finch/internal/version"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type HeadersInterceptor struct{}

func NewHeadersInterceptor() *HeadersInterceptor {
	return &HeadersInterceptor{}
}

func (h *HeadersInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		h.setHeaders(ctx)

		return handler(ctx, req)
	}
}

func (h *HeadersInterceptor) setHeaders(ctx context.Context) {
	if err := grpc.SetHeader(ctx, metadata.Pairs(
		"x-finch-commit", version.Commit(),
		"x-finch-release", version.Release(),
	)); err != nil {
		slog.Warn("failed to set finch response headers", "error", err)
	}
}
