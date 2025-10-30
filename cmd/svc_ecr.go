package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"bitbucket.org/cover42/eiscli/internal/aws"
	"bitbucket.org/cover42/eiscli/internal/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/spf13/cobra"
)

var (
	ecrRegion     string
	ecrCreate     bool
	ecrAllRegions bool
)

var svcECRCmd = &cobra.Command{
	Use:   "ecr [service-name]",
	Short: "Manage ECR registry for a service",
	Long: `Check if ECR registry exists for a service and optionally create it.

The command uses AWS_PROFILE environment variable or the default AWS profile from config.
Repositories can be managed in different AWS regions (eu-central-1 or eu-central-2).

If service-name is not provided, it will be auto-detected from the git repository.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		// Load configuration
		cfg, err := config.Load()
		if err != nil {
			fmt.Printf("Error loading configuration: %v\n", err)
			return
		}

		// Get service name
		serviceName := getServiceName(args)
		if serviceName == "" {
			return
		}

		if ecrAllRegions {
			checkAllRegions(ctx, cfg, serviceName)
		} else {
			checkSingleRegion(ctx, cfg, serviceName, ecrRegion)
		}
	},
}

func checkSingleRegion(ctx context.Context, cfg *config.Config, serviceName, region string) {
	// Use AWS_PROFILE environment variable or default profile from config
	profile := os.Getenv("AWS_PROFILE")
	if profile == "" {
		profile = cfg.AWS.DefaultProfile
	}

	fmt.Printf("🔍 Checking ECR registry for service: %s\n", serviceName)
	fmt.Printf("   Region: %s\n", region)
	fmt.Printf("   AWS Profile: %s\n\n", profile)

	// Create ECR client
	ecrClient, err := aws.NewECRClient(ctx, profile, region)
	if err != nil {
		fmt.Printf("❌ Error creating ECR client: %v\n", err)
		fmt.Println("\nMake sure:")
		fmt.Printf("  1. AWS profile '%s' is configured in ~/.aws/config\n", profile)
		fmt.Println("  2. You have valid AWS credentials")
		fmt.Println("  3. You have ECR permissions")
		return
	}

	// Check if repository exists
	exists, repo, err := ecrClient.RepositoryExists(ctx, serviceName)
	if err != nil {
		fmt.Printf("❌ Error checking repository: %v\n", err)
		return
	}

	if exists {
		displayRepositoryInfo(ecrClient, repo)
	} else {
		handleRepositoryNotFound(ctx, ecrClient, cfg, serviceName, region)
	}
}

func checkAllRegions(ctx context.Context, cfg *config.Config, serviceName string) {
	fmt.Printf("🔍 Checking ECR registries for service: %s\n\n", serviceName)

	// Use AWS_PROFILE environment variable or default profile from config
	profile := os.Getenv("AWS_PROFILE")
	if profile == "" {
		profile = cfg.AWS.DefaultProfile
	}

	regions := []struct {
		name   string
		region string
	}{
		{"Frankfurt", "eu-central-1"},
		{"Zurich", "eu-central-2"},
	}

	for i, reg := range regions {
		if i > 0 {
			fmt.Println()
		}

		fmt.Printf("═══ %s (%s) ═══\n", reg.name, reg.region)
		fmt.Printf("AWS Profile: %s\n", profile)

		ecrClient, err := aws.NewECRClient(ctx, profile, reg.region)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			continue
		}

		exists, repo, err := ecrClient.RepositoryExists(ctx, serviceName)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			continue
		}

		if exists {
			fmt.Printf("✅ Repository exists\n")
			fmt.Printf("   URI: %s\n", *repo.RepositoryUri)
			fmt.Printf("   Created: %v\n", repo.CreatedAt.Format("2006-01-02 15:04:05"))
			consoleURL := ecrClient.GetConsoleURL(serviceName)
			fmt.Printf("   Console: %s\n", consoleURL)
		} else {
			fmt.Printf("⚠️  Repository not found\n")
			fmt.Printf("   To create: eiscli svc ecr %s --region %s --create\n", serviceName, reg.region)
		}
	}
}

func displayRepositoryInfo(client *aws.ECRClient, repo *types.Repository) {
	fmt.Printf("✅ ECR repository exists!\n\n")
	fmt.Printf("Repository Details:\n")
	fmt.Printf("  Name:    %s\n", *repo.RepositoryName)
	fmt.Printf("  URI:     %s\n", *repo.RepositoryUri)
	fmt.Printf("  ARN:     %s\n", *repo.RepositoryArn)
	fmt.Printf("  Created: %v\n", repo.CreatedAt.Format("2006-01-02 15:04:05"))

	if repo.ImageScanningConfiguration != nil {
		fmt.Printf("  Scan on push: %v\n", repo.ImageScanningConfiguration.ScanOnPush)
	}
	fmt.Printf("  Tag mutability: %s\n", repo.ImageTagMutability)

	consoleURL := client.GetConsoleURL(*repo.RepositoryName)
	fmt.Printf("\n🔗 View in AWS Console:\n   %s\n", consoleURL)
}

func handleRepositoryNotFound(ctx context.Context, client *aws.ECRClient, cfg *config.Config, serviceName, region string) {
	fmt.Printf("⚠️  ECR repository does not exist.\n\n")

	if ecrCreate {
		createRepository(ctx, client, serviceName)
	} else {
		fmt.Printf("To create the repository, run:\n")
		fmt.Printf("  eiscli svc ecr %s --region %s --create\n", serviceName, region)
	}
}

func createRepository(ctx context.Context, client *aws.ECRClient, defaultName string) {
	// Prompt for repository name
	fmt.Printf("Enter ECR repository name [%s]: ", defaultName)
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("❌ Error reading input: %v\n", err)
		return
	}

	repoName := strings.TrimSpace(input)
	if repoName == "" {
		repoName = defaultName
	}

	fmt.Printf("\n🚀 Creating ECR repository '%s'...\n", repoName)

	repo, err := client.CreateRepository(ctx, repoName)
	if err != nil {
		fmt.Printf("❌ Error creating repository: %v\n", err)
		return
	}

	fmt.Printf("✅ Repository created successfully!\n\n")
	displayRepositoryInfo(client, repo)
}

func init() {
	svcCmd.AddCommand(svcECRCmd)

	svcECRCmd.Flags().StringVarP(&ecrRegion, "region", "r", "eu-central-1",
		"AWS region (eu-central-1 or eu-central-2)")
	svcECRCmd.Flags().BoolVarP(&ecrCreate, "create", "c", false,
		"Create the repository if it doesn't exist (prompts for name)")
	svcECRCmd.Flags().BoolVarP(&ecrAllRegions, "all-regions", "a", false,
		"Check all regions (both eu-central-1 and eu-central-2)")
}
