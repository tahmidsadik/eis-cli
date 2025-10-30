package bitbucket

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// RestClient is a simple REST API client for Bitbucket
type RestClient struct {
	baseURL     string
	username    string
	password    string
	workspace   string
	client      *http.Client
	useOAuth    bool
	tokenStore  *TokenStore
	oauthClient *OAuthClient
}

// NewRestClient creates a new REST API client with Basic Auth
func NewRestClient(username, password, workspace string) *RestClient {
	return &RestClient{
		baseURL:   "https://api.bitbucket.org/2.0",
		username:  username,
		password:  password,
		workspace: workspace,
		useOAuth:  false,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewRestClientWithOAuth creates a new REST API client with OAuth
func NewRestClientWithOAuth(workspace string, tokenStore *TokenStore, oauthClient *OAuthClient) *RestClient {
	return &RestClient{
		baseURL:     "https://api.bitbucket.org/2.0",
		workspace:   workspace,
		useOAuth:    true,
		tokenStore:  tokenStore,
		oauthClient: oauthClient,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// doRequest performs an HTTP request with authentication
func (c *RestClient) doRequest(method, path string) (map[string]interface{}, error) {
	// Refresh token if needed (OAuth only)
	if c.useOAuth {
		if err := c.ensureValidToken(); err != nil {
			return nil, err
		}
	}

	url := c.baseURL + path

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set authentication header
	if c.useOAuth {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.tokenStore.AccessToken))
	} else {
		req.SetBasicAuth(c.username, c.password)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle 401 Unauthorized (token might be invalid)
	if resp.StatusCode == 401 && c.useOAuth {
		return nil, fmt.Errorf("authentication failed: token may be invalid (try running 'eiscli auth login' again)")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return result, nil
}

// doRequestWithBody performs an HTTP request with a JSON body
func (c *RestClient) doRequestWithBody(method, path string, body interface{}) (map[string]interface{}, error) {
	// Refresh token if needed (OAuth only)
	if c.useOAuth {
		if err := c.ensureValidToken(); err != nil {
			return nil, err
		}
	}

	url := c.baseURL + path

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest(method, url, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set authentication header
	if c.useOAuth {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.tokenStore.AccessToken))
	} else {
		req.SetBasicAuth(c.username, c.password)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle 401 Unauthorized (token might be invalid)
	if resp.StatusCode == 401 && c.useOAuth {
		return nil, fmt.Errorf("authentication failed: token may be invalid (try running 'eiscli auth login' again)")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return result, nil
}

// ListPipelines fetches pipelines for a repository
func (c *RestClient) ListPipelines(repoSlug string, limit int) ([]*Pipeline, error) {
	path := fmt.Sprintf("/repositories/%s/%s/pipelines?pagelen=%d&sort=-created_on",
		c.workspace, repoSlug, limit)

	data, err := c.doRequest("GET", path)
	if err != nil {
		return nil, err
	}

	pipelines := make([]*Pipeline, 0)

	if values, ok := data["values"].([]interface{}); ok {
		for _, v := range values {
			if pipelineData, ok := v.(map[string]interface{}); ok {
				pipeline, err := parsePipeline(pipelineData)
				if err != nil {
					fmt.Printf("Warning: failed to parse pipeline: %v\n", err)
					continue
				}
				pipelines = append(pipelines, pipeline)
			}
		}
	}

	return pipelines, nil
}

// ListPipelinesWithSteps fetches pipelines with their steps and log snippets
func (c *RestClient) ListPipelinesWithSteps(repoSlug string, limit int, logLines int) ([]*Pipeline, error) {
	pipelines, err := c.ListPipelines(repoSlug, limit)
	if err != nil {
		return nil, err
	}

	// Fetch steps for each pipeline
	for _, pipeline := range pipelines {
		steps, err := c.GetPipelineSteps(repoSlug, pipeline.UUID)
		if err != nil {
			// Don't fail the whole request if steps fail
			continue
		}

		// Fetch log snippet for each step
		for _, step := range steps {
			// Fetch logs for all completed steps
			if step.State == "COMPLETED" {
				logSnippet, err := c.GetStepLog(repoSlug, pipeline.UUID, step.UUID, logLines)
				if err == nil && logSnippet != "" {
					step.LogSnippet = logSnippet
				}
			}
		}

		pipeline.Steps = steps
	}

	return pipelines, nil
}

// ListRepositories fetches all repositories in the workspace
func (c *RestClient) ListRepositories() ([]*Repository, error) {
	path := fmt.Sprintf("/repositories/%s", c.workspace)

	data, err := c.doRequest("GET", path)
	if err != nil {
		return nil, err
	}

	repositories := make([]*Repository, 0)

	if values, ok := data["values"].([]interface{}); ok {
		for _, v := range values {
			if repoData, ok := v.(map[string]interface{}); ok {
				repo := &Repository{}

				if slug, ok := repoData["slug"].(string); ok {
					repo.Slug = slug
				}
				if name, ok := repoData["name"].(string); ok {
					repo.Name = name
				}
				if desc, ok := repoData["description"].(string); ok {
					repo.Description = desc
				}
				if fullName, ok := repoData["full_name"].(string); ok {
					repo.FullName = fullName
				}

				repositories = append(repositories, repo)
			}
		}
	}

	return repositories, nil
}

// GetPipelineSteps fetches steps for a specific pipeline
func (c *RestClient) GetPipelineSteps(repoSlug, pipelineUUID string) ([]*PipelineStep, error) {
	path := fmt.Sprintf("/repositories/%s/%s/pipelines/%s/steps/",
		c.workspace, repoSlug, pipelineUUID)

	data, err := c.doRequest("GET", path)
	if err != nil {
		return nil, err
	}

	steps := make([]*PipelineStep, 0)

	if values, ok := data["values"].([]interface{}); ok {
		for _, v := range values {
			if stepData, ok := v.(map[string]interface{}); ok {
				step := &PipelineStep{}

				if uuid, ok := stepData["uuid"].(string); ok {
					step.UUID = uuid
				}
				if name, ok := stepData["name"].(string); ok {
					step.Name = name
				}
				if stateData, ok := stepData["state"].(map[string]interface{}); ok {
					if stateName, ok := stateData["name"].(string); ok {
						step.State = stateName
					}
					if resultData, ok := stateData["result"].(map[string]interface{}); ok {
						if resultName, ok := resultData["name"].(string); ok {
							step.Result = resultName
						}
					}
				}
				if durationMs, ok := stepData["duration_in_seconds"].(float64); ok {
					step.DurationSec = int(durationMs)
				}

				steps = append(steps, step)
			}
		}
	}

	return steps, nil
}

// GetStepLog fetches the last N lines of a step's log
func (c *RestClient) GetStepLog(repoSlug, pipelineUUID, stepUUID string, lines int) (string, error) {
	// Refresh token if needed (OAuth only)
	if c.useOAuth {
		if err := c.ensureValidToken(); err != nil {
			return "", err
		}
	}

	// Use the /logs/{log_uuid} endpoint where log_uuid is the step UUID for main container
	path := fmt.Sprintf("/repositories/%s/%s/pipelines/%s/steps/%s/logs/%s",
		c.workspace, repoSlug, pipelineUUID, stepUUID, stepUUID)

	url := c.baseURL + path

	// Estimate bytes needed: ~100 bytes per line
	bytesToFetch := lines * 100
	rangeHeader := fmt.Sprintf("bytes=-%d", bytesToFetch)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set authentication header
	if c.useOAuth {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.tokenStore.AccessToken))
	} else {
		req.SetBasicAuth(c.username, c.password)
	}
	// Don't set Accept header - server returns application/octet-stream
	req.Header.Set("Range", rangeHeader)

	// Create a client that follows redirects (307 to long-term storage)
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Allow up to 10 redirects
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			// Copy auth header to redirect (Bitbucket returns 307 to S3 or similar)
			return nil
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Accept 200 (full content) or 206 (partial content from Range)
	if resp.StatusCode != 200 && resp.StatusCode != 206 {
		// Return empty string if log is not available
		return "", nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Get last N lines
	logText := string(body)
	logLines := strings.Split(logText, "\n")

	// Trim empty lines and get last N non-empty lines
	var nonEmptyLines []string
	for _, line := range logLines {
		if strings.TrimSpace(line) != "" {
			nonEmptyLines = append(nonEmptyLines, line)
		}
	}

	start := len(nonEmptyLines) - lines
	if start < 0 {
		start = 0
	}

	lastLines := nonEmptyLines[start:]
	return strings.Join(lastLines, "\n"), nil
}

// ListRepositoryVariables fetches repository-level pipeline variables
func (c *RestClient) ListRepositoryVariables(repoSlug string) ([]map[string]interface{}, error) {
	path := fmt.Sprintf("/repositories/%s/%s/pipelines_config/variables/",
		c.workspace, repoSlug)

	data, err := c.doRequest("GET", path)
	if err != nil {
		return nil, err
	}

	variables := make([]map[string]interface{}, 0)

	if values, ok := data["values"].([]interface{}); ok {
		for _, v := range values {
			if varData, ok := v.(map[string]interface{}); ok {
				variables = append(variables, varData)
			}
		}
	}

	return variables, nil
}

// CreateRepositoryVariable creates a new repository-level pipeline variable
func (c *RestClient) CreateRepositoryVariable(repoSlug, key, value string, secured bool) error {
	path := fmt.Sprintf("/repositories/%s/%s/pipelines_config/variables/",
		c.workspace, repoSlug)

	requestBody := map[string]interface{}{
		"key":     key,
		"value":   value,
		"secured": secured,
	}

	_, err := c.doRequestWithBody("POST", path, requestBody)
	if err != nil {
		return fmt.Errorf("failed to create repository variable: %w", err)
	}

	return nil
}

// ListDeploymentEnvironments fetches all deployment environments for a repository
func (c *RestClient) ListDeploymentEnvironments(repoSlug string) ([]map[string]interface{}, error) {
	path := fmt.Sprintf("/repositories/%s/%s/environments/",
		c.workspace, repoSlug)

	data, err := c.doRequest("GET", path)
	if err != nil {
		return nil, err
	}

	environments := make([]map[string]interface{}, 0)

	if values, ok := data["values"].([]interface{}); ok {
		for _, v := range values {
			if envData, ok := v.(map[string]interface{}); ok {
				environments = append(environments, envData)
			}
		}
	}

	return environments, nil
}

// ListDeploymentVariables fetches deployment variables for a specific environment
func (c *RestClient) ListDeploymentVariables(repoSlug, environmentUUID string) ([]map[string]interface{}, error) {
	path := fmt.Sprintf("/repositories/%s/%s/deployments_config/environments/%s/variables",
		c.workspace, repoSlug, environmentUUID)

	data, err := c.doRequest("GET", path)
	if err != nil {
		return nil, err
	}

	variables := make([]map[string]interface{}, 0)

	if values, ok := data["values"].([]interface{}); ok {
		for _, v := range values {
			if varData, ok := v.(map[string]interface{}); ok {
				variables = append(variables, varData)
			}
		}
	}

	return variables, nil
}

// CreateDeploymentVariable creates a new deployment variable for a specific environment
func (c *RestClient) CreateDeploymentVariable(repoSlug, environmentUUID, key, value string, secured bool) error {
	path := fmt.Sprintf("/repositories/%s/%s/deployments_config/environments/%s/variables",
		c.workspace, repoSlug, environmentUUID)

	requestBody := map[string]interface{}{
		"key":     key,
		"value":   value,
		"secured": secured,
	}

	_, err := c.doRequestWithBody("POST", path, requestBody)
	if err != nil {
		return fmt.Errorf("failed to create deployment variable: %w", err)
	}

	return nil
}

// CreateDeploymentEnvironment creates a new deployment environment for a repository
func (c *RestClient) CreateDeploymentEnvironment(repoSlug, envName, envType string, rank int) (map[string]interface{}, error) {
	path := fmt.Sprintf("/repositories/%s/%s/environments/",
		c.workspace, repoSlug)

	requestBody := map[string]interface{}{
		"name": envName,
		"environment_type": map[string]interface{}{
			"name": envType,
			"rank": rank,
		},
	}

	data, err := c.doRequestWithBody("POST", path, requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create deployment environment: %w", err)
	}

	return data, nil
}

// ListWorkspaceVariables fetches workspace-level pipeline variables
func (c *RestClient) ListWorkspaceVariables() ([]map[string]interface{}, error) {
	path := fmt.Sprintf("/workspaces/%s/pipelines-config/variables", c.workspace)

	data, err := c.doRequest("GET", path)
	if err != nil {
		return nil, err
	}

	variables := make([]map[string]interface{}, 0)

	if values, ok := data["values"].([]interface{}); ok {
		for _, v := range values {
			if varData, ok := v.(map[string]interface{}); ok {
				variables = append(variables, varData)
			}
		}
	}

	return variables, nil
}

// GetWorkspaceVariable fetches a specific workspace variable by UUID
func (c *RestClient) GetWorkspaceVariable(uuid string) (map[string]interface{}, error) {
	path := fmt.Sprintf("/workspaces/%s/pipelines-config/variables/%s", c.workspace, uuid)

	data, err := c.doRequest("GET", path)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// CreateWorkspaceVariable creates a new workspace-level pipeline variable
func (c *RestClient) CreateWorkspaceVariable(key, value string, secured bool) (map[string]interface{}, error) {
	path := fmt.Sprintf("/workspaces/%s/pipelines-config/variables", c.workspace)

	requestBody := map[string]interface{}{
		"key":     key,
		"value":   value,
		"secured": secured,
	}

	data, err := c.doRequestWithBody("POST", path, requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace variable: %w", err)
	}

	return data, nil
}

// UpdateWorkspaceVariable updates an existing workspace-level pipeline variable
func (c *RestClient) UpdateWorkspaceVariable(uuid, key, value string, secured bool) (map[string]interface{}, error) {
	path := fmt.Sprintf("/workspaces/%s/pipelines-config/variables/%s", c.workspace, uuid)

	requestBody := map[string]interface{}{
		"key":     key,
		"value":   value,
		"secured": secured,
	}

	data, err := c.doRequestWithBody("PUT", path, requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to update workspace variable: %w", err)
	}

	return data, nil
}

// ensureValidToken ensures the OAuth token is valid, refreshing if necessary
func (c *RestClient) ensureValidToken() error {
	if c.tokenStore.NeedsRefresh() {
		newTokenStore, err := c.oauthClient.RefreshAccessToken(c.tokenStore.RefreshToken)
		if err != nil {
			return fmt.Errorf("failed to refresh access token: %w (try running 'eiscli auth login' again)", err)
		}

		// Save the new tokens
		if err := newTokenStore.Save(); err != nil {
			return fmt.Errorf("failed to save refreshed token: %w", err)
		}

		// Update the current token store
		c.tokenStore = newTokenStore
	}
	return nil
}
