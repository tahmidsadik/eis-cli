package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"bitbucket.org/cover42/eiscli/internal/bitbucket"
	"bitbucket.org/cover42/eiscli/internal/config"
	"bitbucket.org/cover42/eiscli/internal/git"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var (
	compareWithService string
	compareVarType     string
)

// ComparedVariable holds variable data from both services for comparison
type ComparedVariable struct {
	Key           string
	SourceValue   string
	SourceSecured bool
	SourceType    string // "Repository", "Test", "Staging", "Production", etc.
	TargetValue   string
	TargetSecured bool
	TargetType    string // "Repository", "Test", "Staging", "Production", etc.
	InSource      bool
	InTarget      bool
}

var svcVariablesCompareCmd = &cobra.Command{
	Use:   "compare [service-name]",
	Short: "Compare variables between two services",
	Long: `Compare deployment and repository variables between two services.

This command fetches variables from both services and displays them in comparison tables:

1. Common Variables: Variables that exist in both services, showing values and types side by side
   to easily identify differences.

2. Unique Variables: Variables that exist in only one service, displayed with clear
   separation between source (left) and target (right) services.

By default, compares combined variables (repository + Test environment).
Use --type to compare specific variable types:
  - combined:    Repository + Test environment (default)
  - repository:  Repository-level variables only
  - test:        Test environment variables only
  - staging:     Staging environment variables only
  - production:  Production environment variables only

If service-name is not provided, it will be auto-detected from the git repository
in the current directory (based on the git remote URL).

Examples:
  # Compare current repo's variables with another service (default: combined)
  eiscli vars compare --with other-service

  # Compare only repository variables
  eiscli vars compare --with other-service --type repository

  # Compare only Test environment variables
  eiscli vars compare --with other-service --type test

  # Compare Production environment variables
  eiscli vars compare my-service --with other-service --type production`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Validate --with flag
		if compareWithService == "" {
			fmt.Println("Error: --with flag is required to specify the service to compare against")
			fmt.Println("\nUsage: eiscli vars compare [source-service] --with <target-service>")
			return
		}

		// Validate --type flag
		validTypes := map[string]bool{
			"combined": true, "repository": true, "test": true,
			"staging": true, "production": true,
		}
		if !validTypes[strings.ToLower(compareVarType)] {
			fmt.Printf("Error: Invalid type '%s'. Valid types: combined, repository, test, staging, production\n", compareVarType)
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

		// Get source service name
		sourceService := ""
		if len(args) > 0 {
			sourceService = args[0]
		}

		// Auto-detect source service name from git repository if not provided
		if sourceService == "" {
			detectedSlug, err := git.DetectRepositorySlug()
			if err != nil {
				fmt.Println("Error: No service name provided and could not auto-detect from git repository")
				fmt.Printf("  %v\n", err)
				fmt.Println("\nUsage:")
				fmt.Println("  1. Run this command from within a git repository, or")
				fmt.Println("  2. Provide a service name: eiscli vars compare <service-name> --with <other-service>")
				return
			}
			sourceService = detectedSlug
			fmt.Printf("Auto-detected source service from git repository: %s\n", sourceService)
		}

		// Ensure source and target are different
		if sourceService == compareWithService {
			fmt.Println("Error: Source and target services must be different")
			return
		}

		// Execute comparison
		executeVariablesComparison(client, sourceService, compareWithService, strings.ToLower(compareVarType))
	},
}

// fetchServiceVariables fetches variables for a service based on the type filter
func fetchServiceVariables(client *bitbucket.Client, serviceName, varType string) (map[string]*ComparedVariable, error) {
	switch varType {
	case "repository":
		return fetchRepositoryVariables(client, serviceName)
	case "test", "staging", "production":
		return fetchDeploymentVariables(client, serviceName, varType)
	default: // "combined"
		return fetchCombinedVariables(client, serviceName)
	}
}

