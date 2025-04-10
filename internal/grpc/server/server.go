package server

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/mikewurtz/taskman/certs"
	pb "github.com/mikewurtz/taskman/gen/proto"
	basegrpc "github.com/mikewurtz/taskman/internal/grpc"
)

// Wraps the grpcServer and listener together
type Server struct {
	grpcServer *grpc.Server
	listener   net.Listener
}

// New sets up the gRPC server and listener with mTLS authentication using TLS v1.3
// Includes interceptors for injecting the client CN into the context for unary and stream calls
func New(serverAddr string) (*Server, error) {
	cert, err := basegrpc.LoadTLSCert(certs.ServerCertName)
	if err != nil {
		return nil, fmt.Errorf("loading server cert: %w", err)
	}

	caPool, err := basegrpc.LoadCACertPool()
	if err != nil {
		return nil, fmt.Errorf("failed to load CA certificate: %w", err)
	}

	// Since we are using TLS v1.3 we do not need to specify cipher suites
	// these are fixed for Go. See: https://github.com/golang/go/blob/master/src/crypto/tls/common.go#L688-L697
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caPool,
		MinVersion:   tls.VersionTLS13,
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP256,
			tls.CurveP384,
		},
	}

	grpcServer := grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsConfig)),
		grpc.ChainUnaryInterceptor(ExtractClientCNInterceptor),
		grpc.ChainStreamInterceptor(ExtractClientCNStreamInterceptor))

	pb.RegisterTaskManagerServer(grpcServer, NewTaskManagerServer())

	lis, err := net.Listen("tcp", serverAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	return &Server{
		grpcServer: grpcServer,
		listener:   lis,
	}, nil
}

// Start starts the gRPC server
func (s *Server) Start() error {
	log.Printf("Server listening on %v", s.listener.Addr())
	return s.grpcServer.Serve(s.listener)
}

func (s *Server) Addr() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return ""
}

func (s *Server) Stop() {
	s.grpcServer.GracefulStop()
}
