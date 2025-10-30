package git

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
)

// DetectRepositorySlug attempts to detect the repository slug from the current directory's git remote
func DetectRepositorySlug() (string, error) {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Open git repository
	repo, err := git.PlainOpenWithOptions(cwd, &git.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		return "", fmt.Errorf("not a git repository or git repository not found: %w", err)
	}

	// Get remote config
	remote, err := repo.Remote("origin")
	if err != nil {
		return "", fmt.Errorf("no 'origin' remote found: %w", err)
	}

	// Get the first URL (usually there's only one or they're the same)
	if len(remote.Config().URLs) == 0 {
		return "", fmt.Errorf("no remote URL configured for 'origin'")
	}

	remoteURL := remote.Config().URLs[0]

	// Extract repository slug from the URL
	slug, err := extractSlugFromURL(remoteURL)
	if err != nil {
		return "", fmt.Errorf("failed to extract repository slug from URL '%s': %w", remoteURL, err)
	}

	return slug, nil
}

// extractSlugFromURL extracts the repository slug from various git URL formats
// Supports:
// - SSH: git@bitbucket.org:cover42/documentservicev2.git
// - HTTPS: https://bitbucket.org/cover42/documentservicev2.git
// - HTTPS with auth: https://user@bitbucket.org/cover42/documentservicev2.git
func extractSlugFromURL(remoteURL string) (string, error) {
	// Handle SSH format: git@bitbucket.org:cover42/documentservicev2.git
	if strings.HasPrefix(remoteURL, "git@") {
		// Remove "git@"
		remoteURL = strings.TrimPrefix(remoteURL, "git@")

		// Split by ':'
		parts := strings.Split(remoteURL, ":")
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid SSH URL format")
		}

		// Get the path part (cover42/documentservicev2.git)
		pathPart := parts[1]

		// Extract slug
		return extractSlugFromPath(pathPart)
	}

	// Handle HTTPS format
	if strings.HasPrefix(remoteURL, "https://") || strings.HasPrefix(remoteURL, "http://") {
		u, err := url.Parse(remoteURL)
		if err != nil {
			return "", fmt.Errorf("invalid HTTPS URL: %w", err)
		}

		// Path will be like /cover42/documentservicev2.git
		return extractSlugFromPath(strings.TrimPrefix(u.Path, "/"))
	}

	return "", fmt.Errorf("unsupported URL format")
}

// extractSlugFromPath extracts the repository slug from a path
// Examples:
// - cover42/documentservicev2.git -> documentservicev2
// - cover42/documentservicev2 -> documentservicev2
func extractSlugFromPath(path string) (string, error) {
	// Remove trailing .git if present
	path = strings.TrimSuffix(path, ".git")

	// Split by '/' and get the last part
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid path format: expected workspace/repo-slug")
	}

	// The slug is the last part
	slug := parts[len(parts)-1]

	if slug == "" {
		return "", fmt.Errorf("empty repository slug")
	}

	return slug, nil
}

// IsGitRepository checks if the current directory is within a git repository
func IsGitRepository() bool {
	cwd, err := os.Getwd()
	if err != nil {
		return false
	}

	// Try to find .git directory by traversing up
	dir := cwd
	for {
		gitDir := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			return true
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root
			break
		}
		dir = parent
	}

	return false
}
