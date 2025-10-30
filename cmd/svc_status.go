package cmd

import (
	"fmt"

	"bitbucket.org/cover42/eiscli/internal/git"
	"github.com/spf13/cobra"
)

var svcStatusCmd = &cobra.Command{
	Use:   "status [service-name]",
	Short: "Check the status of an EIS service",
	Long: `Display the status of a service including:
  - Repository information from Bitbucket
  - ECR registry details from AWS
  - Pipeline variables and configuration
  - Current deployment status

If service-name is not provided, it will be auto-detected from the git repository
in the current directory (based on the git remote URL).`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		serviceName := ""
		if len(args) > 0 {
			serviceName = args[0]
		}

		// Auto-detect service name from git repository if not provided
		if serviceName == "" {
			detectedSlug, err := git.DetectRepositorySlug()
			if err != nil {
				fmt.Println("Error: No service name provided and could not auto-detect from git repository")
				fmt.Printf("  %v\n", err)
				fmt.Println("\nUsage:")
				fmt.Println("  1. Run this command from within a git repository, or")
				fmt.Println("  2. Provide a service name: eiscli svc status <service-name>")
				return
			}
			serviceName = detectedSlug
			fmt.Printf("Auto-detected service from git repository: %s\n", serviceName)
		}

		fmt.Printf("Checking status for service: %s\n", serviceName)
		fmt.Println("\nRepository Information:")
		fmt.Println("  VCS: Bitbucket")
		fmt.Println("  Status: This feature is under development...")
		fmt.Println("\nECR Registry:")
		fmt.Println("  Status: This feature is under development...")
		fmt.Println("\nPipeline Variables:")
		fmt.Println("  Status: This feature is under development...")
	},
}

func init() {
	svcCmd.AddCommand(svcStatusCmd)
}
