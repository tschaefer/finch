/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package grpc

import (
	"context"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type LoggingInterceptor struct{}

func NewLoggingInterceptor() *LoggingInterceptor {
	return &LoggingInterceptor{}
}

func (l *LoggingInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		resp, err := handler(ctx, req)

		go l.log(ctx, info, err)

		return resp, err
	}
}

func (l *LoggingInterceptor) log(ctx context.Context, info *grpc.UnaryServerInfo, err error) {
	md, _ := metadata.FromIncomingContext(ctx)

	remoteAddr := ""
	for _, h := range []string{"x-forwarded-for", "x-real-ip"} {
		if v := md.Get(h); len(v) > 0 && v[0] != "" {
			remoteAddr = v[0]
			break
		}
	}
	if remoteAddr == "" {
		if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
			remoteAddr = p.Addr.String()
			for i := len(remoteAddr) - 1; i >= 0; i-- {
				if remoteAddr[i] == ':' {
					remoteAddr = remoteAddr[:i]
					break
				}
			}
		}
	}

	userAgent := ""
	if v := md.Get("user-agent"); len(v) > 0 {
		userAgent = v[0]
	}

	requestPath := info.FullMethod

	var code codes.Code
	var msg string
	if err == nil {
		code = codes.OK
		msg = "ok"
	} else {
		if st, ok := status.FromError(err); ok {
			code = st.Code()
			msg = st.Message()
		} else {
			code = codes.Unknown
			msg = err.Error()
		}
	}

	args := []any{
		slog.String("code", code.String()),
		slog.String("request_path", requestPath),
		slog.String("remote_addr", remoteAddr),
		slog.String("user_agent", userAgent),
	}

	switch code {
	case codes.OK:
		slog.Info(msg, args...)
	case codes.Internal, codes.DataLoss, codes.Unknown, codes.Unavailable, codes.DeadlineExceeded:
		slog.Error(msg, args...)
	case codes.InvalidArgument, codes.NotFound, codes.AlreadyExists, codes.FailedPrecondition,
		codes.Unauthenticated, codes.PermissionDenied, codes.ResourceExhausted,
		codes.Canceled, codes.Aborted, 418:
		slog.Warn(msg, args...)
	default:
		slog.Info(msg, args...)
	}
}
