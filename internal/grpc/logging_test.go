package grpc

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var record bytes.Buffer

type notifyHandler struct {
	h  slog.Handler
	ch chan struct{}
}

func (n *notifyHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return n.h.Enabled(ctx, level)
}

func (n *notifyHandler) Handle(ctx context.Context, r slog.Record) error {
	if err := n.h.Handle(ctx, r); err != nil {
		return err
	}
	select {
	case n.ch <- struct{}{}:
	default:
	}
	return nil
}

func (n *notifyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &notifyHandler{h: n.h.WithAttrs(attrs), ch: n.ch}
}

func (n *notifyHandler) WithGroup(name string) slog.Handler {
	return &notifyHandler{h: n.h.WithGroup(name), ch: n.ch}
}

func setupLogger() chan struct{} {
	record.Reset()
	ch := make(chan struct{}, 1)

	jsonHandler := slog.NewJSONHandler(&record, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	slog.SetDefault(slog.New(&notifyHandler{h: jsonHandler, ch: ch}))

	return ch
}

func waitForLog(t *testing.T, ch chan struct{}) {
	select {
	case <-ch:
	case <-time.After(1 * time.Second):
		t.Fatalf("timeout waiting for log")
	}
}

func TestLoggingInterceptor_Success(t *testing.T) {
	ch := setupLogger()

	interceptor := NewLoggingInterceptor()
	unary := interceptor.Unary()

	ctx := context.Background()

	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}

	resp, err := unary(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/test"}, handler)
	assert.NoError(t, err)
	assert.Equal(t, "ok", resp)

	waitForLog(t, ch)

	assert.NotEmpty(t, record.String(), "log should not be empty")
	var content map[string]any
	err = json.Unmarshal(record.Bytes(), &content)
	assert.NoError(t, err)

	assert.Equal(t, "ok", content["msg"])
	assert.Equal(t, codes.OK.String(), content["code"])
	assert.Equal(t, "/test", content["request_path"])
	assert.Equal(t, "INFO", content["level"])
}

func TestLoggingInterceptor_Error(t *testing.T) {
	ch := setupLogger()

	interceptor := NewLoggingInterceptor()
	unary := interceptor.Unary()

	ctx := context.Background()

	handler := func(ctx context.Context, req any) (any, error) {
		return nil, status.Error(codes.Internal, "boom")
	}

	_, err := unary(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/test"}, handler)
	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())

	waitForLog(t, ch)

	assert.NotEmpty(t, record.String(), "log should not be empty")
	var content map[string]any
	err = json.Unmarshal(record.Bytes(), &content)
	assert.NoError(t, err)

	assert.Equal(t, "boom", content["msg"])
	assert.Equal(t, codes.Internal.String(), content["code"])
	assert.Equal(t, "/test", content["request_path"])
	assert.Equal(t, "ERROR", content["level"])
}

func TestLoggingInterceptor_Warn(t *testing.T) {
	ch := setupLogger()

	interceptor := NewLoggingInterceptor()
	unary := interceptor.Unary()

	ctx := context.Background()

	handler := func(ctx context.Context, req any) (any, error) {
		return nil, status.Error(codes.Unauthenticated, "unauthenticated")
	}

	_, err := unary(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/test"}, handler)
	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())

	waitForLog(t, ch)

	assert.NotEmpty(t, record.String(), "log should not be empty")
	var content map[string]any
	err = json.Unmarshal(record.Bytes(), &content)
	assert.NoError(t, err)

	assert.Equal(t, "unauthenticated", content["msg"])
	assert.Equal(t, codes.Unauthenticated.String(), content["code"])
	assert.Equal(t, "/test", content["request_path"])
	assert.Equal(t, "WARN", content["level"])
}
