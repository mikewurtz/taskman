package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	basegrpc "github.com/mikewurtz/taskman/internal/grpc"
)

func TestExtractClientCNInterceptor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc         string
		setUpTestCtx func() context.Context
		wantCN       string
		wantCalled   bool
		wantErrCode  codes.Code
	}{
		{
			desc: "with valid client cert",
			setUpTestCtx: func() context.Context {
				cert := &x509.Certificate{
					Subject: pkix.Name{
						CommonName: "fake-client001",
					},
				}
				tlsInfo := credentials.TLSInfo{
					State: tls.ConnectionState{
						PeerCertificates: []*x509.Certificate{cert},
					},
				}
				return peer.NewContext(context.Background(), &peer.Peer{
					AuthInfo: tlsInfo,
				})
			},
			wantCN:      "fake-client001",
			wantCalled:  true,
			wantErrCode: codes.OK,
		},
		{
			desc: "with no peer context",
			setUpTestCtx: func() context.Context {
				return context.Background()
			},
			wantCN:      "",
			wantCalled:  false,
			wantErrCode: codes.Unauthenticated,
		},
		{
			desc: "with peer info but no tls info",
			setUpTestCtx: func() context.Context {
				return peer.NewContext(context.Background(), &peer.Peer{})
			},
			wantCN:      "",
			wantCalled:  false,
			wantErrCode: codes.Unauthenticated,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()
			var called bool

			handler := func(ctx context.Context, req any) (any, error) {
				called = true
				if cn, ok := ctx.Value(basegrpc.ClientCNKey).(string); ok {
					require.Equal(t, tt.wantCN, cn, "Common Name mismatch")
				} else if tt.wantCN != "" {
					require.Fail(t, "Expected CN in context")
				}
				return nil, nil
			}

			ctx := tt.setUpTestCtx()
			_, err := ExtractClientCNInterceptor(ctx, nil, nil, handler)
			if tt.wantErrCode != codes.OK {
				require.Error(t, err, "Expected error from unary interceptor")
				require.Equal(t, tt.wantErrCode, status.Code(err), "Error code mismatch")
			} else {
				require.NoError(t, err, "Unexpected error from unary interceptor")
			}
			require.Equal(t, tt.wantCalled, called, "Handler called status mismatch")
		})
	}
}

type mockServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (m *mockServerStream) Context() context.Context {
	return m.ctx
}

func TestExtractClientCNStreamInterceptor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc         string
		setUpTestCtx func() context.Context
		wantCN       string
		wantCalled   bool
		wantErrCode  codes.Code
	}{
		{
			desc: "with valid client cert",
			setUpTestCtx: func() context.Context {
				cert := &x509.Certificate{
					Subject: pkix.Name{
						CommonName: "fake-client001",
					},
				}
				tlsInfo := credentials.TLSInfo{
					State: tls.ConnectionState{
						PeerCertificates: []*x509.Certificate{cert},
					},
				}
				return peer.NewContext(context.Background(), &peer.Peer{
					AuthInfo: tlsInfo,
				})
			},
			wantCN:      "fake-client001",
			wantCalled:  true,
			wantErrCode: codes.OK,
		},
		{
			desc: "without peer context set",
			setUpTestCtx: func() context.Context {
				return context.Background()
			},
			wantCN:      "",
			wantCalled:  false,
			wantErrCode: codes.Unauthenticated,
		},
		{
			desc: "with peer context but no tls info",
			setUpTestCtx: func() context.Context {
				return peer.NewContext(context.Background(), &peer.Peer{})
			},
			wantCN:      "",
			wantCalled:  false,
			wantErrCode: codes.Unauthenticated,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()
			var called bool

			handler := func(srv any, stream grpc.ServerStream) error {
				called = true
				if cn, ok := stream.Context().Value(basegrpc.ClientCNKey).(string); ok {
					require.Equal(t, tt.wantCN, cn, "Common Name mismatch")
				} else if tt.wantCN != "" {
					require.Fail(t, "Expected CN in context")
				}
				return nil
			}

			stream := &mockServerStream{ctx: tt.setUpTestCtx()}
			err := ExtractClientCNStreamInterceptor(nil, stream, nil, handler)
			if tt.wantErrCode != codes.OK {
				require.Error(t, err, "Expected error from stream interceptor")
				require.Equal(t, tt.wantErrCode, status.Code(err), "Error code mismatch")
			} else {
				require.NoError(t, err, "Unexpected error from stream interceptor")
			}
			require.Equal(t, tt.wantCalled, called, "Handler called status mismatch")
		})
	}
}
