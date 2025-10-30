package cmd

import (
	"context"
	"fmt"
	"strings"

	"bitbucket.org/cover42/eiscli/internal/aws"
	"bitbucket.org/cover42/eiscli/internal/bitbucket"
	"bitbucket.org/cover42/eiscli/internal/config"
	"bitbucket.org/cover42/eiscli/internal/git"
	"github.com/spf13/cobra"
)

var (
	updateWorkspaceVarRegion     string
	updateWorkspaceVarAllRegions bool
	updateWorkspaceVarEnv        string
)

var svcECRUpdateWorkspaceVarCmd = &cobra.Command{
	Use:   "update-workspace-var [service-name]",
	Short: "Create/update workspace variables for ECR image URIs",
	Long: `Create or update Bitbucket workspace-level variables for ECR registry URIs.

This command will:
  1. Verify the ECR repository exists in the AWS account
  2. Get the ECR repository URI
  3. Create or update the workspace variable {SERVICENAME}_IMAGE_URI

Variable naming:
  - Service "documentgenerator" in eu-central-1 → DOCUMENTGENERATOR_IMAGE_URI
  - Service "documentgenerator" in eu-central-2 → ZURICH_DOCUMENTGENERATOR_IMAGE_URI

The --env flag is used only to determine which AWS profile to use for authentication.
The command will fail if the ECR repository doesn't exist. Use 'eiscli svc ecr --create' to create it first.

If service-name is not provided, it will be auto-detected from the git repository.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		// Load configuration
		cfg, err := config.Load()
		if err != nil {
			fmt.Printf("❌ Error loading configuration: %v\n", err)
			return
		}

		// Get service name
		serviceName := getServiceNameForWorkspaceVar(args)
		if serviceName == "" {
			return
		}

		if updateWorkspaceVarAllRegions {
			// Update both regions
			if err := updateWorkspaceVariableForRegion(ctx, cfg, serviceName, "eu-central-1"); err != nil {
				fmt.Printf("❌ Failed for eu-central-1: %v\n", err)
				return
			}

			fmt.Println()

			if err := updateWorkspaceVariableForRegion(ctx, cfg, serviceName, "eu-central-2"); err != nil {
				fmt.Printf("❌ Failed for eu-central-2: %v\n", err)
				return
			}
		} else {
			// Update single region
			if err := updateWorkspaceVariableForRegion(ctx, cfg, serviceName, updateWorkspaceVarRegion); err != nil {
				fmt.Printf("❌ Failed: %v\n", err)
				return
			}
		}

		fmt.Println("\n✅ Workspace variable(s) updated successfully!")
	},
}

func getServiceNameForWorkspaceVar(args []string) string {
	if len(args) > 0 {
		return args[0]
	}

	// Auto-detect from git repository
	detectedSlug, err := git.DetectRepositorySlug()
	if err != nil {
		fmt.Println("❌ Error: No service name provided and could not auto-detect from git repository")
		fmt.Printf("  %v\n", err)
		fmt.Println("\nUsage:")
		fmt.Println("  1. Run this command from within a git repository, or")
		fmt.Println("  2. Provide a service name: eiscli svc ecr update-workspace-var <service-name>")
		return ""
	}

	fmt.Printf("🔍 Auto-detected service from git repository: %s\n\n", detectedSlug)
	return detectedSlug
}

func updateWorkspaceVariableForRegion(ctx context.Context, cfg *config.Config, serviceName, region string) error {
	// Determine AWS profile based on environment
	profile := aws.GetProfileForEnvironment(updateWorkspaceVarEnv, cfg)
	profileDesc := aws.GetProfileDescription(updateWorkspaceVarEnv, cfg)

	isZurich := region == "eu-central-2"
	regionName := "Frankfurt"
	if isZurich {
		regionName = "Zurich"
	}

	fmt.Printf("═══ %s Region (%s) ═══\n", regionName, region)
	fmt.Printf("Service: %s\n", serviceName)
	fmt.Printf("AWS Profile: %s\n", profileDesc)
	fmt.Printf("Environment: %s\n\n", updateWorkspaceVarEnv)

	// Step 1: Check if ECR repository exists
	ecrClient, err := aws.NewECRClient(ctx, profile, region)
	if err != nil {
		return fmt.Errorf("failed to create ECR client: %w\nMake sure AWS profile '%s' is configured in ~/.aws/config", err, profile)
	}

	exists, repo, err := ecrClient.RepositoryExists(ctx, serviceName)
	if err != nil {
		return fmt.Errorf("failed to check ECR repository: %w", err)
	}

	if !exists {
		fmt.Printf("⚠️  ECR repository '%s' does not exist in %s (%s)\n\n", serviceName, regionName, region)
		fmt.Println("You must create the ECR repository first:")
		fmt.Printf("  eiscli svc ecr %s --region %s --create\n", serviceName, region)
		return fmt.Errorf("ECR repository does not exist")
	}

	// Step 2: Get ECR URI
	ecrURI := *repo.RepositoryUri
	fmt.Printf("✅ ECR repository found\n")
	fmt.Printf("   URI: %s\n\n", ecrURI)

	// Step 3: Create/update workspace variable
	variableName := generateVariableName(serviceName, isZurich)

	// Create Bitbucket client
	bbClient, err := bitbucket.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create Bitbucket client: %w", err)
	}

	fmt.Printf("🔄 Updating workspace variable: %s\n", variableName)

	// Create or update the variable (not secured as it's a public ECR URI)
	wasUpdated, oldValue, err := bbClient.CreateOrUpdateWorkspaceVariable(variableName, ecrURI, false)
	if err != nil {
		return fmt.Errorf("failed to create/update workspace variable: %w", err)
	}

	if wasUpdated {
		fmt.Printf("✅ Updated workspace variable\n")
		fmt.Printf("   Old value: %s\n", oldValue)
		fmt.Printf("   New value: %s\n", ecrURI)
	} else {
		fmt.Printf("✅ Created workspace variable\n")
		fmt.Printf("   Value: %s\n", ecrURI)
	}

	return nil
}

func generateVariableName(serviceName string, isZurich bool) string {
	// Convert service name to uppercase
	upperServiceName := strings.ToUpper(serviceName)

	if isZurich {
		return fmt.Sprintf("ZURICH_%s_IMAGE_URI", upperServiceName)
	}

	return fmt.Sprintf("%s_IMAGE_URI", upperServiceName)
}

func init() {
	svcECRCmd.AddCommand(svcECRUpdateWorkspaceVarCmd)

	svcECRUpdateWorkspaceVarCmd.Flags().StringVarP(&updateWorkspaceVarRegion, "region", "r", "eu-central-1",
		"AWS region (eu-central-1 or eu-central-2)")
	svcECRUpdateWorkspaceVarCmd.Flags().BoolVarP(&updateWorkspaceVarAllRegions, "all-regions", "a", false,
		"Update workspace variables for both eu-central-1 and eu-central-2 regions")
	svcECRUpdateWorkspaceVarCmd.Flags().StringVarP(&updateWorkspaceVarEnv, "env", "e", "testing",
		"Environment to use for AWS profile selection (testing, staging, prod, etc.)")
}
