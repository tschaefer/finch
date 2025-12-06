package grpc

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestAuthInterceptor_Success(t *testing.T) {
	interceptor := NewAuthInterceptor(&mockedConfig)
	unary := interceptor.Unary()

	user, password := mockedConfig.Credentials()
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

func TestAuthInterceptor_MissingMetadata(t *testing.T) {
	interceptor := NewAuthInterceptor(&mockedConfig)
	unary := interceptor.Unary()

	ctx := context.Background() // no metadata

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

func TestAuthInterceptor_InvalidHeaderPrefix(t *testing.T) {
	interceptor := NewAuthInterceptor(&mockedConfig)
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

func TestAuthInterceptor_InvalidBase64(t *testing.T) {
	interceptor := NewAuthInterceptor(&mockedConfig)
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

func TestAuthInterceptor_WrongCredentials(t *testing.T) {
	interceptor := NewAuthInterceptor(&mockedConfig)
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
