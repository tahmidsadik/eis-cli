package bitbucket

import (
	"fmt"
	"time"

	"bitbucket.org/cover42/eiscli/internal/config"
	"bitbucket.org/cover42/eiscli/internal/git"
	"github.com/ktrysmt/go-bitbucket"
)

// Client wraps the Bitbucket API client
type Client struct {
	client      *bitbucket.Client
	restClient  *RestClient
	workspace   string
	useOAuth    bool
	tokenStore  *TokenStore
	oauthClient *OAuthClient
}

// NewClient creates a new Bitbucket client
func NewClient(cfg *config.Config) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// Determine authentication method
	if cfg.Bitbucket.UseOAuth {
		// Load or refresh OAuth token
		tokenStore, err := LoadOrRefreshToken(cfg)
		if err != nil {
			return nil, fmt.Errorf("OAuth authentication failed: %w", err)
		}

		// Create OAuth client for token refresh
		oauthClient := NewOAuthClient(cfg.Bitbucket.ClientID, cfg.Bitbucket.ClientSecret)

		// Create REST client with OAuth
		restClient := NewRestClientWithOAuth(cfg.Bitbucket.Workspace, tokenStore, oauthClient)

		// Note: go-bitbucket library doesn't support OAuth yet, so we'll use nil
		// All API calls go through RestClient anyway
		return &Client{
			client:      nil,
			restClient:  restClient,
			workspace:   cfg.Bitbucket.Workspace,
			useOAuth:    true,
			tokenStore:  tokenStore,
			oauthClient: oauthClient,
		}, nil
	}

	// Use Basic Auth (legacy)
	client := bitbucket.NewBasicAuth(cfg.Bitbucket.Username, cfg.Bitbucket.AppPassword)
	restClient := NewRestClient(cfg.Bitbucket.Username, cfg.Bitbucket.AppPassword, cfg.Bitbucket.Workspace)

	return &Client{
		client:     client,
		restClient: restClient,
		workspace:  cfg.Bitbucket.Workspace,
		useOAuth:   false,
	}, nil
}

// IsUsingOAuth returns true if the client is using OAuth authentication
func (c *Client) IsUsingOAuth() bool {
	return c.useOAuth
}

// GetTokenStore returns the token store (nil if using Basic Auth)
func (c *Client) GetTokenStore() *TokenStore {
	return c.tokenStore
}

// Pipeline represents a Bitbucket pipeline with relevant information
type Pipeline struct {
	UUID          string
	BuildNumber   int
	State         PipelineState
	CreatedOn     time.Time
	CompletedOn   *time.Time
	DurationSec   int
	Target        PipelineTarget
	Trigger       PipelineTrigger
	Creator       string
	BuildSecsUsed int
	Repository    string // full_name like "cover42/authservicev2"
	WebURL        string // Bitbucket web URL
	Steps         []*PipelineStep
}

// PipelineState represents the state of a pipeline
type PipelineState struct {
	Name   string // PENDING, IN_PROGRESS, SUCCESSFUL, FAILED, STOPPED, ERROR, PAUSED
	Result *PipelineResult
}

// PipelineResult represents the result of a pipeline
type PipelineResult struct {
	Name string // SUCCESSFUL, FAILED, ERROR, STOPPED
}

// PipelineTarget represents what the pipeline built
type PipelineTarget struct {
	RefType  string // branch, tag, etc
	RefName  string
	Commit   *Commit
	Selector *PipelineSelector
}

// Commit represents a git commit
type Commit struct {
	Hash    string
	Message string
}

// PipelineSelector represents the pipeline selector
type PipelineSelector struct {
	Type    string // branches, tags, custom, default
	Pattern string
}

// PipelineTrigger represents what triggered the pipeline
type PipelineTrigger struct {
	Type string // push, manual, schedule
}

// PipelineStep represents a step in the pipeline
type PipelineStep struct {
	UUID        string
	Name        string
	State       string // COMPLETED, FAILED, etc
	Result      string // SUCCESSFUL, FAILED, etc
	DurationSec int
	LogSnippet  string // Last few lines of log
}

// PipelinesOptions holds options for listing pipelines
type PipelinesOptions struct {
	RepoSlug string
	Limit    int
	Sort     string
	Status   string
}

// ListPipelines retrieves the last N pipelines for a repository
func (c *Client) ListPipelines(opts *PipelinesOptions) ([]*Pipeline, error) {
	if opts.RepoSlug == "" {
		return nil, fmt.Errorf("repository slug is required")
	}

	limit := opts.Limit
	if limit == 0 {
		limit = 10 // Default limit
	}

	// Use the REST client for better reliability
	return c.restClient.ListPipelines(opts.RepoSlug, limit)
}

