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
	Example:       `$ taskman-server --server-address localhost:50051`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		server, err := server.New(cmd.Context(), serverAddr)
		if err != nil {
			return fmt.Errorf("failed to initialize server: %w", err)
		}

		startErrCh := make(chan error, 1)

		// Start the server in a goroutine so we can handle shutdown signals
		go func() {
			defer close(startErrCh)
			if err := server.Start(); err != nil {
				startErrCh <- fmt.Errorf("server exited with error: %w", err)
			} else {
				startErrCh <- nil
			}
		}()
		select {
		case <-cmd.Context().Done():
			log.Println("Shutdown signal received. Stopping server...")
			server.Shutdown()
			log.Println("Server stopped")
			return nil

		case err := <-startErrCh:
			if err != nil {
				log.Printf("Server exited with error: %v", err)
				return err
			}
			log.Println("Server exited cleanly before signal")
			return nil
		}
	},
}

func init() {
	rootCmd.Flags().StringVar(&serverAddr, "server-address", "localhost:50051",
		"The gRPC server address to expose the server on. Defaults to localhost:50051 if not set.")

}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		log.Println(err)
		// Call stop directly to ensure the context is stopped before we exit
		stop()
		os.Exit(1)
	}
}
