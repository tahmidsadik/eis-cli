package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"bitbucket.org/cover42/eiscli/internal/bitbucket"
	"bitbucket.org/cover42/eiscli/internal/config"
	"bitbucket.org/cover42/eiscli/internal/git"
	"bitbucket.org/cover42/eiscli/internal/kubernetes"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var (
	addVariableType    string
	addEnvironmentName string
	addAutoCreateEnv   bool
	addEnvTypeOverride string
)

// VariableToAdd represents a variable that the user wants to add
type VariableToAdd struct {
	Key     string
	Value   string
	Secured bool
}

var svcVariablesAddCmd = &cobra.Command{
	Use:   "add [service-name]",
	Short: "Interactively add variables to Bitbucket",
	Long: `Interactively add repository or deployment variables to Bitbucket.

This command prompts you to enter multiple variables with their keys, values, and sensitivity.
Variables are automatically detected as secured based on naming patterns (PASSWORD, SECRET, KEY, TOKEN, etc.),
but you can override the suggestion for each variable.

By default, creates repository-level variables. Use --type deployment to create deployment variables.

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
				fmt.Println("  2. Provide a service name: eiscli vars add <service-name>")
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

		// Handle deployment variables
		var targetEnv *bitbucket.Environment
		if addVariableType == "deployment" {
			if addEnvironmentName == "" {
				fmt.Println("Error: --env flag is required when using --type deployment")
				fmt.Println("\nUsage: eiscli vars add [service-name] --type deployment --env <environment>")
				return
			}

			// Load config to check for auto-create setting
			shouldAutoCreate := addAutoCreateEnv
			if cfg.Deployment.AutoCreateEnvironments {
				shouldAutoCreate = true
			}

			// Ensure environment exists (will create if necessary)
			createdEnv, err := bitbucket.EnsureEnvironmentExists(
				client,
				serviceName,
				addEnvironmentName,
				shouldAutoCreate,
				addEnvTypeOverride,
			)
			if err != nil {
				fmt.Printf("\nError: %v\n", err)
				return
			}
			targetEnv = createdEnv
			fmt.Printf("Target environment: %s (UUID: %s)\n\n", targetEnv.Name, targetEnv.UUID)
		}

		// Collect variables interactively
		variables, err := collectVariablesInteractively()
		if err != nil {
			fmt.Printf("\nError: %v\n", err)
			return
		}

		if len(variables) == 0 {
			fmt.Println("\nNo variables were added.")
			return
		}

		// Display final preview
		displayVariablesTable(variables)

		// Confirm and create
		if addVariableType == "deployment" {
			if err := confirmAndCreateDeploymentVariables(client, serviceName, targetEnv.UUID, variables); err != nil {
				fmt.Printf("\nError: %v\n", err)
				os.Exit(1)
			}
		} else {
			if err := confirmAndCreateRepositoryVariables(client, serviceName, variables); err != nil {
				fmt.Printf("\nError: %v\n", err)
				os.Exit(1)
			}
		}
	},
}

func collectVariablesInteractively() ([]VariableToAdd, error) {
	reader := bufio.NewReader(os.Stdin)
	variables := []VariableToAdd{}

	fmt.Println("Enter variables (press Enter with an empty key to finish):")
	fmt.Println(strings.Repeat("=", 80))

	for {
		// Prompt for key
		fmt.Print("\nVariable Key (or press Enter to finish): ")
		key, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read key: %w", err)
		}
		key = strings.TrimSpace(key)

		// Empty key means user is done
		if key == "" {
			break
		}

		// Prompt for value (cannot be empty)
		var value string
		for {
			fmt.Print("Variable Value: ")
			valueInput, err := reader.ReadString('\n')
			if err != nil {
				return nil, fmt.Errorf("failed to read value: %w", err)
			}
			value = strings.TrimSpace(valueInput)

			if value != "" {
				break
			}
			fmt.Println("⚠️  Value cannot be empty. Please enter a value.")
		}

		// Auto-detect if secured and ask for confirmation
		autoDetectedSecured := kubernetes.IsSecuredVariable(key)
		var secured bool

		if autoDetectedSecured {
			fmt.Print("Mark as secured? [Y/n]: ")
		} else {
			fmt.Print("Mark as secured? [y/N]: ")
		}

		securedInput, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read secured input: %w", err)
		}
		securedInput = strings.TrimSpace(strings.ToLower(securedInput))

		// Parse response based on auto-detection
		if autoDetectedSecured {
			// Default is YES for auto-detected secrets
			secured = securedInput != "n" && securedInput != "no"
		} else {
			// Default is NO for non-secrets
			secured = securedInput == "y" || securedInput == "yes"
		}

		variables = append(variables, VariableToAdd{
			Key:     key,
			Value:   value,
			Secured: secured,
		})

		fmt.Printf("✓ Added: %s\n", key)
	}

	return variables, nil
}

func displayVariablesTable(variables []VariableToAdd) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("Variables to be created:")
	fmt.Println(strings.Repeat("=", 80))

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Key", "Value", "Is_Sensitive")

	for _, v := range variables {
		securedStr := "No"
		if v.Secured {
			securedStr = "Yes"
		}
		table.Append(v.Key, v.Value, securedStr)
	}

	table.Render()
	fmt.Printf("\nTotal: %d variable(s)\n", len(variables))
}

func confirmAndCreateRepositoryVariables(client *bitbucket.Client, serviceName string, variables []VariableToAdd) error {
	// Final confirmation
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Printf("Create these %d repository variable(s)? [y/N]: ", len(variables))

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read confirmation: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		fmt.Println("\nCanceled. No variables were created.")
		return nil
	}

	// Create variables
	fmt.Println("\nCreating repository variables...")
	return createRepositoryVariables(client, serviceName, variables)
}

func confirmAndCreateDeploymentVariables(client *bitbucket.Client, serviceName, envUUID string, variables []VariableToAdd) error {
	// Final confirmation
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Printf("Create these %d deployment variable(s)? [y/N]: ", len(variables))

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read confirmation: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		fmt.Println("\nCanceled. No variables were created.")
		return nil
	}

	// Create variables
	fmt.Println("\nCreating deployment variables...")
	return createDeploymentVariables(client, serviceName, envUUID, variables)
}

func createRepositoryVariables(client *bitbucket.Client, serviceName string, variables []VariableToAdd) error {
	successCount := 0
	failCount := 0

	greenColor := color.New(color.FgGreen).SprintFunc()
	redColor := color.New(color.FgRed).SprintFunc()

	for _, v := range variables {
		err := client.CreateRepositoryVariable(serviceName, v.Key, v.Value, v.Secured)
		if err != nil {
			fmt.Printf("  %s Failed to create %s: %v\n", redColor("✗"), v.Key, err)
			failCount++
		} else {
			securedLabel := ""
			if v.Secured {
				securedLabel = " (secured)"
			}
			fmt.Printf("  %s Created %s%s\n", greenColor("✓"), v.Key, securedLabel)
			successCount++
		}
	}

	fmt.Println()
	if failCount > 0 {
		fmt.Printf("Summary: %s, %s\n",
			greenColor(fmt.Sprintf("%d created", successCount)),
			redColor(fmt.Sprintf("%d failed", failCount)))
		return fmt.Errorf("some variables failed to create")
	} else {
		fmt.Printf("Summary: %s\n", greenColor(fmt.Sprintf("%d created", successCount)))
	}

	return nil
}

func createDeploymentVariables(client *bitbucket.Client, serviceName, envUUID string, variables []VariableToAdd) error {
	successCount := 0
	failCount := 0

	greenColor := color.New(color.FgGreen).SprintFunc()
	redColor := color.New(color.FgRed).SprintFunc()

	for _, v := range variables {
		err := client.CreateDeploymentVariable(serviceName, envUUID, v.Key, v.Value, v.Secured)
		if err != nil {
			fmt.Printf("  %s Failed to create %s: %v\n", redColor("✗"), v.Key, err)
			failCount++
		} else {
			securedLabel := ""
			if v.Secured {
				securedLabel = " (secured)"
			}
			fmt.Printf("  %s Created %s%s\n", greenColor("✓"), v.Key, securedLabel)
			successCount++
		}
	}

	fmt.Println()
	if failCount > 0 {
		fmt.Printf("Summary: %s, %s\n",
			greenColor(fmt.Sprintf("%d created", successCount)),
			redColor(fmt.Sprintf("%d failed", failCount)))
		return fmt.Errorf("some variables failed to create")
	} else {
		fmt.Printf("Summary: %s\n", greenColor(fmt.Sprintf("%d created", successCount)))
	}

	return nil
}

func init() {
	varsCmd.AddCommand(svcVariablesAddCmd)
	svcVariablesAddCmd.Flags().StringVarP(&addVariableType, "type", "t", "repository", "Type of variables (repository, deployment)")
	svcVariablesAddCmd.Flags().StringVarP(&addEnvironmentName, "env", "e", "", "Environment name (required for deployment variables)")
	svcVariablesAddCmd.Flags().BoolVar(&addAutoCreateEnv, "auto-create-env", false, "Automatically create missing environments without prompting")
	svcVariablesAddCmd.Flags().StringVar(&addEnvTypeOverride, "env-type", "", "Override environment type (Test, Staging, Production)")
}
