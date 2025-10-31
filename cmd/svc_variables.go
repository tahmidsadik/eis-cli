package cmd

import (
	"fmt"
	"os"
	"strings"

	"bitbucket.org/cover42/eiscli/internal/bitbucket"
	"bitbucket.org/cover42/eiscli/internal/config"
	"bitbucket.org/cover42/eiscli/internal/git"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var (
	variableType    string
	environmentName string
	showAllEnvs     bool
	autoCreateEnv   bool
	envTypeOverride string
)

var varsCmd = &cobra.Command{
	Use:   "vars [service-name]",
	Short: "List Bitbucket deployment and repository variables",
	Long: `Display deployment variables and repository variables configured in Bitbucket.
These variables are used in pipeline builds and deployments.

By default, shows deployment variables for the "Test" environment.
Use --type repository to view repository-level variables.
Use --env to filter by a specific environment (deployment variables only).
Use --all to show all environments (deployment variables only).

If the specified environment doesn't exist, you'll be prompted to create it.
Use --auto-create-env to create missing environments without prompting.
Use --env-type to override the inferred environment type.

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
				fmt.Println("  2. Provide a service name: eiscli vars <service-name>")
				return
			}
			serviceName = detectedSlug
			fmt.Printf("Auto-detected service from git repository: %s\n\n", serviceName)
		}

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

		// Handle based on variable type
		if variableType == "repository" {
			displayRepositoryVariables(client, serviceName)
		} else { // deployment is default
			displayDeploymentVariables(client, serviceName)
		}
	},
}

func displayRepositoryVariables(client *bitbucket.Client, serviceName string) {
	fmt.Printf("Repository Variables for: %s\n\n", serviceName)

	variables, err := client.GetRepositoryVariables(serviceName)
	if err != nil {
		fmt.Printf("Error fetching repository variables: %v\n", err)
		return
	}

	if len(variables) == 0 {
		fmt.Println("No repository variables found.")
		return
	}

	// Create table
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Name", "Value", "Secured")

	for _, v := range variables {
		value := v.Value
		if v.Secured {
			value = "********"
		}
		securedStr := "No"
		if v.Secured {
			securedStr = "Yes"
		}
		table.Append(v.Key, value, securedStr)
	}

	table.Render()
	fmt.Printf("\nTotal: %d variable(s)\n", len(variables))
}

func displayDeploymentVariables(client *bitbucket.Client, serviceName string) {
	// Fetch all environments
	environments, err := client.GetDeploymentEnvironments(serviceName)
	if err != nil {
		fmt.Printf("Error fetching deployment environments: %v\n", err)
		return
	}

	if len(environments) == 0 {
		fmt.Println("No deployment environments found for this repository.")
		return
	}

	// If --all flag is set, display variables for all environments
	if showAllEnvs {
		displayAllEnvironmentVariables(client, serviceName, environments)
		return
	}

	// Find the matching environment
	var targetEnv *bitbucket.Environment
	for _, env := range environments {
		if strings.EqualFold(env.Name, environmentName) {
			targetEnv = env
			break
		}
	}

	// If environment not found, try to create it
	if targetEnv == nil {
		// Load config to check for auto-create setting
		cfg, err := config.Load()
		shouldAutoCreate := autoCreateEnv
		if err == nil && cfg.Deployment.AutoCreateEnvironments {
			shouldAutoCreate = true
		}

		// Try to ensure environment exists (will prompt or auto-create)
		createdEnv, err := bitbucket.EnsureEnvironmentExists(
			client,
			serviceName,
			environmentName,
			shouldAutoCreate,
			envTypeOverride,
		)

		if err != nil {
			fmt.Printf("\nError: %v\n", err)
			return
		}

		targetEnv = createdEnv
		fmt.Println()
	}

	// Fetch variables for the target environment
	fmt.Printf("Deployment Variables for: %s (Environment: %s)\n\n", serviceName, targetEnv.Name)

	variables, err := client.GetDeploymentVariablesForEnv(serviceName, targetEnv.UUID)
	if err != nil {
		fmt.Printf("Error fetching deployment variables: %v\n", err)
		return
	}

	if len(variables) == 0 {
		fmt.Printf("No deployment variables found for environment '%s'.\n", targetEnv.Name)
		return
	}

	// Create table
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Name", "Value", "Secured")

	for _, v := range variables {
		value := v.Value
		if v.Secured {
			value = "********"
		}
		securedStr := "No"
		if v.Secured {
			securedStr = "Yes"
		}
		table.Append(v.Key, value, securedStr)
	}

	table.Render()
	fmt.Printf("\nTotal: %d variable(s)\n", len(variables))
}

func displayAllEnvironmentVariables(client *bitbucket.Client, serviceName string, environments []*bitbucket.Environment) {
	fmt.Printf("Deployment Variables for: %s (All Environments)\n\n", serviceName)

	totalVars := 0

	for _, env := range environments {
		fmt.Printf("Environment: %s (%s)\n", env.Name, env.Type)
		fmt.Println(strings.Repeat("-", 80))

		variables, err := client.GetDeploymentVariablesForEnv(serviceName, env.UUID)
		if err != nil {
			fmt.Printf("  Error fetching variables: %v\n\n", err)
			continue
		}

		if len(variables) == 0 {
			fmt.Println("  No variables found.")
			fmt.Println()
			continue
		}

		// Create table
		table := tablewriter.NewWriter(os.Stdout)
		table.Header("Name", "Value", "Secured")

		for _, v := range variables {
			value := v.Value
			if v.Secured {
				value = "********"
			}
			securedStr := "No"
			if v.Secured {
				securedStr = "Yes"
			}
			table.Append(v.Key, value, securedStr)
		}

		table.Render()
		fmt.Printf("  Total: %d variable(s)\n\n", len(variables))
		totalVars += len(variables)
	}

	fmt.Printf("Grand Total: %d variable(s) across %d environment(s)\n", totalVars, len(environments))
}

func init() {
	rootCmd.AddCommand(varsCmd)
	varsCmd.Flags().StringVarP(&variableType, "type", "t", "deployment", "Type of variables to display (deployment, repository)")
	varsCmd.Flags().StringVarP(&environmentName, "env", "e", "Test", "Environment name to filter (deployment variables only)")
	varsCmd.Flags().BoolVarP(&showAllEnvs, "all", "a", false, "Show variables for all environments (deployment variables only)")
	varsCmd.Flags().BoolVar(&autoCreateEnv, "auto-create-env", false, "Automatically create missing environments without prompting")
	varsCmd.Flags().StringVar(&envTypeOverride, "env-type", "", "Override environment type (Test, Staging, Production)")
}
