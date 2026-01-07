/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package grpc

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tschaefer/finch/internal/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var testAuthCfg = config.NewFromData(&config.Data{
	Credentials: config.Credentials{
		Username: "testuser",
		Password: "testpass",
	},
}, "")

func TestAuthInterceptorSucceeds(t *testing.T) {
	interceptor := NewAuthInterceptor(testAuthCfg)
	unary := interceptor.Unary()

	user, password := testAuthCfg.Credentials()
	token := base64.StdEncoding.EncodeToString([]byte(user + ":" + password))
	md := metadata.Pairs("authorization", "Basic "+token)
	ctx := metadata.NewIncomingContext(context.Background(), md)

	called := false
	handler := func(ctx context.Context, req any) (any, error) {
		called = true
		return "ok", nil
	}

	resp, err := unary(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/test"}, handler)
	assert.NoError(t, err)
	assert.Equal(t, "ok", resp)
	assert.True(t, called, "handler should have been invoked")
}

func TestAuthInterceptorReturnsError_MissingMetadata(t *testing.T) {
	interceptor := NewAuthInterceptor(testAuthCfg)
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
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.False(t, handlerCalled, "handler must not be called on auth failure")
}

func TestAuthInterceptorReturnsError_InvalidAuthType(t *testing.T) {
	interceptor := NewAuthInterceptor(testAuthCfg)
	unary := interceptor.Unary()

	md := metadata.Pairs("authorization", "Bearer sometoken")
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
	assert.False(t, handlerCalled)
}

func TestAuthInterceptorReturnsError_InvalidAuthEncoding(t *testing.T) {
	interceptor := NewAuthInterceptor(testAuthCfg)
	unary := interceptor.Unary()

	md := metadata.Pairs("authorization", "Basic not-base64!!")
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
	assert.False(t, handlerCalled)
}

func TestAuthInterceptorReturnsError_InvalidCredentials(t *testing.T) {
	interceptor := NewAuthInterceptor(testAuthCfg)
	unary := interceptor.Unary()

	token := base64.StdEncoding.EncodeToString([]byte("baduser:badpass"))
	md := metadata.Pairs("authorization", "Basic "+token)
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
	assert.False(t, handlerCalled)
}
