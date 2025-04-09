package main

import (
	"fmt"
	"os"

	"github.com/mikewurtz/taskman/cmd/cli/commands"
)

func main() {
	if err := commands.RootCmd.Execute(); err != nil {
		if _, logErr := fmt.Fprintf(commands.RootCmd.ErrOrStderr(), "Error: %v\n", err); logErr != nil {
			// fall back to fmt.Print output if fmt.FPrintf fails
			fmt.Printf("failed to log error: %v\n", logErr)
		}
		os.Exit(1)
	}
}
