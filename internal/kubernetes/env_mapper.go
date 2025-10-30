package kubernetes

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// MapOverlayToEnvironment maps Kubernetes overlay folder names to Bitbucket environment names
func MapOverlayToEnvironment(overlayName string) string {
	mapping := map[string]string{
		"testing":     "Test",
		"staging":     "Staging",
		"prod":        "Production",
		"prod-zurich": "Production-Zurich",
		"dev":         "Development",
	}

	// Case-insensitive lookup
	lowerOverlay := strings.ToLower(overlayName)
	if envName, ok := mapping[lowerOverlay]; ok {
		return envName
	}

	// If no mapping found, return title case of the overlay name
	return cases.Title(language.English).String(overlayName)
}

// GetAvailableOverlays lists available overlay directories in the kubernetes folder
func GetAvailableOverlays(kubernetesPath string) ([]string, error) {
	overlaysPath := filepath.Join(kubernetesPath, "overlays")

	// Check if overlays directory exists
	if _, err := os.Stat(overlaysPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("overlays directory not found at: %s", overlaysPath)
	}

	entries, err := os.ReadDir(overlaysPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read overlays directory: %w", err)
	}

	var overlays []string
	for _, entry := range entries {
		if entry.IsDir() {
			overlays = append(overlays, entry.Name())
		}
	}

	return overlays, nil
}

// FindEnvTemplate finds the .env.template file for a given overlay
func FindEnvTemplate(kubernetesPath, overlayName string) (string, error) {
	templatePath := filepath.Join(kubernetesPath, "overlays", overlayName, ".env.template")

	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		return "", fmt.Errorf(".env.template file not found at: %s", templatePath)
	}

	return templatePath, nil
}
