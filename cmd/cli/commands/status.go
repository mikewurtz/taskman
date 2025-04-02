package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mikewurtz/taskman/internal/grpc/client"
)

var statusCmd = &cobra.Command{
	Use:   `get-status <task-id> --user-id <user-id> [--server-address <host:port>] [--help]`,
	Short: "Get the status of a task by its task ID",
	Long: `Retrieve the status of a task using its unique task ID. The command displays details such as whether 
the task is running, start time, process ID, and exit code and end time if the process has ended.

Arguments:
  <task-id>             
      The UUID of the task to query (e.g., a7da14c7-b47a-4535-a263-5bb26e503002)

Options:
  --user-id <user-id>   
      The user or client ID issuing the request (e.g., client001). This flag is required.
  --server-address <host:port>
      The gRPC server address to connect to (e.g., localhost:50051). Defaults to localhost:50051 if not set.
  --help
      Display help information for the get-status command.`,
	Example: `$ taskman get-status a7da14c7-b47a-4535-a263-5bb26e503002 --user-id client001`,
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if userID == "" {
			cmd.Usage()
			return fmt.Errorf("--user-id is required")
		}
		taskID := args[0]
		manager := client.NewManager(userID, serverAddr)
		return manager.GetTaskStatus(taskID)
	},
}