// ListPipelinesWithSteps retrieves pipelines with their steps and log snippets
func (c *Client) ListPipelinesWithSteps(opts *PipelinesOptions, logLines int) ([]*Pipeline, error) {
	if opts.RepoSlug == "" {
		return nil, fmt.Errorf("repository slug is required")
	}

	limit := opts.Limit
	if limit == 0 {
		limit = 10 // Default limit
	}

	if logLines == 0 {
		logLines = 25 // Default log lines
	}

	// Use the REST client for better reliability
	return c.restClient.ListPipelinesWithSteps(opts.RepoSlug, limit, logLines)
}

// parsePipeline converts the API response to our Pipeline struct
func parsePipeline(data map[string]interface{}) (*Pipeline, error) {
	pipeline := &Pipeline{}

	// UUID
	if uuid, ok := data["uuid"].(string); ok {
		pipeline.UUID = uuid
	}

	// Build number
	if buildNum, ok := data["build_number"].(float64); ok {
		pipeline.BuildNumber = int(buildNum)
	}

	// Build seconds used
	if buildSecs, ok := data["build_seconds_used"].(float64); ok {
		pipeline.BuildSecsUsed = int(buildSecs)
	}

	// State
	if stateData, ok := data["state"].(map[string]interface{}); ok {
		if name, ok := stateData["name"].(string); ok {
			pipeline.State.Name = name
		}
		if resultData, ok := stateData["result"].(map[string]interface{}); ok {
			result := &PipelineResult{}
			if name, ok := resultData["name"].(string); ok {
				result.Name = name
			}
			pipeline.State.Result = result
		}
	}

	// Created on
	if createdStr, ok := data["created_on"].(string); ok {
		if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
			pipeline.CreatedOn = t
		}
	}

	// Completed on
	if completedStr, ok := data["completed_on"].(string); ok {
		if t, err := time.Parse(time.RFC3339, completedStr); err == nil {
			pipeline.CompletedOn = &t
			// Calculate duration
			pipeline.DurationSec = int(t.Sub(pipeline.CreatedOn).Seconds())
		}
	}

	// Target
	if targetData, ok := data["target"].(map[string]interface{}); ok {
		target := PipelineTarget{}

		if refType, ok := targetData["ref_type"].(string); ok {
			target.RefType = refType
		}
		if refName, ok := targetData["ref_name"].(string); ok {
			target.RefName = refName
		}

		// Commit
		if commitData, ok := targetData["commit"].(map[string]interface{}); ok {
			commit := &Commit{}
			if hash, ok := commitData["hash"].(string); ok {
				commit.Hash = hash
			}
			if message, ok := commitData["message"].(string); ok {
				commit.Message = message
			}
			target.Commit = commit
		}

		// Selector
		if selectorData, ok := targetData["selector"].(map[string]interface{}); ok {
			selector := &PipelineSelector{}
			if sType, ok := selectorData["type"].(string); ok {
				selector.Type = sType
			}
			if pattern, ok := selectorData["pattern"].(string); ok {
				selector.Pattern = pattern
			}
			target.Selector = selector
		}

		pipeline.Target = target
	}

	// Trigger
	if triggerData, ok := data["trigger"].(map[string]interface{}); ok {
		trigger := PipelineTrigger{}
		if tType, ok := triggerData["type"].(string); ok {
			trigger.Type = tType
		}
		pipeline.Trigger = trigger
	}

	// Creator
	if creatorData, ok := data["creator"].(map[string]interface{}); ok {
		if displayName, ok := creatorData["display_name"].(string); ok {
			pipeline.Creator = displayName
		} else if username, ok := creatorData["username"].(string); ok {
			pipeline.Creator = username
		}
	}

	// Repository
	if repoData, ok := data["repository"].(map[string]interface{}); ok {
		if fullName, ok := repoData["full_name"].(string); ok {
			pipeline.Repository = fullName
			// Construct web URL: https://bitbucket.org/{workspace}/{repo_slug}/pipelines/results/{build_number}
			pipeline.WebURL = fmt.Sprintf("https://bitbucket.org/%s/pipelines/results/%d",
				fullName, pipeline.BuildNumber)
		}
	}

	return pipeline, nil
}