func fetchRepositoryVariables(client *bitbucket.Client, serviceName string) (map[string]*ComparedVariable, error) {
	variables := make(map[string]*ComparedVariable)

	repoVars, err := client.GetRepositoryVariables(serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repository variables: %w", err)
	}

	for _, v := range repoVars {
		value := v.Value
		if v.Secured {
			value = "********"
		}
		variables[v.Key] = &ComparedVariable{
			Key:           v.Key,
			SourceValue:   value,
			SourceSecured: v.Secured,
			SourceType:    "Repository",
			InSource:      true,
		}
	}

	return variables, nil
}

func fetchDeploymentVariables(client *bitbucket.Client, serviceName, envName string) (map[string]*ComparedVariable, error) {
	variables := make(map[string]*ComparedVariable)

	// Capitalize environment name for display and matching
	displayName := capitalizeFirst(envName)

	environments, err := client.GetDeploymentEnvironments(serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch deployment environments: %w", err)
	}

	var targetEnv *bitbucket.Environment
	for _, env := range environments {
		if strings.EqualFold(env.Name, envName) {
			targetEnv = env
			break
		}
	}

	if targetEnv == nil {
		return variables, nil // No matching environment, return empty
	}

	deployVars, err := client.GetDeploymentVariablesForEnv(serviceName, targetEnv.UUID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s environment variables: %w", displayName, err)
	}

	for _, v := range deployVars {
		value := v.Value
		if v.Secured {
			value = "********"
		}
		variables[v.Key] = &ComparedVariable{
			Key:           v.Key,
			SourceValue:   value,
			SourceSecured: v.Secured,
			SourceType:    displayName,
			InSource:      true,
		}
	}

	return variables, nil
}

func fetchCombinedVariables(client *bitbucket.Client, serviceName string) (map[string]*ComparedVariable, error) {
	variables := make(map[string]*ComparedVariable)

	// Fetch repository variables
	repoVars, err := client.GetRepositoryVariables(serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repository variables: %w", err)
	}

	for _, v := range repoVars {
		value := v.Value
		if v.Secured {
			value = "********"
		}
		variables[v.Key] = &ComparedVariable{
			Key:           v.Key,
			SourceValue:   value,
			SourceSecured: v.Secured,
			SourceType:    "Repository",
			InSource:      true,
		}
	}

	// Fetch Test environment variables
	environments, err := client.GetDeploymentEnvironments(serviceName)
	if err != nil {
		// Don't fail, just warn
		fmt.Printf("Warning: Could not fetch deployment environments for %s: %v\n", serviceName, err)
		return variables, nil
	}

	var testEnv *bitbucket.Environment
	for _, env := range environments {
		if strings.EqualFold(env.Name, "Test") {
			testEnv = env
			break
		}
	}

	if testEnv != nil {
		deployVars, err := client.GetDeploymentVariablesForEnv(serviceName, testEnv.UUID)
		if err != nil {
			fmt.Printf("Warning: Could not fetch Test environment variables for %s: %v\n", serviceName, err)
		} else {
			for _, v := range deployVars {
				value := v.Value
				if v.Secured {
					value = "********"
				}
				variables[v.Key] = &ComparedVariable{
					Key:           v.Key,
					SourceValue:   value,
					SourceSecured: v.Secured,
					SourceType:    "Test",
					InSource:      true,
				}
			}
		}
	}

	return variables, nil
}

