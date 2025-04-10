package commands

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mikewurtz/taskman/internal/grpc/client"
)

var startCmd = &cobra.Command{
	Use:   `start --user-id <user-id> [--server-address <host:port>] [--help] -- <command> [args...]`,
	Short: "Start a new task by executing the specified command",
	Long: `Start a new task by executing the specified command. The --user-id flag is required to identify the client initiating the request.

Arguments:
  <command> [args...]
        The command to execute, followed by any optional arguments. The command should be passed as:
        - A command with space-separated arguments (e.g., ls /myFolder or sh -c 'command with spaces')
        The binary can be a full path or must exist in the system's PATH.
        Example: "ls"

Options:
  --user-id <user-id>
        The user or client ID issuing the request (e.g., client001). This flag is required.
  --server-address <host:port>
        The gRPC server address to connect to (e.g., localhost:50051). Defaults to localhost:50051 if not set.
  --help
        Display help information for the start command.`,
	Example:       `$ taskman start --user-id client001 -- ls /myFolder`,
	Args:          cobra.MinimumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {

		command := args[0]
		if command == "" {
			if err := cmd.Usage(); err != nil {
				return fmt.Errorf("failed to display usage: %w", err)
			}
			return errors.New("command is required")
		}
		var cmdArgs []string
		if len(args) > 1 {
			cmdArgs = args[1:]
		}

		manager, err := client.NewManager(userID, serverAddr)
		if err != nil {
			return fmt.Errorf("failed to set up gRPC client: %w", err)
		}
		defer func() {
			if closeErr := manager.Close(); closeErr != nil {
				if _, logErr := fmt.Fprintf(cmd.OutOrStderr(), "failed to close manager: %v\n", closeErr); logErr != nil {
					// Fallback to fmt.Printf output if logging to cmd.OutOrStderr fails.
					fmt.Printf("failed to log close error: %v\n", logErr)
				}
			}
		}()

		taskID, err := manager.StartTask(cmd.Context(), command, cmdArgs)
		if err != nil {
			return fmt.Errorf("failed to start task: %w", err)
		}

		output := fmt.Sprintf("TASK ID\n-------\n%s\n", taskID)
		if _, err = fmt.Fprint(cmd.OutOrStdout(), output); err != nil {
			return fmt.Errorf("failed to print output: %w", err)
		}
		return nil
	},
}