// GetPipeline retrieves a specific pipeline by UUID
func (c *Client) GetPipeline(repoSlug, pipelineUUID string) (*Pipeline, error) {
	if repoSlug == "" || pipelineUUID == "" {
		return nil, fmt.Errorf("repository slug and pipeline UUID are required")
	}

	opts := &bitbucket.PipelinesOptions{
		Owner:    c.workspace,
		RepoSlug: repoSlug,
		IDOrUuid: pipelineUUID,
	}

	res, err := c.client.Repositories.Pipelines.Get(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get pipeline: %w", err)
	}

	if pipelineData, ok := res.(map[string]interface{}); ok {
		return parsePipeline(pipelineData)
	}

	return nil, fmt.Errorf("unexpected response format")
}

// Repository represents a Bitbucket repository
type Repository struct {
	Slug        string
	Name        string
	Description string
	FullName    string
}

// Project represents a Bitbucket project
type Project struct {
	Key         string
	Name        string
	Description string
	UUID        string
}

// Variable represents a Bitbucket pipeline or deployment variable
type Variable struct {
	Key     string
	Value   string
	Secured bool
}

// Environment represents a Bitbucket deployment environment
type Environment struct {
	UUID string
	Name string
	Type string
}

// PullRequest represents a Bitbucket pull request
type PullRequest struct {
	ID                int
	Title             string
	Description       string
	State             string // OPEN, MERGED, DECLINED, SUPERSEDED
	SourceBranch      string
	DestinationBranch string
	Author            string
	Reviewers         []string // List of reviewer UUIDs/usernames
	CreatedOn         time.Time
	UpdatedOn         time.Time
	WebURL            string
}

// PullRequestOptions holds options for listing pull requests
type PullRequestOptions struct {
	State       string // OPEN, MERGED, DECLINED, SUPERSEDED, or empty for all
	Limit       int    // Number of results to return
	Author      string // Filter by PR author (supports "@me" for current user)
	AuthorEmail string // Filter by PR author email (used when Author is "@me")
}

// CreatePullRequest creates a new pull request
func (c *Client) CreatePullRequest(repoSlug, sourceBranch, destBranch, title, description string) (*PullRequest, error) {
	if repoSlug == "" {
		return nil, fmt.Errorf("repository slug is required")
	}
	if sourceBranch == "" {
		return nil, fmt.Errorf("source branch is required")
	}
	if destBranch == "" {
		return nil, fmt.Errorf("destination branch is required")
	}
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}

	return c.restClient.CreatePullRequest(repoSlug, sourceBranch, destBranch, title, description)
}

// ListPullRequests retrieves pull requests for a repository
func (c *Client) ListPullRequests(repoSlug string, opts *PullRequestOptions) ([]*PullRequest, error) {
	if repoSlug == "" {
		return nil, fmt.Errorf("repository slug is required")
	}

	state := "OPEN" // Default to open PRs
	limit := 25     // Default limit
	author := ""
	authorEmail := ""

	if opts != nil {
		if opts.State != "" {
			state = opts.State
		}
		if opts.Limit > 0 {
			limit = opts.Limit
		}
		if opts.Author != "" {
			author = opts.Author
			// Resolve "@me" to current user's UUID/username and git email
			if author == "@me" {
				userInfo, err := c.restClient.GetCurrentUser()
				if err != nil {
					return nil, fmt.Errorf("failed to get current user: %w", err)
				}
				// Prefer UUID for matching, fallback to username
				switch {
				case userInfo.UUID != "":
					author = userInfo.UUID
				case userInfo.Username != "":
					author = userInfo.Username
				default:
					return nil, fmt.Errorf("failed to resolve current user identifier")
				}

				// Get git user email for additional filtering
				gitEmail, err := git.GetUserEmail()
				if err == nil {
					authorEmail = gitEmail
				}
				// Note: If git email can't be retrieved, we'll still filter by UUID/username
			}
		}
		// Use provided AuthorEmail if set (though typically it's only set when Author is "@me")
		if opts.AuthorEmail != "" {
			authorEmail = opts.AuthorEmail
		}
	}

	return c.restClient.ListPullRequests(repoSlug, state, limit, author, authorEmail)
}

// GetDefaultBranch retrieves the default branch for a repository
func (c *Client) GetDefaultBranch(repoSlug string) (string, error) {
	if repoSlug == "" {
		return "", fmt.Errorf("repository slug is required")
	}

	return c.restClient.GetRepositoryDefaultBranch(repoSlug)
}

// ListRepositories retrieves all repositories in the workspace
func (c *Client) ListRepositories() ([]*Repository, error) {
	// Use the REST client for better reliability
	return c.restClient.ListRepositories()
}

