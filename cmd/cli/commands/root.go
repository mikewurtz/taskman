package commands

import (
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
	Example: `$ taskman start --user-id client001 --server-address localhost50051 -- /bin/ls /myFolder`,
}

func init() {

	RootCmd.PersistentFlags().StringVar(&userID,
		"user-id", "", "The user or client ID issuing the request (e.g., client001)")
	RootCmd.PersistentFlags().StringVar(&serverAddr,
		"server-address", "localhost:50051", "The gRPC server address to connect to.Defaults to localhost:50051 if not set.")

	RootCmd.MarkPersistentFlagRequired("user-id")

	RootCmd.AddCommand(startCmd)
	RootCmd.AddCommand(statusCmd)
	RootCmd.AddCommand(streamCmd)
	RootCmd.AddCommand(stopCmd)
}
