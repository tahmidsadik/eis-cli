package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	serviceName string
	environment string
	region      string
)

// IngressFile represents the structure of an ingress YAML file
type IngressFile struct {
	APIVersion string          `yaml:"apiVersion"`
	Kind       string          `yaml:"kind"`
	Metadata   IngressMetadata `yaml:"metadata"`
	Spec       IngressSpec     `yaml:"spec"`
	filePath   string
}

type IngressMetadata struct {
	Annotations map[string]string `yaml:"annotations"`
	Name        string            `yaml:"name"`
	Namespace   string            `yaml:"namespace"`
}

type IngressSpec struct {
	Rules []IngressRule `yaml:"rules"`
}

type IngressRule struct {
	HTTP IngressHTTP `yaml:"http"`
}

type IngressHTTP struct {
	Paths []IngressPath `yaml:"paths"`
}

type IngressPath struct {
	Backend IngressBackend `yaml:"backend"`
	Path    string         `yaml:"path"`
}

type IngressBackend struct {
	ServiceName string `yaml:"serviceName"`
	ServicePort int    `yaml:"servicePort"`
}

// AWSAPIConfig represents the structure of the aws-api-configs annotation
type AWSAPIConfig struct {
	Name                 string       `json:"name"`
	AuthorizationEnabled bool         `json:"authorization_enabled"`
	Authorizers          []Authorizer `json:"authorizers"`
}

type Authorizer struct {
	ExcludedPaths  []string `json:"excluded_paths"`
	AuthorizerType string   `json:"authorizer_type"`
	AuthorizerName string   `json:"authorizer_name"`
	LambdaARN      string   `json:"lambda_arn"`
	IDSource       string   `json:"id_source"`
}

// ConfigStatus represents the configuration status of a service in an ingress file
type ConfigStatus struct {
	FilePath        string
	HasAllPaths     bool
	FoundPaths      []string
	HasExcludedAPI  bool
	HasExcludedJSON bool
	IsConfigured    bool
	IsMisconfigured bool
}

var svcIngressCmd = &cobra.Command{
	Use:   "ingress",
	Short: "Manage service ingress configurations",
	Long:  `Manage service ingress configurations in the API Gateway.`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var svcIngressAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a service to the ingress controller",
	Long: `Add a service to the API Gateway ingress controller configuration.
This command will register the service in all API ingress files for the specified environment and region.`,
	Run: runIngressAdd,
}

func init() {
	svcCmd.AddCommand(svcIngressCmd)
	svcIngressCmd.AddCommand(svcIngressAddCmd)

	svcIngressAddCmd.Flags().StringVarP(&serviceName, "service", "s", "", "Service name (required)")
	svcIngressAddCmd.Flags().StringVarP(&environment, "env", "e", "", "Environment (required)")
	svcIngressAddCmd.Flags().StringVarP(&region, "region", "r", "frankfurt", "Region (default: frankfurt)")
	_ = svcIngressAddCmd.MarkFlagRequired("service")
	_ = svcIngressAddCmd.MarkFlagRequired("env")
}

