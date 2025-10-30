package cmd

import (
	"fmt"

	"bitbucket.org/cover42/eiscli/internal/bitbucket"
	"bitbucket.org/cover42/eiscli/internal/config"
	"github.com/spf13/cobra"
)

var svcListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all repositories in the workspace",
	Long:  `Display all repositories available in the configured Bitbucket workspace.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Load configuration
		cfg, err := config.Load()
		if err != nil {
			fmt.Printf("Error loading configuration: %v\n", err)
			fmt.Println("\nPlease set the following environment variables:")
			fmt.Println("  EISCLI_BITBUCKET_USERNAME")
			fmt.Println("  EISCLI_BITBUCKET_APP_PASSWORD")
			fmt.Println("  EISCLI_BITBUCKET_WORKSPACE")
			return
		}

		// Validate configuration
		if err := cfg.Validate(); err != nil {
			fmt.Printf("Configuration error: %v\n", err)
			return
		}

		// Create Bitbucket client
		client, err := bitbucket.NewClient(cfg)
		if err != nil {
			fmt.Printf("Error creating Bitbucket client: %v\n", err)
			return
		}

		fmt.Printf("Fetching repositories from workspace: %s\n\n", cfg.Bitbucket.Workspace)

		// List repositories
		repos, err := client.ListRepositories()
		if err != nil {
			fmt.Printf("Error fetching repositories: %v\n", err)
			return
		}

		if len(repos) == 0 {
			fmt.Println("No repositories found in this workspace.")
			return
		}

		fmt.Printf("Found %d repositories:\n\n", len(repos))
		for i, repo := range repos {
			fmt.Printf("%d. %s\n", i+1, repo.Slug)
			if repo.Name != "" && repo.Name != repo.Slug {
				fmt.Printf("   Name: %s\n", repo.Name)
			}
			if repo.Description != "" {
				fmt.Printf("   Description: %s\n", repo.Description)
			}
			fmt.Println()
		}
	},
}

func init() {
	svcCmd.AddCommand(svcListCmd)
}