// ListProjects retrieves all projects in the workspace
func (c *Client) ListProjects() ([]*Project, error) {
	rawProjects, err := c.restClient.ListProjects()
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	projects := make([]*Project, 0, len(rawProjects))
	for _, projectData := range rawProjects {
		project := &Project{}

		if key, ok := projectData["key"].(string); ok {
			project.Key = key
		}
		if name, ok := projectData["name"].(string); ok {
			project.Name = name
		}
		if desc, ok := projectData["description"].(string); ok {
			project.Description = desc
		}
		if uuid, ok := projectData["uuid"].(string); ok {
			project.UUID = uuid
		}

		projects = append(projects, project)
	}

	return projects, nil
}

// GetRepositoryVariables retrieves repository-level pipeline variables
func (c *Client) GetRepositoryVariables(repoSlug string) ([]*Variable, error) {
	if repoSlug == "" {
		return nil, fmt.Errorf("repository slug is required")
	}

	rawVars, err := c.restClient.ListRepositoryVariables(repoSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository variables: %w", err)
	}

	variables := make([]*Variable, 0, len(rawVars))
	for _, varData := range rawVars {
		variable := &Variable{}

		if key, ok := varData["key"].(string); ok {
			variable.Key = key
		}
		if value, ok := varData["value"].(string); ok {
			variable.Value = value
		}
		if secured, ok := varData["secured"].(bool); ok {
			variable.Secured = secured
		}

		variables = append(variables, variable)
	}

	return variables, nil
}

// CreateRepositoryVariable creates a new repository-level pipeline variable
func (c *Client) CreateRepositoryVariable(repoSlug, key, value string, secured bool) error {
	if repoSlug == "" || key == "" {
		return fmt.Errorf("repository slug and key are required")
	}

	return c.restClient.CreateRepositoryVariable(repoSlug, key, value, secured)
}

// GetDeploymentEnvironments retrieves all deployment environments for a repository
func (c *Client) GetDeploymentEnvironments(repoSlug string) ([]*Environment, error) {
	if repoSlug == "" {
		return nil, fmt.Errorf("repository slug is required")
	}

	rawEnvs, err := c.restClient.ListDeploymentEnvironments(repoSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment environments: %w", err)
	}

	environments := make([]*Environment, 0, len(rawEnvs))
	for _, envData := range rawEnvs {
		environment := &Environment{}

		if uuid, ok := envData["uuid"].(string); ok {
			environment.UUID = uuid
		}
		if name, ok := envData["name"].(string); ok {
			environment.Name = name
		}
		if envType, ok := envData["environment_type"].(map[string]interface{}); ok {
			if typeName, ok := envType["name"].(string); ok {
				environment.Type = typeName
			}
		}

		environments = append(environments, environment)
	}

	return environments, nil
}

// GetDeploymentVariablesForEnv retrieves deployment variables for a specific environment
func (c *Client) GetDeploymentVariablesForEnv(repoSlug, envUUID string) ([]*Variable, error) {
	if repoSlug == "" || envUUID == "" {
		return nil, fmt.Errorf("repository slug and environment UUID are required")
	}

	rawVars, err := c.restClient.ListDeploymentVariables(repoSlug, envUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment variables: %w", err)
	}

	variables := make([]*Variable, 0, len(rawVars))
	for _, varData := range rawVars {
		variable := &Variable{}

		if key, ok := varData["key"].(string); ok {
			variable.Key = key
		}
		if value, ok := varData["value"].(string); ok {
			variable.Value = value
		}
		if secured, ok := varData["secured"].(bool); ok {
			variable.Secured = secured
		}

		variables = append(variables, variable)
	}

	return variables, nil
}

// CreateDeploymentVariable creates a new deployment variable for a specific environment
func (c *Client) CreateDeploymentVariable(repoSlug, envUUID, key, value string, secured bool) error {
	if repoSlug == "" || envUUID == "" || key == "" {
		return fmt.Errorf("repository slug, environment UUID, and key are required")
	}

	return c.restClient.CreateDeploymentVariable(repoSlug, envUUID, key, value, secured)
}

