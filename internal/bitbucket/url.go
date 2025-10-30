package bitbucket

import "fmt"

// BuildRepositoryURL constructs the main Bitbucket repository URL
func BuildRepositoryURL(workspace, repoSlug string) string {
	return fmt.Sprintf("https://bitbucket.org/%s/%s", workspace, repoSlug)
}

// BuildPipelinesURL constructs the Bitbucket pipelines page URL
func BuildPipelinesURL(workspace, repoSlug string) string {
	return fmt.Sprintf("https://bitbucket.org/%s/%s/pipelines", workspace, repoSlug)
}

// BuildPullRequestsURL constructs the Bitbucket pull requests page URL
func BuildPullRequestsURL(workspace, repoSlug string) string {
	return fmt.Sprintf("https://bitbucket.org/%s/%s/pull-requests", workspace, repoSlug)
}

// BuildDeploymentVariablesURL constructs the Bitbucket deployment variables settings page URL
func BuildDeploymentVariablesURL(workspace, repoSlug string) string {
	return fmt.Sprintf("https://bitbucket.org/%s/%s/admin/addon/admin/pipelines/deployment-settings", workspace, repoSlug)
}

// BuildSettingsURL constructs the Bitbucket repository settings page URL
func BuildSettingsURL(workspace, repoSlug string) string {
	return fmt.Sprintf("https://bitbucket.org/%s/%s/admin", workspace, repoSlug)
}
