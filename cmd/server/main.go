package main

import (
	"fmt"
	"os"

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

		if err := server.Start(); err != nil {
			return fmt.Errorf("server failed: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.Flags().StringVar(&serverAddr, "server-address", "localhost:50051",
		"The gRPC server address to expose the server on. Defaults to localhost:50051 if not set.")

}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