// CreateDeploymentEnvironment creates a new deployment environment for a repository
func (c *Client) CreateDeploymentEnvironment(repoSlug, envName, envType string) (*Environment, error) {
	if repoSlug == "" {
		return nil, fmt.Errorf("repository slug is required")
	}
	if envName == "" {
		return nil, fmt.Errorf("environment name is required")
	}

	// Validate environment name
	if err := ValidateEnvironmentName(envName); err != nil {
		return nil, err
	}

	// Validate environment type
	if err := ValidateEnvironmentType(envType); err != nil {
		return nil, err
	}

	// Get rank for the environment type
	rank := GetEnvironmentRank(envType)

	// Create the environment via REST API
	envData, err := c.restClient.CreateDeploymentEnvironment(repoSlug, envName, envType, rank)
	if err != nil {
		return nil, err
	}

	// Parse the response into Environment struct
	environment := &Environment{}

	if uuid, ok := envData["uuid"].(string); ok {
		environment.UUID = uuid
	}
	if name, ok := envData["name"].(string); ok {
		environment.Name = name
	}
	if envTypeData, ok := envData["environment_type"].(map[string]interface{}); ok {
		if typeName, ok := envTypeData["name"].(string); ok {
			environment.Type = typeName
		}
	}

	return environment, nil
}

// GetWorkspaceVariables retrieves workspace-level pipeline variables
func (c *Client) GetWorkspaceVariables() ([]*Variable, error) {
	rawVars, err := c.restClient.ListWorkspaceVariables()
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace variables: %w", err)
	}

	variables := make([]*Variable, 0, len(rawVars))
	for _, varData := range rawVars {
		variable := &Variable{}

		if key, ok := varData["key"].(string); ok {
			variable.Key = key
		}
		if value, ok := varData["value"].(string); ok {
			variable.Value = value
		}
		if secured, ok := varData["secured"].(bool); ok {
			variable.Secured = secured
		}

		// Store UUID if available for updates
		if uuid, ok := varData["uuid"].(string); ok {
			// Store UUID in a new field if needed, or we can fetch it later
			// For now, we'll need to search by key when updating
			_ = uuid
		}

		variables = append(variables, variable)
	}

	return variables, nil
}

// CreateOrUpdateWorkspaceVariable creates or updates a workspace-level pipeline variable
// Returns true if updated, false if created
func (c *Client) CreateOrUpdateWorkspaceVariable(key, value string, secured bool) (bool, string, error) {
	// Get all workspace variables
	rawVars, err := c.restClient.ListWorkspaceVariables()
	if err != nil {
		return false, "", fmt.Errorf("failed to list workspace variables: %w", err)
	}

	// Check if variable already exists
	var existingUUID string
	var existingValue string
	for _, varData := range rawVars {
		if varKey, ok := varData["key"].(string); ok && varKey == key {
			if uuid, ok := varData["uuid"].(string); ok {
				existingUUID = uuid
			}
			if val, ok := varData["value"].(string); ok {
				existingValue = val
			}
			break
		}
	}

	// Update if exists, create if not
	if existingUUID != "" {
		_, err := c.restClient.UpdateWorkspaceVariable(existingUUID, key, value, secured)
		if err != nil {
			return false, "", fmt.Errorf("failed to update workspace variable: %w", err)
		}
		return true, existingValue, nil
	}

	_, err = c.restClient.CreateWorkspaceVariable(key, value, secured)
	if err != nil {
		return false, "", fmt.Errorf("failed to create workspace variable: %w", err)
	}
	return false, "", nil
}

// CreateRepository creates a new repository in Bitbucket
func (c *Client) CreateRepository(repoSlug, projectKey string, isPrivate bool) (*Repository, error) {
	if repoSlug == "" {
		return nil, fmt.Errorf("repository slug is required")
	}

	data, err := c.restClient.CreateRepository(repoSlug, projectKey, isPrivate)
	if err != nil {
		return nil, err
	}

	// Parse the response into Repository struct
	repo := &Repository{}
	if slug, ok := data["slug"].(string); ok {
		repo.Slug = slug
	}
	if name, ok := data["name"].(string); ok {
		repo.Name = name
	}
	if desc, ok := data["description"].(string); ok {
		repo.Description = desc
	}
	if fullName, ok := data["full_name"].(string); ok {
		repo.FullName = fullName
	}

	return repo, nil
}

// SetRepositoryPermissions sets permissions for a group on a repository
func (c *Client) SetRepositoryPermissions(repoSlug, groupSlug, permission string) error {
	if repoSlug == "" || groupSlug == "" {
		return fmt.Errorf("repository slug and group slug are required")
	}
	if permission == "" {
		return fmt.Errorf("permission is required")
	}

	return c.restClient.SetRepositoryPermissions(repoSlug, groupSlug, permission)
}

// SetRepositoryDefaultBranch sets the default branch for a repository
func (c *Client) SetRepositoryDefaultBranch(repoSlug, branchName string) error {
	if repoSlug == "" {
		return fmt.Errorf("repository slug is required")
	}
	if branchName == "" {
		return fmt.Errorf("branch name is required")
	}

	return c.restClient.SetRepositoryDefaultBranch(repoSlug, branchName)
}