func executeVariablesComparison(client *bitbucket.Client, sourceService, targetService, varType string) {
	var typeLabel string
	if varType == "combined" {
		typeLabel = "Repository + Test"
	} else {
		typeLabel = capitalizeFirst(varType)
	}

	fmt.Printf("\nComparing %s variables: %s vs %s\n", typeLabel, sourceService, targetService)
	fmt.Println(strings.Repeat("=", 80))

	// Fetch variables from source service
	fmt.Printf("\nFetching variables from %s...\n", sourceService)
	sourceVars, err := fetchServiceVariables(client, sourceService, varType)
	if err != nil {
		fmt.Printf("Error fetching variables from %s: %v\n", sourceService, err)
		return
	}

	// Fetch variables from target service
	fmt.Printf("Fetching variables from %s...\n", targetService)
	targetVarsRaw, err := fetchServiceVariables(client, targetService, varType)
	if err != nil {
		fmt.Printf("Error fetching variables from %s: %v\n", targetService, err)
		return
	}

	// Build comparison map
	comparisonMap := make(map[string]*ComparedVariable)

	// Add source variables
	for key, v := range sourceVars {
		comparisonMap[key] = &ComparedVariable{
			Key:           key,
			SourceValue:   v.SourceValue,
			SourceSecured: v.SourceSecured,
			SourceType:    v.SourceType,
			InSource:      true,
			InTarget:      false,
		}
	}

	// Add/merge target variables
	for key, v := range targetVarsRaw {
		if existing, exists := comparisonMap[key]; exists {
			existing.TargetValue = v.SourceValue
			existing.TargetSecured = v.SourceSecured
			existing.TargetType = v.SourceType
			existing.InTarget = true
		} else {
			comparisonMap[key] = &ComparedVariable{
				Key:           key,
				TargetValue:   v.SourceValue,
				TargetSecured: v.SourceSecured,
				TargetType:    v.SourceType,
				InSource:      false,
				InTarget:      true,
			}
		}
	}

	// Categorize variables
	var commonVars, sourceOnlyVars, targetOnlyVars []*ComparedVariable

	for _, v := range comparisonMap {
		switch {
		case v.InSource && v.InTarget:
			commonVars = append(commonVars, v)
		case v.InSource:
			sourceOnlyVars = append(sourceOnlyVars, v)
		default:
			targetOnlyVars = append(targetOnlyVars, v)
		}
	}

	// Sort by type (deployment first, then repository) and then alphabetically by key
	sortByTypeAndKey := func(vars []*ComparedVariable, useSourceType bool) {
		sort.Slice(vars, func(i, j int) bool {
			var typeI, typeJ string
			if useSourceType {
				typeI, typeJ = vars[i].SourceType, vars[j].SourceType
			} else {
				typeI, typeJ = vars[i].TargetType, vars[j].TargetType
			}
			// Repository comes last, deployment environments come first
			iIsRepo := typeI == "Repository"
			jIsRepo := typeJ == "Repository"
			if iIsRepo != jIsRepo {
				return !iIsRepo // non-Repository (deployment) comes first
			}
			// If same category, sort by type name, then by key
			if typeI != typeJ {
				return typeI < typeJ
			}
			return vars[i].Key < vars[j].Key
		})
	}

	sortByTypeAndKey(commonVars, true)
	sortByTypeAndKey(sourceOnlyVars, true)
	sortByTypeAndKey(targetOnlyVars, false)

	// Display summary
	fmt.Printf("\nSummary:\n")
	fmt.Printf("  %s: %d variable(s)\n", sourceService, len(sourceVars))
	fmt.Printf("  %s: %d variable(s)\n", targetService, len(targetVarsRaw))
	fmt.Printf("  Common: %d variable(s)\n", len(commonVars))
	fmt.Printf("  Unique to %s: %d variable(s)\n", sourceService, len(sourceOnlyVars))
	fmt.Printf("  Unique to %s: %d variable(s)\n", targetService, len(targetOnlyVars))

	// Display common variables table
	if len(commonVars) > 0 {
		displayCommonVariablesTable(commonVars, sourceService, targetService)
	}

	// Display unique variables table
	if len(sourceOnlyVars) > 0 || len(targetOnlyVars) > 0 {
		displayUniqueVariablesTable(sourceOnlyVars, targetOnlyVars, sourceService, targetService)
	}

	if len(commonVars) == 0 && len(sourceOnlyVars) == 0 && len(targetOnlyVars) == 0 {
		fmt.Println("\nNo variables found in either service.")
	}
}

