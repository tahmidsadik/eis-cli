package cmd

import (
	"fmt"
	"strings"
	"time"

	"bitbucket.org/cover42/eiscli/internal/bitbucket"
	"bitbucket.org/cover42/eiscli/internal/config"
	"bitbucket.org/cover42/eiscli/internal/git"
	"github.com/spf13/cobra"
)

var (
	pipelineLimit    int
	pipelineShowLog  bool
	pipelineLogLines int
)

var svcPipelineCmd = &cobra.Command{
	Use:   "pipeline [service-name]",
	Short: "List pipeline build status from Bitbucket",
	Long: `Display the status of the last pipeline builds for a service.
Shows build status, duration, commit information, clickable Bitbucket links, and more.

If service-name is not provided, it will be auto-detected from the git repository
in the current directory (based on the git remote URL).

Use the --logs flag to see detailed pipeline steps and their status.`,
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
				fmt.Println("  2. Provide a service name: eiscli svc pipeline <service-name>")
				return
			}
			serviceName = detectedSlug
			fmt.Printf("Auto-detected service from git repository: %s\n", serviceName)
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

		fmt.Printf("Fetching pipeline builds for service: %s (last %d builds)\n", serviceName, pipelineLimit)

		// Fetch pipelines (with or without steps/logs)
		var pipelines []*bitbucket.Pipeline
		if pipelineShowLog {
			fmt.Println("Fetching pipeline steps and logs...")
			pipelines, err = client.ListPipelinesWithSteps(&bitbucket.PipelinesOptions{
				RepoSlug: serviceName,
				Limit:    pipelineLimit,
			}, pipelineLogLines)
		} else {
			pipelines, err = client.ListPipelines(&bitbucket.PipelinesOptions{
				RepoSlug: serviceName,
				Limit:    pipelineLimit,
			})
		}

		if err != nil {
			fmt.Printf("\nError fetching pipelines: %v\n", err)
			return
		}

		if len(pipelines) == 0 {
			fmt.Println("\nNo pipelines found for this repository.")
			return
		}

		// Display pipelines
		fmt.Println("\n" + strings.Repeat("=", 100))
		for i, p := range pipelines {
			displayPipeline(p, i+1)
			if i < len(pipelines)-1 {
				fmt.Println(strings.Repeat("-", 100))
			}
		}
		fmt.Println(strings.Repeat("=", 100))
	},
}

func displayPipeline(p *bitbucket.Pipeline, index int) {
	// Status icon
	statusIcon := getStatusIcon(p.State.Name)
	resultStatus := "N/A"
	if p.State.Result != nil {
		resultStatus = p.State.Result.Name
	}

	// Format times
	createdTime := p.CreatedOn.Format("2006-01-02 15:04:05")
	duration := formatDuration(p.DurationSec)

	// Truncate commit message if too long
	commitMsg := "N/A"
	commitHash := "N/A"
	if p.Target.Commit != nil {
		commitHash = p.Target.Commit.Hash[:7]                       // Short hash
		commitMsg = strings.Split(p.Target.Commit.Message, "\n")[0] // First line only
		if len(commitMsg) > 60 {
			commitMsg = commitMsg[:57] + "..."
		}
	}

	// Display pipeline info
	fmt.Printf("#%d  Build #%d  %s %s\n", index, p.BuildNumber, statusIcon, p.State.Name)
	fmt.Printf("    Result:     %s\n", resultStatus)
	fmt.Printf("    Target:     %s: %s\n", p.Target.RefType, p.Target.RefName)
	fmt.Printf("    Commit:     %s - %s\n", commitHash, commitMsg)
	fmt.Printf("    Trigger:    %s\n", p.Trigger.Type)
	fmt.Printf("    Creator:    %s\n", p.Creator)
	fmt.Printf("    Created:    %s\n", createdTime)
	fmt.Printf("    Duration:   %s\n", duration)
	if p.BuildSecsUsed > 0 {
		fmt.Printf("    Build Time: %d seconds\n", p.BuildSecsUsed)
	}

	// Display web URL
	if p.WebURL != "" {
		fmt.Printf("    Link:       %s\n", p.WebURL)
	}

	// Display steps if available
	if len(p.Steps) > 0 {
		fmt.Printf("\n    Steps:\n")
		for i, step := range p.Steps {
			stepIcon := getStatusIcon(step.State)
			stepStatus := step.State
			if step.Result != "" {
				stepStatus = step.Result
			}
			fmt.Printf("      %d. %s %s - %s\n", i+1, stepIcon, step.Name, stepStatus)

			// Display log snippet if available
			if step.LogSnippet != "" {
				fmt.Printf("         Last log lines:\n")
				lines := strings.Split(step.LogSnippet, "\n")
				for _, line := range lines {
					if strings.TrimSpace(line) != "" {
						fmt.Printf("         │ %s\n", line)
					}
				}
			}
		}
	}
}

func getStatusIcon(status string) string {
	switch strings.ToUpper(status) {
	case "COMPLETED":
		return "✓"
	case "SUCCESSFUL":
		return "✓"
	case "FAILED":
		return "✗"
	case "ERROR":
		return "✗"
	case "STOPPED":
		return "■"
	case "IN_PROGRESS":
		return "●"
	case "PENDING":
		return "○"
	case "PAUSED":
		return "⏸"
	default:
		return "?"
	}
}

func formatDuration(seconds int) string {
	if seconds == 0 {
		return "N/A"
	}

	duration := time.Duration(seconds) * time.Second
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	secs := int(duration.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, secs)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, secs)
	}
	return fmt.Sprintf("%ds", secs)
}

func init() {
	svcCmd.AddCommand(svcPipelineCmd)
	svcPipelineCmd.Flags().IntVarP(&pipelineLimit, "limit", "l", 5, "Number of pipeline builds to display")
	svcPipelineCmd.Flags().BoolVarP(&pipelineShowLog, "logs", "s", false, "Show pipeline steps and log snippets")
	svcPipelineCmd.Flags().IntVar(&pipelineLogLines, "log-lines", 10, "Number of log lines to display per step")
}
