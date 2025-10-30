package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "eiscli",
	Short: "EIS CLI - Manage services and repositories of the EIS platform",
	Long: `EIS CLI is a command-line tool for developers working with the EIS platform.
It helps manage services, repositories, pipelines, and deployment configurations.`,
	Version: "0.1.0",
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}

// ExecuteContext is used for testing
func ExecuteContext() error {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	return nil
}
