package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
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
		server, err := server.New(cmd.Context(), serverAddr)
		if err != nil {
			return fmt.Errorf("failed to initialize server: %w", err)
		}

		// Start the server in a goroutine so we can handle signals
		go func() {
			if err := server.Start(); err != nil {
				log.Printf("server exited with error: %v", err)
			}
		}()

		log.Println("Taskman server is running. Press Ctrl+C to stop.")

		// Use ctx to block until a signal is received:
		<-cmd.Context().Done()
		log.Println("Shutdown signal received. Stopping server...")

		// Stop accepting new connections and attempt to clean up all tasks
		server.Shutdown()

		log.Println("Server stopped cleanly.")
		return nil
	},
}

func init() {
	rootCmd.Flags().StringVar(&serverAddr, "server-address", "localhost:50051",
		"The gRPC server address to expose the server on. Defaults to localhost:50051 if not set.")

}

// checkCgroupV2Controllers checks if the provided cgroup v2 controllers are enabled
// Example call: checkCgroupV2Controllers("/sys/fs/cgroup/cgroup.subtree_control", []string{"cpu", "memory", "io"})
func checkCgroupV2Controllers(path string, required []string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	enabled := strings.Fields(string(data))
	controllerSet := make(map[string]bool)
	for _, ctrl := range enabled {
		controllerSet[ctrl] = true
	}

	var missing []string
	for _, ctrl := range required {
		if !controllerSet[ctrl] {
			missing = append(missing, ctrl)
		}
	}

	return missing, nil
}

func main() {
	// First check if the cgroup v2 controllers are enabled
	missing, err := checkCgroupV2Controllers("/sys/fs/cgroup/cgroup.subtree_control", []string{"cpu", "memory", "io"})
	if err != nil {
		log.Printf("failed to check cgroup v2 controllers: %v", err)
		os.Exit(1)
	}

	if len(missing) > 0 {
		for _, ctrl := range missing {
			log.Printf(`cgroup v2 controller %s not enabled. To enable it, run:
			  echo "+%s" | sudo tee /sys/fs/cgroup/cgroup.subtree_control`, ctrl, ctrl)
		}
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		log.Println(err)
		// Call stop directly to ensure the context is stopped before we exit
		stop()
		os.Exit(1)
	}
}
