package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/mikewurtz/taskman/certs"
	pb "github.com/mikewurtz/taskman/gen/proto"
	basegrpc "github.com/mikewurtz/taskman/internal/grpc"
	"github.com/mikewurtz/taskman/internal/task/cgroups"
)

// Wraps the grpcServer and listener together
type Server struct {
	grpcServer *grpc.Server
	listener   net.Listener
	taskServer *taskManagerServer
}

// New sets up the gRPC server and listener with mTLS authentication using TLS v1.3
// Includes interceptors for injecting the client CN into the context for unary and stream calls
func New(ctx context.Context, serverAddr string) (*Server, error) {
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

	taskServer := NewTaskManagerServer(ctx)
	pb.RegisterTaskManagerServer(grpcServer, taskServer)

	lis, err := net.Listen("tcp", serverAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	return &Server{
		grpcServer: grpcServer,
		listener:   lis,
		taskServer: taskServer,
	}, nil
}

// Start starts the gRPC server
func (s *Server) Start() error {
	// first check if the cgroup v2 controllers are enabled
	err := cgroups.CheckAndEnableCgroupV2Controllers("/sys/fs/cgroup/cgroup.subtree_control", []string{"cpu", "memory", "io"})
	if err != nil {
		log.Printf("failed to check cgroup v2 controllers: %v", err)
		return err
	}

	log.Printf("Server listening on %v (Ctrl+C to stop)", s.listener.Addr())
	return s.grpcServer.Serve(s.listener)
}

func (s *Server) Addr() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return ""
}

// Shutdown shuts down the gRPC server and waits for all tasks to complete
func (s *Server) Shutdown() {
	log.Println("Shutting down gRPC server...")

	// GracefulStop with timeout
	done := make(chan struct{})
	go func() {
		s.grpcServer.GracefulStop()
		close(done)
	}()

	// give the ongoing RPCs a chance to complete
	// TODO: make this timeout configurable
	select {
	case <-done:
		log.Println("gRPC server stopped gracefully.")
	case <-time.After(10 * time.Second):
		log.Println("GracefulStop timed out; forcing shutdown.")
		s.grpcServer.Stop()
	}

	log.Println("Waiting for all tasks to complete...")
	if err := s.taskServer.taskManager.WaitForTasks(); err != nil {
		log.Printf("Error waiting for tasks to complete: %v", err)
	}
}
