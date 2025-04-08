package commands

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

var (
	// Shared flags across commands
	userID     string
	serverAddr string
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "taskman",
	Short: "Taskman is a client for managing tasks via a gRPC server",
	Long: `A CLI tool to start, check the status, stream output, and stop tasks executed by a remote gRPC server.
This client connects to a taskman-server instance over a secure mTLS connection.`,
	Example: `  $ taskman --user-id client001 start -- /bin/ls /myFolder
  $ taskman --user-id client001 --server-address localhost:50051 get-status 123e4567-e89b-12d3-a456-426614174000
  $ taskman --user-id client001 --server-address localhost:50051 stream 123e4567-e89b-12d3-a456-426614174000
  $ taskman --user-id client001 --server-address localhost:50053 stop 123e4567-e89b-12d3-a456-426614174000`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if userID == "" {
			return fmt.Errorf("--user-id is required")
		}
		return nil
	},
}

func init() {

	RootCmd.PersistentFlags().StringVar(&userID,
		"user-id", "", "The user or client ID issuing the request (e.g., client001)")
	RootCmd.PersistentFlags().StringVar(&serverAddr,
		"server-address", "localhost:50051", "The gRPC server address to connect to.Defaults to localhost:50051 if not set.")

	err := RootCmd.MarkPersistentFlagRequired("user-id")
	if err != nil {
		log.Fatalf("failed to mark --user-id as required: %v", err)
	}

	RootCmd.AddCommand(startCmd)
	RootCmd.AddCommand(statusCmd)
	RootCmd.AddCommand(streamCmd)
	RootCmd.AddCommand(stopCmd)
}
