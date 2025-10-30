package aws

import (
	"strings"

	"bitbucket.org/cover42/eiscli/internal/config"
)

// GetProfileForEnvironment returns the AWS profile to use for a given environment
func GetProfileForEnvironment(env string, cfg *config.Config) string {
	envLower := strings.ToLower(env)

	// Non-production environments use the nonprod profile
	nonProdEnvs := map[string]bool{
		"staging":     true,
		"development": true,
		"dev":         true,
	}

	if nonProdEnvs[envLower] {
		return cfg.AWS.NonProdProfile
	}

	// All other environments (testing, test, production, production-zurich) use default profile
	return cfg.AWS.DefaultProfile
}

// GetProfileDescription returns a human-readable description of which profile is used
func GetProfileDescription(env string, cfg *config.Config) string {
	profile := GetProfileForEnvironment(env, cfg)
	envLower := strings.ToLower(env)

	nonProdEnvs := map[string]bool{
		"staging":     true,
		"development": true,
		"dev":         true,
	}

	if nonProdEnvs[envLower] {
		return "Non-Production (" + profile + ")"
	}
	return "Production (" + profile + ")"
}
