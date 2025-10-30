package bitbucket

import (
	"fmt"
	"regexp"
	"strings"
)

// Environment type constants
const (
	EnvironmentTypeTest       = "Test"
	EnvironmentTypeStaging    = "Staging"
	EnvironmentTypeProduction = "Production"
)

// Environment type ranks (used by Bitbucket API)
const (
	RankTest       = 0
	RankStaging    = 1
	RankProduction = 2
)

// DetermineEnvironmentType infers the environment type from the environment name
// using common naming patterns
func DetermineEnvironmentType(envName string) string {
	nameLower := strings.ToLower(envName)

	// Test/Development patterns
	testPatterns := []string{
		"test", "testing", "dev", "development",
	}
	for _, pattern := range testPatterns {
		if nameLower == pattern {
			return EnvironmentTypeTest
		}
	}

	// Staging patterns
	stagingPatterns := []string{
		"stage", "staging",
	}
	for _, pattern := range stagingPatterns {
		if nameLower == pattern {
			return EnvironmentTypeStaging
		}
	}

	// Production patterns
	productionPatterns := []string{
		"prod", "production",
	}
	for _, pattern := range productionPatterns {
		if nameLower == pattern || strings.Contains(nameLower, pattern) {
			return EnvironmentTypeProduction
		}
	}

	// Default to Test for unknown patterns
	return EnvironmentTypeTest
}

// GetEnvironmentRank returns the rank for a given environment type
func GetEnvironmentRank(envType string) int {
	switch envType {
	case EnvironmentTypeTest:
		return RankTest
	case EnvironmentTypeStaging:
		return RankStaging
	case EnvironmentTypeProduction:
		return RankProduction
	default:
		return RankTest
	}
}

// ValidateEnvironmentName checks if an environment name is valid
// Names must be alphanumeric with optional hyphens and underscores
func ValidateEnvironmentName(name string) error {
	if name == "" {
		return fmt.Errorf("environment name cannot be empty")
	}

	if len(name) > 100 {
		return fmt.Errorf("environment name too long (max 100 characters)")
	}

	// Allow alphanumeric, hyphens, underscores, and spaces
	validPattern := regexp.MustCompile(`^[a-zA-Z0-9_\- ]+$`)
	if !validPattern.MatchString(name) {
		return fmt.Errorf("environment name can only contain letters, numbers, spaces, hyphens, and underscores")
	}

	return nil
}

// ValidateEnvironmentType checks if an environment type is valid
func ValidateEnvironmentType(envType string) error {
	validTypes := []string{
		EnvironmentTypeTest,
		EnvironmentTypeStaging,
		EnvironmentTypeProduction,
	}

	for _, validType := range validTypes {
		if envType == validType {
			return nil
		}
	}

	return fmt.Errorf("invalid environment type '%s'. Must be one of: %s",
		envType, strings.Join(validTypes, ", "))
}
