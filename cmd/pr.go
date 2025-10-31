package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"bitbucket.org/cover42/eiscli/internal/bitbucket"
	"bitbucket.org/cover42/eiscli/internal/browser"
	"bitbucket.org/cover42/eiscli/internal/config"
	"bitbucket.org/cover42/eiscli/internal/git"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var (
	prTitle       string
	prDescription string
	prBaseBranch  string
	prOpenBrowser bool
	prListState   string
	prListLimit   int
	prListAuthor  string
)

var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "Manage pull requests",
	Long: `Manage Bitbucket pull requests.

This command provides subcommands to create and list pull requests,
similar to GitHub CLI's 'gh pr' functionality.`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var prCreateCmd = &cobra.Command{
	Use:   "create [service-name]",
	Short: "Create a pull request",
	Long: `Create a pull request from the current branch to the default branch.

If service-name is not provided, it will be auto-detected from the git repository
in the current directory (based on the git remote URL).

By default, prompts for title and description. Use --title and --body flags
to provide them directly.`,
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
				fmt.Println("  2. Provide a service name: eiscli pr create <service-name>")
				return
			}
			serviceName = detectedSlug
			fmt.Printf("Auto-detected service from git repository: %s\n", serviceName)
		}

		// Get current branch
		currentBranch, err := git.GetCurrentBranch()
		if err != nil {
			fmt.Printf("Error: Failed to get current branch: %v\n", err)
			return
		}
		fmt.Printf("Current branch: %s\n", currentBranch)

		// Load configuration
		cfg, err := config.Load()
		if err != nil {
			fmt.Printf("Error loading configuration: %v\n", err)
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

		// Get default branch
		defaultBranch := prBaseBranch
		if defaultBranch == "" {
			defaultBranch, err = client.GetDefaultBranch(serviceName)
			if err != nil {
				fmt.Printf("Warning: Failed to get default branch from API: %v\n", err)
				fmt.Println("Defaulting to 'main'. Use --base to specify a different branch.")
				defaultBranch = "main"
			}
		}
		fmt.Printf("Base branch: %s\n", defaultBranch)

		// Check if we're already on the base branch
		if currentBranch == defaultBranch {
			fmt.Printf("Error: Current branch '%s' is the same as base branch '%s'\n", currentBranch, defaultBranch)
			fmt.Println("Please switch to a different branch before creating a pull request.")
			return
		}

		// Get title and description
		title := prTitle
		description := prDescription

		if title == "" {
			title, err = promptForTitle()
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}
		}

		if description == "" {
			description, err = promptForDescription()
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}
		}

		// Create the pull request
		fmt.Printf("\nCreating pull request from '%s' to '%s'...\n", currentBranch, defaultBranch)
		pr, err := client.CreatePullRequest(serviceName, currentBranch, defaultBranch, title, description)
		if err != nil {
			fmt.Printf("Error: Failed to create pull request: %v\n", err)
			return
		}

		// Display success message
		greenColor := color.New(color.FgGreen).SprintFunc()
		fmt.Printf("\n%s Pull request created successfully!\n", greenColor("✓"))
		fmt.Printf("PR #%d: %s\n", pr.ID, pr.Title)
		fmt.Printf("URL: %s\n", pr.WebURL)

		// Open in browser if requested
		if prOpenBrowser {
			fmt.Println("\nOpening pull request in browser...")
			err := browser.Open(pr.WebURL)
			if err != nil {
				fmt.Printf("Warning: Failed to open browser: %v\n", err)
				fmt.Println("Please copy and paste the URL above into your browser.")
			}
		}
	},
}

var prListCmd = &cobra.Command{
	Use:   "list [service-name]",
	Short: "List pull requests",
	Long: `List pull requests for a repository.

If service-name is not provided, it will be auto-detected from the git repository
in the current directory (based on the git remote URL).

By default, shows open pull requests. Use --state to filter by state.
Use --author to filter by PR author. The special value "@me" can be
used to filter PRs created by the current user (matched by Bitbucket UUID/username or git email).

Examples:
  eiscli pr list --author "@me"
  eiscli pr list --author "username"
  eiscli pr list --state OPEN --author "@me"`,
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
				fmt.Println("  2. Provide a service name: eiscli pr list <service-name>")
				return
			}
			serviceName = detectedSlug
			fmt.Printf("Auto-detected service from git repository: %s\n", serviceName)
		}

		// Load configuration
		cfg, err := config.Load()
		if err != nil {
			fmt.Printf("Error loading configuration: %v\n", err)
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

		// Determine state filter
		state := prListState
		if state == "" {
			state = "OPEN" // Default to open PRs
		}

		// Build options
		opts := &bitbucket.PullRequestOptions{
			State:  state,
			Limit:  prListLimit,
			Author: prListAuthor,
		}

		// Fetch pull requests
		filterMsg := fmt.Sprintf("state: %s", state)
		if prListAuthor != "" {
			filterMsg += fmt.Sprintf(", author: %s", prListAuthor)
		}
		fmt.Printf("Fetching pull requests (%s)...\n", filterMsg)
		prs, err := client.ListPullRequests(serviceName, opts)
		if err != nil {
			fmt.Printf("Error: Failed to fetch pull requests: %v\n", err)
			return
		}

		if len(prs) == 0 {
			fmt.Println("\nNo pull requests found.")
			return
		}

		// Display pull requests in a table
		fmt.Println()
		displayPullRequestsTable(prs)
	},
}

