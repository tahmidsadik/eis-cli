package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var svcNewCmd = &cobra.Command{
	Use:   "new [service-name]",
	Short: "Create a new EIS service",
	Long: `Create a new service in the EIS platform with all necessary scaffolding.
This command will set up the repository, ECR registry, and pipeline configuration.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		serviceName := ""
		if len(args) > 0 {
			serviceName = args[0]
		}

		if serviceName == "" {
			fmt.Println("Please provide a service name")
			fmt.Println("Usage: eiscli svc new <service-name>")
			return
		}

		fmt.Printf("Creating new service: %s\n", serviceName)
		fmt.Println("This feature is under development...")
	},
}

func init() {
	svcCmd.AddCommand(svcNewCmd)
}
