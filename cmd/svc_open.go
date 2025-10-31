package cmd

import (
	"fmt"

	"bitbucket.org/cover42/eiscli/internal/bitbucket"
	"bitbucket.org/cover42/eiscli/internal/browser"
	"bitbucket.org/cover42/eiscli/internal/config"
	"github.com/spf13/cobra"
)

var svcOpenCmd = &cobra.Command{
	Use:   "open [service-name]",
	Short: "Open service repository in browser",
	Long: `Open the Bitbucket repository page in your default browser and print the URL.

If service-name is not provided, it will be auto-detected from the git repository
in the current directory (based on the git remote URL).

Subcommands:
  pipelines  - Open the pipelines page
  prs        - Open the pull requests page
  vars       - Open the variables page (deployment or repository)
  settings   - Open the repository settings page`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		serviceName := getServiceName(args)
		if serviceName == "" {
			return
		}

		cfg, err := config.Load()
		if err != nil {
			fmt.Printf("Error: Failed to load configuration: %v\n", err)
			return
		}

		url := bitbucket.BuildRepositoryURL(cfg.Bitbucket.Workspace, serviceName)
		openURL(url, serviceName, "repository")
	},
}

var openPipelinesCmd = &cobra.Command{
	Use:   "pipelines [service-name]",
	Short: "Open pipelines page in browser",
	Long: `Open the Bitbucket pipelines page in your default browser and print the URL.

If service-name is not provided, it will be auto-detected from the git repository
in the current directory.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		serviceName := getServiceName(args)
		if serviceName == "" {
			return
		}

		cfg, err := config.Load()
		if err != nil {
			fmt.Printf("Error: Failed to load configuration: %v\n", err)
			return
		}

		url := bitbucket.BuildPipelinesURL(cfg.Bitbucket.Workspace, serviceName)
		openURL(url, serviceName, "pipelines")
	},
}

var prsCmd = &cobra.Command{
	Use:   "prs [service-name]",
	Short: "Open pull requests page in browser",
	Long: `Open the Bitbucket pull requests page in your default browser and print the URL.

If service-name is not provided, it will be auto-detected from the git repository
in the current directory.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		serviceName := getServiceName(args)
		if serviceName == "" {
			return
		}

		cfg, err := config.Load()
		if err != nil {
			fmt.Printf("Error: Failed to load configuration: %v\n", err)
			return
		}

		url := bitbucket.BuildPullRequestsURL(cfg.Bitbucket.Workspace, serviceName)
		openURL(url, serviceName, "pull requests")
	},
}

var openVarsType string

var openVarsCmd = &cobra.Command{
	Use:   "vars [service-name]",
	Short: "Open variables page in browser",
	Long: `Open the Bitbucket variables settings page in your default browser and print the URL.

By default, opens the deployment variables page. Use --type repository to open the repository variables page.

If service-name is not provided, it will be auto-detected from the git repository
in the current directory.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		serviceName := getServiceName(args)
		if serviceName == "" {
			return
		}

		cfg, err := config.Load()
		if err != nil {
			fmt.Printf("Error: Failed to load configuration: %v\n", err)
			return
		}

		var url string
		var pageType string
		if openVarsType == "repository" {
			url = bitbucket.BuildRepositoryVariablesURL(cfg.Bitbucket.Workspace, serviceName)
			pageType = "repository variables"
		} else {
			url = bitbucket.BuildDeploymentVariablesURL(cfg.Bitbucket.Workspace, serviceName)
			pageType = "deployment variables"
		}
		openURL(url, serviceName, pageType)
	},
}

var settingsCmd = &cobra.Command{
	Use:   "settings [service-name]",
	Short: "Open repository settings page in browser",
	Long: `Open the Bitbucket repository settings page in your default browser and print the URL.

If service-name is not provided, it will be auto-detected from the git repository
in the current directory.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		serviceName := getServiceName(args)
		if serviceName == "" {
			return
		}

		cfg, err := config.Load()
		if err != nil {
			fmt.Printf("Error: Failed to load configuration: %v\n", err)
			return
		}

		url := bitbucket.BuildSettingsURL(cfg.Bitbucket.Workspace, serviceName)
		openURL(url, serviceName, "settings")
	},
}

// openURL prints the URL and opens it in the browser
func openURL(url, serviceName, pageType string) {
	fmt.Printf("Opening %s %s page...\n", serviceName, pageType)
	fmt.Printf("URL: %s\n", url)

	err := browser.Open(url)
	if err != nil {
		fmt.Printf("\nWarning: Failed to open browser: %v\n", err)
		fmt.Println("Please copy and paste the URL above into your browser.")
	} else {
		fmt.Println("✓ Opened in browser")
	}
}

func init() {
	rootCmd.AddCommand(svcOpenCmd)
	svcOpenCmd.AddCommand(openPipelinesCmd)
	svcOpenCmd.AddCommand(prsCmd)
	svcOpenCmd.AddCommand(openVarsCmd)
	svcOpenCmd.AddCommand(settingsCmd)
	openVarsCmd.Flags().StringVarP(&openVarsType, "type", "t", "deployment", "Type of variables page to open (deployment, repository)")
}
