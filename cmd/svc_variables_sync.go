package cmd

import (
	"bufio"
	"fmt"
	"os"
	"sort"
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
	syncEnvironment     string
	kubernetesPath      string
	applySyncChanges    bool
	syncAutoCreateEnv   bool
	syncEnvTypeOverride string
)

// VariableToSync represents a variable that needs to be synced
type VariableToSync struct {
	Key     string
	Value   string // actual value to be set
	Secured bool
	Status  string // "NEW" or "EXISTS"
	IsNew   bool   // true if variable needs to be created
}

var svcVariablesSyncCmd = &cobra.Command{
	Use:   "sync [service-name]",
	Short: "Sync deployment variables from Kubernetes templates to Bitbucket",
	Long: `Sync deployment variables from Kubernetes .env.template files to Bitbucket deployment environments.

This command reads variables from kubernetes/overlays/{env}/.env.template files and syncs them
to the corresponding Bitbucket deployment environment.

By default, shows a preview of changes (like terraform plan). Use --apply to actually create the variables.

The command only ADDS missing variables and never removes existing ones.
Variables with names containing PASSWORD, SECRET, KEY, TOKEN, etc. are automatically marked as secured.

If the target environment doesn't exist, you'll be prompted to create it.
Use --auto-create-env to create missing environments without prompting.
Use --env-type to override the inferred environment type.

Environment mapping:
  testing      → Test
  staging      → Staging
  prod         → Production
  prod-zurich  → Production-Zurich
  dev          → Development

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
				fmt.Println("  2. Provide a service name: eiscli svc variables sync <service-name> --env <environment>")
				return
			}
			serviceName = detectedSlug
			fmt.Printf("Auto-detected service from git repository: %s\n\n", serviceName)
		}

		// Validate environment parameter
		if syncEnvironment == "" {
			fmt.Println("Error: --env flag is required")
			fmt.Println("\nUsage: eiscli svc variables sync [service-name] --env <environment>")
			fmt.Println("\nAvailable environments: testing, staging, prod, prod-zurich, dev")
			return
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

		// Execute sync
		if err := executeSyncPlan(client, serviceName, syncEnvironment, kubernetesPath, applySyncChanges); err != nil {
			fmt.Printf("\nError: %v\n", err)
			os.Exit(1)
		}
	},
}

func collectVariableValues(varsToSync []VariableToSync) error {
	fmt.Println("\nEnter values for new variables:")
	fmt.Println(strings.Repeat("=", 80))

	for i := range varsToSync {
		for {
			securedLabel := ""
			if varsToSync[i].Secured {
				securedLabel = " (secured)"
			}
			fmt.Printf("\n%s%s: ", varsToSync[i].Key, securedLabel)

			reader := bufio.NewReader(os.Stdin)
			value, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read input: %w", err)
			}

			value = strings.TrimSpace(value)
			if value == "" {
				fmt.Println("⚠️  Value cannot be empty. Please enter a value.")
				continue
			}

			varsToSync[i].Value = value
			break
		}
	}

	return nil
}

func displayFinalPreview(varsToSync []VariableToSync) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("Final Preview - Variables to be created:")
	fmt.Println(strings.Repeat("=", 80))

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Variable Name", "Value", "Secured")

	for _, v := range varsToSync {
		securedStr := "No"
		if v.Secured {
			securedStr = "Yes"
		}
		table.Append(v.Key, v.Value, securedStr)
	}

	table.Render()
}

func executeSyncPlan(client *bitbucket.Client, serviceName, overlayName, k8sPath string, apply bool) error {
	// Step 1: Map overlay name to Bitbucket environment name
	envName := kubernetes.MapOverlayToEnvironment(overlayName)
	fmt.Printf("Syncing variables for environment: %s (overlay: %s)\n", envName, overlayName)
	fmt.Println(strings.Repeat("=", 80))

	// Step 2: Find and parse .env.template file
	templatePath, err := kubernetes.FindEnvTemplate(k8sPath, overlayName)
	if err != nil {
		// Show available overlays
		overlays, listErr := kubernetes.GetAvailableOverlays(k8sPath)
		if listErr == nil && len(overlays) > 0 {
			fmt.Printf("\n%v\n\n", err)
			fmt.Println("Available overlays in kubernetes folder:")
			for _, overlay := range overlays {
				mappedEnv := kubernetes.MapOverlayToEnvironment(overlay)
				fmt.Printf("  - %s (maps to: %s)\n", overlay, mappedEnv)
			}
		}
		return err
	}

	fmt.Printf("Reading template file: %s\n", templatePath)

	templateKeys, err := kubernetes.ParseEnvTemplate(templatePath)
	if err != nil {
		return fmt.Errorf("failed to parse template file: %w", err)
	}

	if len(templateKeys) == 0 {
		return fmt.Errorf("no variables found in template file")
	}

	fmt.Printf("Found %d variable(s) in template\n\n", len(templateKeys))

	// Step 3: Ensure deployment environment exists in Bitbucket
	// Load config to check for auto-create setting
	cfg, configErr := config.Load()
	shouldAutoCreate := syncAutoCreateEnv
	if configErr == nil && cfg.Deployment.AutoCreateEnvironments {
		shouldAutoCreate = true
	}

	// Ensure environment exists (will create if necessary)
	targetEnv, err := bitbucket.EnsureEnvironmentExists(
		client,
		serviceName,
		envName,
		shouldAutoCreate,
		syncEnvTypeOverride,
	)
	if err != nil {
		return err
	}

	fmt.Printf("\nTarget Bitbucket environment: %s (UUID: %s)\n\n", targetEnv.Name, targetEnv.UUID)

	// Step 4: Fetch existing variables
	existingVars, err := client.GetDeploymentVariablesForEnv(serviceName, targetEnv.UUID)
	if err != nil {
		return fmt.Errorf("failed to fetch existing deployment variables: %w", err)
	}

	// Create a map of existing variable keys with their secured status for quick lookup
	existingVarsMap := make(map[string]*bitbucket.Variable)
	for _, v := range existingVars {
		existingVarsMap[v.Key] = v
	}

	fmt.Printf("Existing variables in Bitbucket: %d\n\n", len(existingVars))

	// Step 5: Build list of ALL variables from template with their status
	var varsToDisplay []VariableToSync
	newVarCount := 0

	for _, key := range templateKeys {
		if existingVar, exists := existingVarsMap[key]; exists {
			// Variable already exists in Bitbucket
			varsToDisplay = append(varsToDisplay, VariableToSync{
				Key:     key,
				Secured: existingVar.Secured,
				Status:  "EXISTS",
				IsNew:   false,
			})
		} else {
			// Variable is new and needs to be created
			secured := kubernetes.IsSecuredVariable(key)
			varsToDisplay = append(varsToDisplay, VariableToSync{
				Key:     key,
				Secured: secured,
				Status:  "NEW",
				IsNew:   true,
			})
			newVarCount++
		}
	}

	// Sort variables: NEW first, then EXISTS
	sort.Slice(varsToDisplay, func(i, j int) bool {
		if varsToDisplay[i].IsNew != varsToDisplay[j].IsNew {
			return varsToDisplay[i].IsNew // true (NEW) comes before false (EXISTS)
		}
		return varsToDisplay[i].Key < varsToDisplay[j].Key // Then alphabetically by key
	})

	// Step 6: Display preview table
	fmt.Printf("Variables in template: %d (New: %d, Existing: %d)\n\n", len(varsToDisplay), newVarCount, len(varsToDisplay)-newVarCount)
	displaySyncPreview(varsToDisplay, newVarCount)

	if newVarCount == 0 {
		fmt.Println("\n✓ All variables from template already exist in Bitbucket")
		fmt.Println("Nothing to sync!")
		return nil
	}

	// Step 7: Apply changes if requested
	if !apply {
		fmt.Println("\n" + strings.Repeat("=", 80))
		fmt.Printf("This is a preview. Run with --apply to create the %d new variable(s).\n", newVarCount)
		return nil
	}

	// Step 8: Filter to only new variables for creation
	var newVarsToCreate []VariableToSync
	for _, v := range varsToDisplay {
		if v.IsNew {
			newVarsToCreate = append(newVarsToCreate, v)
		}
	}

	// Step 9: Collect values for new variables
	if err := collectVariableValues(newVarsToCreate); err != nil {
		return fmt.Errorf("failed to collect variable values: %w", err)
	}

	// Step 10: Display final preview with values
	displayFinalPreview(newVarsToCreate)

	// Step 11: Final confirmation
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Printf("Do you want to create these %d variable(s) with the values shown above? [y/N]: ", newVarCount)

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read confirmation: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		fmt.Println("\nSync canceled.")
		return nil
	}

	fmt.Println("\nCreating variables...")
	return applyVariableSync(client, serviceName, targetEnv.UUID, newVarsToCreate)
}

func displaySyncPreview(varsToDisplay []VariableToSync, newVarCount int) {
	// Define colors
	greenColor := color.New(color.FgGreen).SprintFunc()
	cyanColor := color.New(color.FgCyan).SprintFunc()

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Variable Name", "Secured", "Status")

	for _, v := range varsToDisplay {
		securedStr := "No"
		if v.Secured {
			securedStr = "Yes"
		}

		// Apply colors based on status
		var varName, secured, status string
		if v.IsNew {
			// Green for new variables
			varName = greenColor(v.Key)
			secured = greenColor(securedStr)
			status = greenColor(v.Status)
		} else {
			// Cyan for existing variables
			varName = cyanColor(v.Key)
			secured = cyanColor(securedStr)
			status = cyanColor(v.Status)
		}

		table.Append(varName, secured, status)
	}

	table.Render()
}

func applyVariableSync(client *bitbucket.Client, serviceName, envUUID string, varsToSync []VariableToSync) error {
	successCount := 0
	failCount := 0

	greenColor := color.New(color.FgGreen).SprintFunc()
	redColor := color.New(color.FgRed).SprintFunc()

	for _, v := range varsToSync {
		// Use the collected value for new variables
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
	} else {
		fmt.Printf("Summary: %s\n", greenColor(fmt.Sprintf("%d created", successCount)))
	}

	if failCount > 0 {
		return fmt.Errorf("some variables failed to create")
	}

	return nil
}

func init() {
	svcVariablesCmd.AddCommand(svcVariablesSyncCmd)
	svcVariablesSyncCmd.Flags().StringVarP(&syncEnvironment, "env", "e", "", "Environment to sync (required: testing, staging, prod, prod-zurich, dev)")
	svcVariablesSyncCmd.Flags().StringVarP(&kubernetesPath, "kubernetes-path", "k", "./kubernetes", "Path to kubernetes folder")
	svcVariablesSyncCmd.Flags().BoolVarP(&applySyncChanges, "apply", "a", false, "Actually apply the changes (without this, just shows preview)")
	svcVariablesSyncCmd.Flags().BoolVar(&syncAutoCreateEnv, "auto-create-env", false, "Automatically create missing environments without prompting")
	svcVariablesSyncCmd.Flags().StringVar(&syncEnvTypeOverride, "env-type", "", "Override environment type (Test, Staging, Production)")
	_ = svcVariablesSyncCmd.MarkFlagRequired("env")
}
