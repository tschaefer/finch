package grpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tschaefer/finch/internal/version"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type fakeServerTransportStream struct {
	header metadata.MD
}

func (f *fakeServerTransportStream) Method() string {
	return "fake"
}

func (f *fakeServerTransportStream) SetHeader(md metadata.MD) error {
	f.header = md
	return nil
}

func (f *fakeServerTransportStream) SendHeader(md metadata.MD) error {
	return nil
}

func (f *fakeServerTransportStream) SetTrailer(md metadata.MD) error {
	return nil
}

func TestHeadersInterceptor_Success(t *testing.T) {
	interceptor := NewHeadersInterceptor()
	unary := interceptor.Unary()

	ctx := context.Background()
	fake := &fakeServerTransportStream{}
	ctx = grpc.NewContextWithServerTransportStream(ctx, fake)

	called := false
	handler := func(ctx context.Context, req any) (any, error) {
		called = true
		return "ok", nil
	}

	version.GitCommit = "test-commit"
	version.Version = "v1.2.3"

	resp, err := unary(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/test"}, handler)
	assert.NoError(t, err)
	assert.Equal(t, "ok", resp)
	assert.True(t, called, "handler should have been invoked")
	assert.NotNil(t, fake.header)
	assert.Contains(t, fake.header, "x-finch-commit")
	assert.Contains(t, fake.header, "x-finch-release")
	assert.Equal(t, version.Commit(), fake.header.Get("x-finch-commit")[0])
	assert.Equal(t, version.Release(), fake.header.Get("x-finch-release")[0])
}