func promptForTitle() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("PR Title: ")
	title, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read title: %w", err)
	}
	title = strings.TrimSpace(title)

	if title == "" {
		return "", fmt.Errorf("title cannot be empty")
	}

	return title, nil
}

func promptForDescription() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("PR Description (press Enter twice to finish, or leave empty):")
	description := ""
	emptyLineCount := 0

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read description: %w", err)
		}

		line = strings.TrimRight(line, "\n\r")
		if line == "" {
			emptyLineCount++
			if emptyLineCount >= 2 {
				break
			}
			description += "\n"
		} else {
			emptyLineCount = 0
			if description != "" {
				description += "\n"
			}
			description += line
		}
	}

	return strings.TrimSpace(description), nil
}

// formatClickablePRID formats a PR ID as a clickable terminal hyperlink with styling
func formatClickablePRID(prID int, prURL string) string {
	if prURL == "" {
		// Fallback to plain format if URL is not available
		return fmt.Sprintf("#%d", prID)
	}

	// OSC 8 escape sequence for terminal hyperlinks
	// Format: \033]8;;{URL}\033\\{text}\033]8;;\033\\
	hyperlinkStart := fmt.Sprintf("\033]8;;%s\033\\", prURL)
	hyperlinkEnd := "\033]8;;\033\\"

	// Style the PR ID with cyan color and underline
	// Format: #123
	prIDText := fmt.Sprintf("#%d", prID)
	styledID := color.New(color.FgCyan, color.Underline).Sprint(prIDText)

	// Combine hyperlink escape sequences with styled text
	return fmt.Sprintf("%s%s%s", hyperlinkStart, styledID, hyperlinkEnd)
}

func displayPullRequestsTable(prs []*bitbucket.PullRequest) {
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("ID", "Title", "Author", "Source → Dest", "State", "Updated")

	for _, pr := range prs {
		// Truncate title if too long
		title := pr.Title
		if len(title) > 50 {
			title = title[:47] + "..."
		}

		// Format branch info
		branchInfo := fmt.Sprintf("%s → %s", pr.SourceBranch, pr.DestinationBranch)

		// Format updated time
		updatedTime := formatTimeAgo(pr.UpdatedOn)

		// Color code state
		state := pr.State
		switch strings.ToUpper(pr.State) {
		case "OPEN":
			state = color.New(color.FgGreen).Sprint(state)
		case "MERGED":
			state = color.New(color.FgBlue).Sprint(state)
		case "DECLINED":
			state = color.New(color.FgRed).Sprint(state)
		}

		// Format PR ID as clickable hyperlink
		prID := formatClickablePRID(pr.ID, pr.WebURL)

		table.Append(prID, title, pr.Author, branchInfo, state, updatedTime)
	}

	table.Render()
	fmt.Printf("\nTotal: %d pull request(s)\n", len(prs))
}

func formatTimeAgo(t time.Time) string {
	if t.IsZero() {
		return "N/A"
	}

	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		minutes := int(diff.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 30*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		return t.Format("2006-01-02")
	}
}

func init() {
	rootCmd.AddCommand(prCmd)
	prCmd.AddCommand(prCreateCmd)
	prCmd.AddCommand(prListCmd)

	// pr create flags
	prCreateCmd.Flags().StringVarP(&prTitle, "title", "t", "", "Pull request title")
	prCreateCmd.Flags().StringVarP(&prDescription, "body", "b", "", "Pull request description")
	prCreateCmd.Flags().StringVarP(&prBaseBranch, "base", "", "", "Base branch (default: repository default branch)")
	prCreateCmd.Flags().BoolVarP(&prOpenBrowser, "web", "w", false, "Open pull request in browser after creation")

	// pr list flags
	prListCmd.Flags().StringVarP(&prListState, "state", "s", "", "Filter by state (OPEN, MERGED, DECLINED, SUPERSEDED). Default: OPEN")
	prListCmd.Flags().IntVarP(&prListLimit, "limit", "l", 25, "Number of pull requests to display")
	prListCmd.Flags().StringVarP(&prListAuthor, "author", "a", "", "Filter by PR author. Use '@me' to filter PRs created by you")
}