func runIngressAdd(cmd *cobra.Command, args []string) {
	// Verify we're in the dist-orchestration repository
	if err := verifyRepository(); err != nil {
		color.Red("✗ %s", err.Error())
		os.Exit(1)
	}

	// Build the ingress directory path
	basePath := getBasePath()
	var ingressDir string
	if region == "zurich" {
		// Zurich doesn't have environment subdirectories, it's directly prod
		if environment != "prod" {
			color.Red("✗ Zurich region only supports 'prod' environment")
			os.Exit(1)
		}
		if basePath == "" {
			ingressDir = filepath.Join("ingress", "zurich")
		} else {
			ingressDir = filepath.Join(basePath, "ingress", "zurich")
		}
	} else {
		// Frankfurt and other regions use environment subdirectories
		if basePath == "" {
			ingressDir = filepath.Join("ingress", region, environment)
		} else {
			ingressDir = filepath.Join(basePath, "ingress", region, environment)
		}
	}

	// Check if environment directory exists
	if _, err := os.Stat(ingressDir); os.IsNotExist(err) {
		color.Red("✗ Environment directory not found: %s", ingressDir)
		os.Exit(1)
	}

	// Find all *api_ingress.yaml files
	files, err := findAPIIngressFiles(ingressDir)
	if err != nil {
		color.Red("✗ Error finding ingress files: %v", err)
		os.Exit(1)
	}

	if len(files) == 0 {
		color.Yellow("⚠ No API ingress files found in %s", ingressDir)
		os.Exit(0)
	}

	// Process each file
	var updated []string
	var skipped []string
	var issues []string

	for _, file := range files {
		status, err := checkServiceConfiguration(file, serviceName)
		if err != nil {
			color.Red("✗ Error checking %s: %v", file, err)
			issues = append(issues, fmt.Sprintf("%s (error: %v)", file, err))
			continue
		}

		if status.IsConfigured {
			skipped = append(skipped, file)
			continue
		}

		if status.IsMisconfigured {
			issueDetail := fmt.Sprintf("%s - Found paths: %v, ExcludedAPI: %v, ExcludedJSON: %v",
				file, status.FoundPaths, status.HasExcludedAPI, status.HasExcludedJSON)
			issues = append(issues, issueDetail)
			continue
		}

		// Add configuration
		if err := addServiceToIngress(file, serviceName); err != nil {
			color.Red("✗ Error updating %s: %v", file, err)
			issues = append(issues, fmt.Sprintf("%s (error: %v)", file, err))
			continue
		}

		updated = append(updated, file)
	}

	// Print summary
	printSummary(environment, region, serviceName, updated, skipped, issues)
}

func verifyRepository() error {
	// Check if we're in the dist-orchestration directory (ingress exists directly)
	if _, err := os.Stat("ingress"); err == nil {
		return nil
	}

	// Check if we're in the parent directory (dist-orchestration/ingress exists)
	ingressPath := filepath.Join("dist-orchestration", "ingress")
	if _, err := os.Stat(ingressPath); err == nil {
		return nil
	}

	return fmt.Errorf("This command must be run from the dist-orchestration repository root directory")
}

// getBasePath returns the base path for dist-orchestration files
// Returns "" if we're already in dist-orchestration, or "dist-orchestration" if we're in parent
func getBasePath() string {
	// Check if we're in the dist-orchestration directory
	if _, err := os.Stat("ingress"); err == nil {
		return ""
	}

	// We're in the parent directory
	return "dist-orchestration"
}

func findAPIIngressFiles(dir string) ([]string, error) {
	var files []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasSuffix(name, "api_ingress.yaml") && !strings.Contains(name, "workflow") {
			files = append(files, filepath.Join(dir, name))
		}
	}

	return files, nil
}

func checkServiceConfiguration(filePath, serviceName string) (*ConfigStatus, error) {
	status := &ConfigStatus{
		FilePath: filePath,
	}

	ingress, err := loadIngressFile(filePath)
	if err != nil {
		return nil, err
	}

	// Check paths
	expectedPaths := []string{
		fmt.Sprintf("/%s", serviceName),
		fmt.Sprintf("/%s/api", serviceName),
		fmt.Sprintf("/%s/api-json", serviceName),
	}

	foundPaths := make(map[string]bool)
	if len(ingress.Spec.Rules) > 0 && len(ingress.Spec.Rules[0].HTTP.Paths) > 0 {
		for _, path := range ingress.Spec.Rules[0].HTTP.Paths {
			for _, expected := range expectedPaths {
				if path.Path == expected && path.Backend.ServiceName == serviceName {
					foundPaths[expected] = true
					status.FoundPaths = append(status.FoundPaths, expected)
				}
			}
		}
	}

	status.HasAllPaths = len(foundPaths) == 3

	// Check excluded paths
	if configStr, ok := ingress.Metadata.Annotations["apigateway.ingress.kubernetes.io/aws-api-configs"]; ok {
		var configs []AWSAPIConfig
		if err := json.Unmarshal([]byte(configStr), &configs); err == nil {
			if len(configs) > 0 && len(configs[0].Authorizers) > 0 {
				excludedPaths := configs[0].Authorizers[0].ExcludedPaths
				for _, path := range excludedPaths {
					if path == fmt.Sprintf("/%s/api", serviceName) {
						status.HasExcludedAPI = true
					}
					if path == fmt.Sprintf("/%s/api-json", serviceName) {
						status.HasExcludedJSON = true
					}
				}
			}
		}
	}

	// Determine configuration state
	allExcluded := status.HasExcludedAPI && status.HasExcludedJSON
	status.IsConfigured = status.HasAllPaths && allExcluded

	// Misconfigured if we have some but not all configuration
	hasAnyPath := len(status.FoundPaths) > 0
	hasAnyExcluded := status.HasExcludedAPI || status.HasExcludedJSON
	status.IsMisconfigured = (hasAnyPath || hasAnyExcluded) && !status.IsConfigured

	return status, nil
}

