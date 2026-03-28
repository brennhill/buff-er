package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// version is set at build time via ldflags.
var version = "dev"

var rootCmd = &cobra.Command{
	Use:     "buff-er",
	Short:   "Get buff while you buffer",
	Long:    "Exercise nudges during AI wait times. Learns your build times, suggests exercises when you'll be waiting.",
	Version: version,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		// If invoked as a hook subcommand, always exit 0 to avoid breaking the AI workflow.
		// Only exit non-zero for non-hook commands (install, doctor, etc.)
		if isHookInvocation() {
			os.Exit(0)
		}
		os.Exit(1)
	}
}

func isHookInvocation() bool {
	for _, arg := range os.Args {
		if arg == "hook" {
			return true
		}
	}
	return false
}

func init() {
	rootCmd.AddCommand(hookCmd)
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(uninstallCmd)
	rootCmd.AddCommand(doctorCmd)
}
