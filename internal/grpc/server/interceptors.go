package server

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	basegrpc "github.com/mikewurtz/taskman/internal/grpc"
)

// ExtractClientCNInterceptor extracts the client's Common Name and injects it into the context
// for unary operations
func ExtractClientCNInterceptor(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	commonName, err := getClientCN(ctx)
	if err != nil || commonName == "" {
		return nil, status.Errorf(codes.Unauthenticated, "failed to get client CN")
	}
	ctxWithCN := context.WithValue(ctx, basegrpc.ClientCNKey, commonName)
	respObj, err := handler(ctxWithCN, req)
	return respObj, err
}

// ExtractClientCNStreamInterceptor extracts the client's Common Name and injects it into the context
// for stream operations
func ExtractClientCNStreamInterceptor(
	srv any,
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	ctx := ss.Context()
	commonName, err := getClientCN(ctx)
	if err != nil || commonName == "" {
		return status.Errorf(codes.Unauthenticated, "failed to get client CN")
	}

	ctxWithCN := context.WithValue(ctx, basegrpc.ClientCNKey, commonName)

	wrappedStream := &wrappedServerStream{
		ServerStream: ss,
		ctx:          ctxWithCN,
	}
	err = handler(srv, wrappedStream)
	return err
}

func getClientCN(ctx context.Context) (string, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		log.Println("No peer found in context")
		return "", fmt.Errorf("no peer found in context")
	}

	var commonName string
	if p.AuthInfo != nil {
		if tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo); ok {
			if len(tlsInfo.State.PeerCertificates) > 0 {
				clientCert := tlsInfo.State.PeerCertificates[0]
				commonName = clientCert.Subject.CommonName
				log.Printf("Client CN: %s", commonName)
			}
		}
	}
	return commonName, nil
}

// wrappedServerStream wraps grpc.ServerStream to replace the context
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

// Context returns the new context we created
func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}