func loadIngressFile(filePath string) (*IngressFile, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var ingress IngressFile
	if err := yaml.Unmarshal(data, &ingress); err != nil {
		return nil, err
	}

	ingress.filePath = filePath
	return &ingress, nil
}

func addServiceToIngress(filePath, serviceName string) error {
	ingress, err := loadIngressFile(filePath)
	if err != nil {
		return err
	}

	// Add paths to spec
	newPaths := []IngressPath{
		{
			Backend: IngressBackend{
				ServiceName: serviceName,
				ServicePort: 80,
			},
			Path: fmt.Sprintf("/%s", serviceName),
		},
		{
			Backend: IngressBackend{
				ServiceName: serviceName,
				ServicePort: 80,
			},
			Path: fmt.Sprintf("/%s/api", serviceName),
		},
		{
			Backend: IngressBackend{
				ServiceName: serviceName,
				ServicePort: 80,
			},
			Path: fmt.Sprintf("/%s/api-json", serviceName),
		},
	}

	if len(ingress.Spec.Rules) > 0 {
		ingress.Spec.Rules[0].HTTP.Paths = append(ingress.Spec.Rules[0].HTTP.Paths, newPaths...)
	}

	// Add excluded paths to annotation
	if configStr, ok := ingress.Metadata.Annotations["apigateway.ingress.kubernetes.io/aws-api-configs"]; ok {
		var configs []AWSAPIConfig
		if err := json.Unmarshal([]byte(configStr), &configs); err != nil {
			return fmt.Errorf("failed to parse aws-api-configs: %w", err)
		}

		if len(configs) > 0 && len(configs[0].Authorizers) > 0 {
			configs[0].Authorizers[0].ExcludedPaths = append(
				configs[0].Authorizers[0].ExcludedPaths,
				fmt.Sprintf("/%s/api", serviceName),
				fmt.Sprintf("/%s/api-json", serviceName),
			)

			updatedConfig, err := json.Marshal(configs)
			if err != nil {
				return fmt.Errorf("failed to marshal aws-api-configs: %w", err)
			}

			ingress.Metadata.Annotations["apigateway.ingress.kubernetes.io/aws-api-configs"] = string(updatedConfig)
		}
	}

	// Write back to file
	data, err := yaml.Marshal(ingress)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func printSummary(env, region, service string, updated, skipped, issues []string) {
	fmt.Println()
	color.Cyan("═══════════════════════════════════════════════════════")
	color.Cyan("                    SUMMARY")
	color.Cyan("═══════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Printf("Environment: %s (%s)\n", color.YellowString(env), color.YellowString(region))
	fmt.Printf("Service: %s\n", color.CyanString(service))
	fmt.Println()

	if len(updated) > 0 {
		color.Green("✓ Updated files:")
		for _, file := range updated {
			fmt.Printf("  • %s\n", file)
		}
		fmt.Println()
	}

	if len(skipped) > 0 {
		color.Yellow("⊘ Skipped files (already configured):")
		for _, file := range skipped {
			fmt.Printf("  • %s\n", file)
		}
		fmt.Println()
	}

	if len(issues) > 0 {
		color.Red("✗ Files with issues (manual intervention required):")
		for _, issue := range issues {
			fmt.Printf("  • %s\n", issue)
		}
		fmt.Println()
		color.Yellow("Please manually remove partial configurations and run the command again.")
		fmt.Println()
	}

	if len(updated) > 0 && len(issues) == 0 {
		color.Green("✓ All files updated successfully!")
	}

	color.Cyan("═══════════════════════════════════════════════════════")
}
