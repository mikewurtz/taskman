package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/mikewurtz/taskman/internal/grpc/client"
)

var stopCmd = &cobra.Command{
	Use:   `stop <task-id> --user-id <user-id> [--server-address <host:port>] [--help]`,
	Short: "Stop a running task by its task ID",
	Long: `Stop a running task identified by its unique task ID.

Arguments:
  <task-id>
        The unique identifier (UUID) of the task to stop.
        Example: a7da14c7-b47a-4535-a263-5bb26e503002

Options:
  --user-id <user-id>
      The user or client ID issuing the request (e.g., client001). This flag is required.
  --server-address <host:port>
      The gRPC server address to connect to (e.g., localhost:50051). Defaults to localhost:50051 if not set.
  --help
      Display help information for the stop command.`,
	Example:      `$ taskman --user-id client001 stop a7da14c7-b47a-4535-a263-5bb26e503002`,
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {

		taskID := args[0]
		if taskID == "" {
			if err := cmd.Usage(); err != nil {
				return fmt.Errorf("failed to display usage: %w", err)
			}
			return fmt.Errorf("task ID is required")
		}

		manager, err := client.NewManager(userID, serverAddr)
		if err != nil {
			return fmt.Errorf("failed to set up gRPC client: %w", err)
		}
		defer func() {
			if err := manager.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "failed to close manager: %v\n", err)
			}
		}()

		return manager.StopTask(taskID)
	},
}
