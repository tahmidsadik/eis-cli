package cmd

import (
	"fmt"

	"bitbucket.org/cover42/eiscli/internal/git"
)

// getServiceName retrieves the service name from args or auto-detects from git repository.
// If serviceName cannot be determined, it prints an error message and returns empty string.
func getServiceName(args []string) string {
	if len(args) > 0 {
		return args[0]
	}

	// Auto-detect service name from git repository
	detectedSlug, err := git.DetectRepositorySlug()
	if err != nil {
		fmt.Println("Error: No service name provided and could not auto-detect from git repository")
		fmt.Printf("  %v\n", err)
		fmt.Println("\nUsage:")
		fmt.Println("  1. Run this command from within a git repository, or")
		fmt.Println("  2. Provide a service name: eiscli <command> <service-name>")
		return ""
	}

	fmt.Printf("Auto-detected service from git repository: %s\n", detectedSlug)
	return detectedSlug
}
