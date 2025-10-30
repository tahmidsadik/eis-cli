package bitbucket

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
)

// PromptCreateEnvironment prompts the user to confirm creating an environment
// Shows the inferred type and allows the user to override it
// Returns: (shouldCreate, finalEnvType, error)
func PromptCreateEnvironment(envName, inferredType string) (bool, string, error) {
	fmt.Printf("\nWould you like to create the '%s' environment? [y/N]: ", envName)

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, "", fmt.Errorf("failed to read confirmation: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		return false, "", nil
	}

	// Show inferred type and ask for confirmation
	fmt.Printf("\nInferred environment type: %s\n", inferredType)
	fmt.Printf("Is this correct? [Y/n]: ")

	typeResponse, err := reader.ReadString('\n')
	if err != nil {
		return false, "", fmt.Errorf("failed to read type confirmation: %w", err)
	}

	typeResponse = strings.TrimSpace(strings.ToLower(typeResponse))

	// If user confirms the inferred type (or just presses enter), use it
	if typeResponse == "" || typeResponse == "y" || typeResponse == "yes" {
		return true, inferredType, nil
	}

	// Otherwise, ask for the correct type
	fmt.Printf("\nAvailable environment types:\n")
	fmt.Printf("  1. %s\n", EnvironmentTypeTest)
	fmt.Printf("  2. %s\n", EnvironmentTypeStaging)
	fmt.Printf("  3. %s\n", EnvironmentTypeProduction)
	fmt.Printf("\nEnter the environment type: ")

	customTypeResponse, err := reader.ReadString('\n')
	if err != nil {
		return false, "", fmt.Errorf("failed to read custom type: %w", err)
	}

	customType := strings.TrimSpace(customTypeResponse)

	// Validate the custom type
	if err := ValidateEnvironmentType(customType); err != nil {
		return false, "", fmt.Errorf("invalid environment type: %w", err)
	}

	return true, customType, nil
}

// EnsureEnvironmentExists checks if an environment exists and creates it if necessary
// Returns the environment (either existing or newly created) or an error
func EnsureEnvironmentExists(
	client *Client,
	repoSlug string,
	envName string,
	autoCreate bool,
	envTypeOverride string,
) (*Environment, error) {
	// First, try to fetch all environments
	environments, err := client.GetDeploymentEnvironments(repoSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch deployment environments: %w", err)
	}

	// Check if the environment already exists
	for _, env := range environments {
		if strings.EqualFold(env.Name, envName) {
			return env, nil
		}
	}

	// Environment doesn't exist, need to create it
	fmt.Printf("\nDeployment environment '%s' not found in Bitbucket.\n", envName)

	if len(environments) > 0 {
		fmt.Println("\nAvailable environments:")
		for _, env := range environments {
			fmt.Printf("  - %s (%s)\n", env.Name, env.Type)
		}
		fmt.Println()
	}

	// Determine the environment type to use
	var envType string
	if envTypeOverride != "" {
		// Use the override type if provided
		envType = envTypeOverride
		if err := ValidateEnvironmentType(envType); err != nil {
			return nil, fmt.Errorf("invalid environment type override: %w", err)
		}
	} else {
		// Infer the type from the name
		envType = DetermineEnvironmentType(envName)
	}

	var shouldCreate bool

	if autoCreate {
		// Auto-create mode: no prompts
		fmt.Printf("Auto-creating environment '%s' (type: %s)...\n", envName, envType)
		shouldCreate = true
	} else {
		// Interactive mode: prompt the user
		var promptErr error
		shouldCreate, envType, promptErr = PromptCreateEnvironment(envName, envType)
		if promptErr != nil {
			return nil, promptErr
		}

		if !shouldCreate {
			return nil, fmt.Errorf("environment creation canceled by user")
		}
	}

	// Create the environment
	greenColor := color.New(color.FgGreen).SprintFunc()
	redColor := color.New(color.FgRed).SprintFunc()

	fmt.Printf("\nCreating deployment environment '%s' (type: %s)...\n", envName, envType)

	newEnv, err := client.CreateDeploymentEnvironment(repoSlug, envName, envType)
	if err != nil {
		// Check if it's a permission error
		if strings.Contains(err.Error(), "403") || strings.Contains(err.Error(), "Forbidden") {
			fmt.Printf("%s Failed to create environment\n\n", redColor("✗"))
			return nil, fmt.Errorf("permission denied: You don't have permission to create deployment environments.\n"+
				"Please ask your Bitbucket workspace admin to:\n"+
				"  1. Grant you 'Deployments: Write' permission, or\n"+
				"  2. Create the '%s' environment manually in Bitbucket", envName)
		}

		// Check if it already exists (race condition)
		if strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "409") {
			fmt.Printf("Environment already exists (created by another process). Continuing...\n")
			// Refetch environments to get the newly created one
			environments, refetchErr := client.GetDeploymentEnvironments(repoSlug)
			if refetchErr != nil {
				return nil, fmt.Errorf("environment exists but failed to refetch: %w", refetchErr)
			}
			for _, env := range environments {
				if strings.EqualFold(env.Name, envName) {
					return env, nil
				}
			}
		}

		fmt.Printf("%s Failed to create environment\n", redColor("✗"))
		return nil, fmt.Errorf("failed to create deployment environment: %w", err)
	}

	fmt.Printf("%s Environment created successfully! (UUID: %s)\n", greenColor("✓"), newEnv.UUID)

	return newEnv, nil
}
