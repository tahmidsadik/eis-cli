package cmd

import (
	"fmt"
	"os"
	"strings"

	"bitbucket.org/cover42/eiscli/internal/bitbucket"
	"bitbucket.org/cover42/eiscli/internal/config"
	"bitbucket.org/cover42/eiscli/internal/git"
	"bitbucket.org/cover42/eiscli/internal/ssh"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	sshLinkTarget string
	sshLinkForce  bool
	sshLinkDryRun bool
)

var sshLinkCmd = &cobra.Command{
	Use:   "ssh-link [service-name]",
	Short: "Link SSH key from service to protorepo for CI/CD access",
	Long: `Create and link an SSH key pair from a microservice repository to protorepo.

This command automates the process of setting up SSH access for CI/CD pipelines
to clone the shared protorepo. It performs the following steps:

1. Checks if an SSH key pair exists in the service repository's pipeline config
2. If not (or if --force is used), generates a new ED25519 SSH key pair
3. Stores the key pair in the service's Bitbucket pipeline SSH configuration
4. Adds the public key as a deploy key to the target repository (protorepo)

The deploy key label follows the format: {service-name}-pipeline

If service-name is not provided, it will be auto-detected from the git repository
in the current directory (based on the git remote URL).

Examples:
  # Auto-detect service from current git repo
  eiscli ssh-link

  # Specify service name explicitly
  eiscli ssh-link myservice

  # Preview changes without applying
  eiscli ssh-link --dry-run

  # Force regenerate SSH key pair
  eiscli ssh-link --force`,
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
				fmt.Println("  2. Provide a service name: eiscli ssh-link <service-name>")
				return
			}
			serviceName = detectedSlug
			fmt.Printf("Auto-detected service from git repository: %s\n\n", serviceName)
		}

		if err := executeSSHLink(serviceName, sshLinkTarget, sshLinkForce, sshLinkDryRun); err != nil {
			fmt.Printf("\nError: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(sshLinkCmd)
	sshLinkCmd.Flags().StringVarP(&sshLinkTarget, "target", "t", "protorepo", "Target repository for deploy key")
	sshLinkCmd.Flags().BoolVarP(&sshLinkForce, "force", "f", false, "Force regenerate SSH key pair even if one exists")
	sshLinkCmd.Flags().BoolVar(&sshLinkDryRun, "dry-run", false, "Show what would be done without making changes")
}

func executeSSHLink(serviceName, targetRepo string, force, dryRun bool) error {
	greenColor := color.New(color.FgGreen).SprintFunc()
	yellowColor := color.New(color.FgYellow).SprintFunc()
	cyanColor := color.New(color.FgCyan).SprintFunc()

	// Header
	fmt.Printf("SSH Key Link: %s -> %s\n", cyanColor(serviceName), cyanColor(targetRepo))
	fmt.Println(strings.Repeat("=", 80))

	if dryRun {
		fmt.Printf("%s\n\n", yellowColor("DRY RUN MODE - No changes will be made"))
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	// Create Bitbucket client
	client, err := bitbucket.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create Bitbucket client: %w", err)
	}

	// Step 1: Check if SSH key pair exists in service repo
	fmt.Printf("\nChecking SSH key pair in %s...\n", serviceName)

	existingKeyPair, err := client.GetPipelineSSHKeyPair(serviceName)
	if err != nil {
		return fmt.Errorf("failed to check SSH key pair: %w", err)
	}

	var publicKey string
	needsNewKey := existingKeyPair == nil || force

	switch {
	case existingKeyPair != nil && !force:
		fmt.Printf("  %s SSH key pair already exists\n", greenColor("-"))
		publicKey = existingKeyPair.PublicKey
	case existingKeyPair != nil && force:
		fmt.Printf("  %s SSH key pair exists, but --force flag is set\n", yellowColor("-"))
		needsNewKey = true
	default:
		fmt.Printf("  %s No SSH key pair found\n", yellowColor("-"))
		needsNewKey = true
	}

	// Step 2: Generate new key pair if needed
	if needsNewKey {
		fmt.Printf("\nGenerating new SSH key pair...\n")

		comment := fmt.Sprintf("%s-pipeline", serviceName)
		keyPair, err := ssh.GenerateED25519KeyPair(comment)
		if err != nil {
			return fmt.Errorf("failed to generate SSH key pair: %w", err)
		}

		publicKey = keyPair.PublicKey

		if dryRun {
			fmt.Printf("  %s Would create SSH key pair in %s\n", yellowColor("[DRY RUN]"), serviceName)
		} else {
			fmt.Printf("  Creating SSH key pair in %s...\n", serviceName)

			createdKeyPair, err := client.CreatePipelineSSHKeyPair(serviceName, keyPair.PrivateKey, keyPair.PublicKey)
			if err != nil {
				return fmt.Errorf("failed to create SSH key pair: %w", err)
			}

			publicKey = createdKeyPair.PublicKey
			fmt.Printf("  %s SSH key pair created successfully\n", greenColor("✓"))
		}
	}

	// Step 3: Check if deploy key already exists in target repo
	fmt.Printf("\nChecking deploy keys in %s...\n", targetRepo)

	deployKeyLabel := fmt.Sprintf("%s-pipeline", serviceName)

	existingDeployKey, err := client.FindDeployKeyByLabel(targetRepo, deployKeyLabel)
	if err != nil {
		return fmt.Errorf("failed to check deploy keys: %w", err)
	}

	deployKeys, err := client.ListDeployKeys(targetRepo)
	if err != nil {
		return fmt.Errorf("failed to list deploy keys: %w", err)
	}
	fmt.Printf("  - Found %d existing deploy key(s)\n", len(deployKeys))

	if existingDeployKey != nil {
		fmt.Printf("  %s Deploy key with label '%s' already exists (ID: %d)\n",
			yellowColor("-"), deployKeyLabel, existingDeployKey.ID)

		if !force {
			fmt.Printf("\n%s SSH key is already linked. Use --force to regenerate.\n", greenColor("✓"))
			return nil
		}

		fmt.Printf("  %s --force flag set, but deploy key already exists.\n", yellowColor("!"))
		fmt.Printf("  Note: Bitbucket does not allow updating deploy keys. You would need to\n")
		fmt.Printf("        manually delete the existing key from %s first.\n", targetRepo)
		return nil
	}

	fmt.Printf("  - No deploy key with label '%s' found\n", deployKeyLabel)

	// Step 4: Add deploy key to target repo
	fmt.Printf("\nAdding deploy key to %s...\n", targetRepo)

	if dryRun {
		fmt.Printf("  %s Would add deploy key with label '%s'\n", yellowColor("[DRY RUN]"), deployKeyLabel)
	} else {
		_, err := client.AddDeployKey(targetRepo, publicKey, deployKeyLabel)
		if err != nil {
			return fmt.Errorf("failed to add deploy key: %w", err)
		}

		fmt.Printf("  %s Deploy key added with label '%s'\n", greenColor("✓"), deployKeyLabel)
	}

	// Success message
	fmt.Printf("\n%s\n", strings.Repeat("=", 80))
	if dryRun {
		fmt.Printf("%s Run without --dry-run to apply changes.\n", yellowColor("DRY RUN COMPLETE."))
	} else {
		fmt.Printf("%s The service can now clone %s in CI/CD pipelines.\n",
			greenColor("Done!"), targetRepo)
		fmt.Printf("\nTo use in bitbucket-pipelines.yml, the private key is automatically\n")
		fmt.Printf("available as the default SSH identity. You may need to add %s\n", targetRepo)
		fmt.Printf("as a known host if not already configured.\n")
	}

	return nil
}
