package cmd

import (
	"github.com/spf13/cobra"
)

var svcCmd = &cobra.Command{
	Use:   "svc",
	Short: "Manage EIS platform services",
	Long: `The svc command provides subcommands to manage services in the EIS platform.
You can create new services, check their status, view pipeline builds, and manage variables.`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(svcCmd)
}