func displayCommonVariablesTable(vars []*ComparedVariable, sourceService, targetService string) {
	fmt.Printf("\n%s\n", strings.Repeat("=", 80))
	fmt.Printf("Common Variables (%d)\n", len(vars))
	fmt.Printf("%s\n\n", strings.Repeat("=", 80))

	// Define colors
	yellowColor := color.New(color.FgYellow).SprintFunc()
	greenColor := color.New(color.FgGreen).SprintFunc()

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Variable", fmt.Sprintf("%s Value", sourceService), fmt.Sprintf("%s Value", targetService), "Match", "Source Type", "Target Type")

	for _, v := range vars {
		var matchStatus string
		if v.SourceValue == v.TargetValue {
			matchStatus = greenColor("=")
		} else {
			matchStatus = yellowColor("!=")
		}

		// Truncate long values for display
		sourceVal := truncateValue(v.SourceValue, 25)
		targetVal := truncateValue(v.TargetValue, 25)

		table.Append(v.Key, sourceVal, targetVal, matchStatus, v.SourceType, v.TargetType)
	}

	table.Render()

	// Legend
	fmt.Println("\nLegend: " + greenColor("=") + " values match, " + yellowColor("!=") + " values differ")
}

func displayUniqueVariablesTable(sourceOnly, targetOnly []*ComparedVariable, sourceService, targetService string) {
	fmt.Printf("\n%s\n", strings.Repeat("=", 80))
	fmt.Printf("Unique Variables\n")
	fmt.Printf("%s\n\n", strings.Repeat("=", 80))

	// Define colors
	blueColor := color.New(color.FgBlue).SprintFunc()
	magentaColor := color.New(color.FgMagenta).SprintFunc()

	// Create side-by-side display
	table := tablewriter.NewWriter(os.Stdout)
	table.Header(
		fmt.Sprintf("%s (Source)", sourceService),
		"Type",
		"Value",
		fmt.Sprintf("%s (Target)", targetService),
		"Type",
		"Value",
	)

	// Determine max rows needed
	maxRows := len(sourceOnly)
	if len(targetOnly) > maxRows {
		maxRows = len(targetOnly)
	}

	for i := 0; i < maxRows; i++ {
		var sourceKey, sourceType, sourceVal, targetKey, targetType, targetVal string

		if i < len(sourceOnly) {
			sourceKey = blueColor(sourceOnly[i].Key)
			sourceType = sourceOnly[i].SourceType
			sourceVal = truncateValue(sourceOnly[i].SourceValue, 15)
		}

		if i < len(targetOnly) {
			targetKey = magentaColor(targetOnly[i].Key)
			targetType = targetOnly[i].TargetType
			targetVal = truncateValue(targetOnly[i].TargetValue, 15)
		}

		table.Append(sourceKey, sourceType, sourceVal, targetKey, targetType, targetVal)
	}

	table.Render()

	// Legend
	fmt.Println("\nLegend: " + blueColor("Blue") + " = only in source, " + magentaColor("Magenta") + " = only in target")
}

func truncateValue(value string, maxLen int) string {
	if len(value) <= maxLen {
		return value
	}
	return value[:maxLen-3] + "..."
}

// capitalizeFirst capitalizes the first letter of a string
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}

func init() {
	varsCmd.AddCommand(svcVariablesCompareCmd)
	svcVariablesCompareCmd.Flags().StringVarP(&compareWithService, "with", "w", "", "Service to compare against (required)")
	svcVariablesCompareCmd.Flags().StringVarP(&compareVarType, "type", "t", "combined", "Type of variables to compare (combined, repository, test, staging, production)")
	_ = svcVariablesCompareCmd.MarkFlagRequired("with")
}
