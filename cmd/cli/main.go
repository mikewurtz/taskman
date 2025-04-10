package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/mikewurtz/taskman/cmd/cli/commands"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := commands.RootCmd.ExecuteContext(ctx); err != nil {
		if _, logErr := fmt.Fprintf(commands.RootCmd.ErrOrStderr(), "Error: %v\n", err); logErr != nil {
			// fall back to fmt.Print output if fmt.FPrintf fails
			fmt.Printf("failed to log error: %v\n", logErr)
		}
		os.Exit(1)
	}
}
