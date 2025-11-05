package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"bitbucket.org/cover42/eiscli/internal/bitbucket"
	"bitbucket.org/cover42/eiscli/internal/config"
	"github.com/fatih/color"
	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
	"github.com/spf13/cobra"
)

const (
	baseRepoURL    = "https://bitbucket.org/cover42/baseapiv2.git"
	baseRepoRef    = "master"
	defaultProject = "Emil v2"
)

var svcNewCmd = &cobra.Command{
	Use:   "new [service-name]",
	Short: "Create a new EIS service",
	Long: `Create a new service in the EIS platform with all necessary scaffolding.
This command will clone the baseapiv2 template, initialize a new git repository,
and optionally create a Bitbucket repository with proper permissions.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		serviceName := ""
		if len(args) > 0 {
			serviceName = args[0]
		}

		if serviceName == "" {
			fmt.Println("Please provide a service name")
			fmt.Println("Usage: eiscli svc new <service-name>")
			os.Exit(1)
		}

		if err := createNewService(serviceName); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	svcCmd.AddCommand(svcNewCmd)
}

func createNewService(serviceName string) error {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	destDir := filepath.Join(cwd, serviceName)

	// Check if destination directory already exists
	if _, err := os.Stat(destDir); err == nil {
		return fmt.Errorf("directory already exists: %s", destDir)
	}

	fmt.Printf("Creating new service: %s\n", color.CyanString(serviceName))
	fmt.Printf("Cloning template repository...\n")

	// Clone the repository
	if err := cloneRepository(destDir); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	fmt.Printf("✓ Cloned template repository\n")

	// Remove .git directory and reinitialize
	fmt.Printf("Initializing new git repository...\n")
	if err := reinitializeGitRepo(destDir); err != nil {
		return fmt.Errorf("failed to reinitialize git repository: %w", err)
	}

	fmt.Printf("✓ Initialized new git repository\n")

	// Ask user if they want to create a Bitbucket repository
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("\nCreate Bitbucket repository? [y/N]: ")
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response == "y" || response == "yes" {
		if err := createBitbucketRepository(serviceName, destDir); err != nil {
			return fmt.Errorf("failed to create Bitbucket repository: %w", err)
		}
	} else {
		fmt.Printf("Skipping Bitbucket repository creation.\n")
		fmt.Printf("You can create it later and add it as a remote:\n")
		fmt.Printf("  cd %s\n", serviceName)
		fmt.Printf("  git remote add origin <repository-url>\n")
	}

	fmt.Printf("\n✓ Service '%s' created successfully!\n", color.GreenString(serviceName))
	fmt.Printf("  Directory: %s\n", destDir)

	return nil
}

func cloneRepository(destDir string) error {
	// Clone the repository using git command
	// We need to clone from master branch specifically
	cmd := exec.Command("git", "clone", "-b", baseRepoRef, baseRepoURL, destDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	return nil
}

func reinitializeGitRepo(repoDir string) error {
	// Remove .git directory
	gitDir := filepath.Join(repoDir, ".git")
	if err := os.RemoveAll(gitDir); err != nil {
		return fmt.Errorf("failed to remove .git directory: %w", err)
	}

	// Initialize new git repository with master as the initial branch
	// Use -b to specify the initial branch name explicitly
	cmd := exec.Command("git", "init", "-b", "master")
	cmd.Dir = repoDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git init failed: %w", err)
	}

	return nil
}

func createBitbucketRepository(serviceName, repoDir string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create Bitbucket client
	client, err := bitbucket.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create Bitbucket client: %w", err)
	}

	// Select project interactively
	selectedProject, err := selectProject(client, defaultProject)
	if err != nil {
		return fmt.Errorf("failed to select project: %w", err)
	}

	var projectKey string
	if selectedProject != nil {
		projectKey = selectedProject.Key
		fmt.Printf("\nSelected project: %s (key: %s)\n", color.CyanString(selectedProject.Name), color.CyanString(projectKey))
	} else {
		fmt.Printf("\nNo project selected. Repository will be created without project assignment.\n")
	}

	fmt.Printf("\nCreating Bitbucket repository...\n")

	// Create the repository
	repo, err := client.CreateRepository(serviceName, projectKey, true)
	if err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}

	fmt.Printf("✓ Created repository: %s\n", repo.FullName)

	// Add remote origin first (needed for pushing)
	fmt.Printf("Adding remote origin...\n")
	remoteURL := fmt.Sprintf("https://bitbucket.org/%s/%s.git", cfg.Bitbucket.Workspace, serviceName)
	cmd := exec.Command("git", "remote", "add", "origin", remoteURL)
	cmd.Dir = repoDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("Warning: Failed to add remote origin: %v\n", err)
	} else {
		fmt.Printf("✓ Added remote origin: %s\n", remoteURL)
	}

	// Create initial commit and push to create master branch
	fmt.Printf("Creating initial commit...\n")
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = repoDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Warning: Failed to stage files: %v\n", err)
	} else {
		cmd = exec.Command("git", "commit", "-m", "Initial commit")
		cmd.Dir = repoDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("Warning: Failed to create initial commit: %v\n", err)
		} else {
			fmt.Printf("✓ Created initial commit\n")

			// Push to master branch
			fmt.Printf("Pushing to master branch...\n")
			cmd = exec.Command("git", "push", "-u", "origin", "master")
			cmd.Dir = repoDir
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				fmt.Printf("Warning: Failed to push to master: %v\n", err)
				fmt.Printf("  You may need to push manually: git push -u origin master\n")
			} else {
				fmt.Printf("✓ Pushed to master branch\n")
			}
		}
	}

	// Set permissions for development and production groups
	// Note: Permissions API only works with apppassword/session/api_token, not OAuth
	if client.IsUsingOAuth() {
		fmt.Printf("\nNote: Repository permissions cannot be set via API when using OAuth authentication.\n")
		fmt.Printf("      Please set permissions manually in Bitbucket:\n")
		fmt.Printf("      - DevelopmentMergeAndWriteAccess: write\n")
		fmt.Printf("      - ProductionMergeAndWriteAccess: write\n")
	} else {
		fmt.Printf("\nSetting repository permissions...\n")

		// Set DevelopmentMergeAndWriteAccess group permissions
		if err := client.SetRepositoryPermissions(serviceName, "DevelopmentMergeAndWriteAccess", "write"); err != nil {
			fmt.Printf("Warning: Failed to set permissions for DevelopmentMergeAndWriteAccess: %v\n", err)
		} else {
			fmt.Printf("✓ Set permissions for DevelopmentMergeAndWriteAccess\n")
		}

		// Set ProductionMergeAndWriteAccess group permissions
		if err := client.SetRepositoryPermissions(serviceName, "ProductionMergeAndWriteAccess", "write"); err != nil {
			fmt.Printf("Warning: Failed to set permissions for ProductionMergeAndWriteAccess: %v\n", err)
		} else {
			fmt.Printf("✓ Set permissions for ProductionMergeAndWriteAccess\n")
		}
	}

	// Set default branch to master (only after branch exists)
	fmt.Printf("Setting default branch to master...\n")
	if err := client.SetRepositoryDefaultBranch(serviceName, "master"); err != nil {
		fmt.Printf("Warning: Failed to set default branch: %v\n", err)
		fmt.Printf("  You may need to set it manually in Bitbucket repository settings.\n")
	} else {
		fmt.Printf("✓ Set default branch to master\n")
	}

	return nil
}

// selectProject allows the user to interactively select a project from the workspace
// Returns the selected project or nil if user chooses to skip project assignment
func selectProject(client *bitbucket.Client, defaultProjectName string) (*bitbucket.Project, error) {
	fmt.Printf("\nFetching projects from workspace...\n")

	projects, err := client.ListProjects()
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	if len(projects) == 0 {
		fmt.Printf("No projects found in workspace. Repository will be created without project assignment.\n")
		return nil, nil
	}

	// Configure fuzzyfinder with custom display and preview
	// Build options
	opts := []fuzzyfinder.Option{
		fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
			// Preview shows project details
			if i == -1 {
				return ""
			}
			p := projects[i]
			preview := fmt.Sprintf("Project: %s\nKey: %s\n", p.Name, p.Key)
			if p.Description != "" {
				preview += fmt.Sprintf("Description: %s\n", p.Description)
			}
			if p.UUID != "" {
				preview += fmt.Sprintf("UUID: %s\n", p.UUID)
			}
			return preview
		}),
		fuzzyfinder.WithPromptString("Select a project (type to search, ↑↓ to navigate, Enter to select, ESC to skip)> "),
	}

	// If default project exists, add it as initial query to help user find it
	if defaultProjectName != "" {
		// Don't auto-fill, but show hint in prompt
		fmt.Printf("Hint: Default project is '%s' - just start typing to search\n", color.YellowString(defaultProjectName))
	}

	idx, err := fuzzyfinder.Find(
		projects,
		func(i int) string {
			// Display format: "Project Name (key: PROJECT_KEY)"
			return fmt.Sprintf("%s (key: %s)", projects[i].Name, projects[i].Key)
		},
		opts...,
	)

	if err != nil {
		// Check if user cancelled (Ctrl+C or ESC)
		if err == fuzzyfinder.ErrAbort {
			fmt.Printf("\nCancelled. Repository will be created without project assignment.\n")
			return nil, nil
		}
		return nil, fmt.Errorf("project selection failed: %w", err)
	}

	// Extract the selected project
	selectedProject := projects[idx]
	fmt.Printf("\nSelected: %s (key: %s)\n", color.CyanString(selectedProject.Name), color.CyanString(selectedProject.Key))

	return selectedProject, nil
}
