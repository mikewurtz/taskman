package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/mikewurtz/taskman/internal/grpc/server"
)

var serverAddr string

var rootCmd = &cobra.Command{
	Use:   "taskman-server",
	Short: "Taskman server manages task lifecycle and streams output to clients",
	Long: `This service manages task lifecycle (start, stop, status) and streams output to clients 
over a secure mTLS connection.`,
	Example: `$ taskman-server --server-address localhost:50051`,
	RunE: func(cmd *cobra.Command, args []string) error {
		server, err := server.NewServer(serverAddr)
		if err != nil {
			return fmt.Errorf("failed to initialize server: %w", err)
		}

		// Start the server in a goroutine so we can handle signals
		go func() {
			if err := server.Start(); err != nil {
				log.Printf("server exited with error: %v", err)
			}
		}()

		// Set up signal handling for SIGINT and SIGTERM
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		log.Println("Taskman server is running. Press Ctrl+C to stop.")

		// Use ctx to block until a signal is received:
		<-ctx.Done()
		log.Println("Shutdown signal received. Stopping server...")
		server.Stop()
		log.Println("Server stopped cleanly.")
		return nil
	},
}

func init() {
	rootCmd.Flags().StringVar(&serverAddr, "server-address", "localhost:50051",
		"The gRPC server address to expose the server on. Defaults to localhost:50051 if not set.")

}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
