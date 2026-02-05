package bitbucket

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// mockPaginatedResponse creates a paginated response with optional next page URL
func mockPaginatedResponse(values []map[string]interface{}, nextURL string) map[string]interface{} {
	response := map[string]interface{}{
		"values":  values,
		"page":    1,
		"pagelen": len(values),
		"size":    len(values),
	}
	if nextURL != "" {
		response["next"] = nextURL
	}
	return response
}

// TestListRepositoryVariablesPagination tests that ListRepositoryVariables handles pagination correctly
func TestListRepositoryVariablesPagination(t *testing.T) {
	tests := []struct {
		name          string
		pages         [][]map[string]interface{}
		expectedCount int
		expectedKeys  []string
	}{
		{
			name: "Single page - no pagination needed",
			pages: [][]map[string]interface{}{
				{
					{"key": "VAR1", "value": "value1", "secured": false},
					{"key": "VAR2", "value": "value2", "secured": false},
				},
			},
			expectedCount: 2,
			expectedKeys:  []string{"VAR1", "VAR2"},
		},
		{
			name: "Two pages - pagination needed",
			pages: [][]map[string]interface{}{
				{
					{"key": "VAR1", "value": "value1", "secured": false},
					{"key": "VAR2", "value": "value2", "secured": false},
				},
				{
					{"key": "VAR3", "value": "value3", "secured": false},
					{"key": "VAR4", "value": "value4", "secured": false},
				},
			},
			expectedCount: 4,
			expectedKeys:  []string{"VAR1", "VAR2", "VAR3", "VAR4"},
		},
		{
			name: "Three pages - multiple pagination",
			pages: [][]map[string]interface{}{
				{
					{"key": "VAR1", "value": "value1", "secured": false},
				},
				{
					{"key": "VAR2", "value": "value2", "secured": false},
				},
				{
					{"key": "VAR3", "value": "value3", "secured": false},
				},
			},
			expectedCount: 3,
			expectedKeys:  []string{"VAR1", "VAR2", "VAR3"},
		},
		{
			name:          "Empty response",
			pages:         [][]map[string]interface{}{{}},
			expectedCount: 0,
			expectedKeys:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pageIndex := 0

			var server *httptest.Server
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if pageIndex >= len(tt.pages) {
					t.Fatalf("Unexpected request for page %d", pageIndex)
					return
				}

				var nextURL string
				if pageIndex < len(tt.pages)-1 {
					nextURL = server.URL + "/2.0/repositories/workspace/repo/pipelines_config/variables/?pagelen=100&page=2"
				}

				response := mockPaginatedResponse(tt.pages[pageIndex], nextURL)
				pageIndex++

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			client := &RestClient{
				workspace: "workspace",
				client:    server.Client(),
				baseURL:   server.URL + "/2.0",
			}

			variables, err := client.ListRepositoryVariables("repo")
			if err != nil {
				t.Fatalf("ListRepositoryVariables returned error: %v", err)
			}

			if len(variables) != tt.expectedCount {
				t.Errorf("Expected %d variables, got %d", tt.expectedCount, len(variables))
			}

			// Verify all expected keys are present
			for _, expectedKey := range tt.expectedKeys {
				found := false
				for _, v := range variables {
					if v["key"] == expectedKey {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected key %s not found in results", expectedKey)
				}
			}
		})
	}
}

// TestListDeploymentVariablesPagination tests that ListDeploymentVariables handles pagination correctly
func TestListDeploymentVariablesPagination(t *testing.T) {
	tests := []struct {
		name          string
		pages         [][]map[string]interface{}
		expectedCount int
	}{
		{
			name: "Single page",
			pages: [][]map[string]interface{}{
				{
					{"key": "ENV_VAR1", "value": "value1", "secured": false},
					{"key": "ENV_VAR2", "value": "value2", "secured": true},
				},
			},
			expectedCount: 2,
		},
		{
			name: "Multiple pages",
			pages: [][]map[string]interface{}{
				{{"key": "VAR1", "value": "v1", "secured": false}},
				{{"key": "VAR2", "value": "v2", "secured": false}},
				{{"key": "VAR3", "value": "v3", "secured": false}},
			},
			expectedCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pageIndex := 0
			envUUID := "{test-env-uuid}"

			var server *httptest.Server
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify the URL contains the environment UUID
				if !strings.Contains(r.URL.Path, envUUID) {
					t.Errorf("Expected URL to contain environment UUID %s, got %s", envUUID, r.URL.Path)
				}

				var nextURL string
				if pageIndex < len(tt.pages)-1 {
					nextURL = server.URL + "/2.0/repositories/workspace/repo/deployments_config/environments/" + envUUID + "/variables?pagelen=100&page=2"
				}

				response := mockPaginatedResponse(tt.pages[pageIndex], nextURL)
				pageIndex++

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			client := &RestClient{
				workspace: "workspace",
				client:    server.Client(),
				baseURL:   server.URL + "/2.0",
			}

			variables, err := client.ListDeploymentVariables("repo", envUUID)
			if err != nil {
				t.Fatalf("ListDeploymentVariables returned error: %v", err)
			}

			if len(variables) != tt.expectedCount {
				t.Errorf("Expected %d variables, got %d", tt.expectedCount, len(variables))
			}
		})
	}
}

// TestListRepositoriesPagination tests that ListRepositories handles pagination correctly
func TestListRepositoriesPagination(t *testing.T) {
	tests := []struct {
		name          string
		pages         [][]map[string]interface{}
		expectedCount int
		expectedSlugs []string
	}{
		{
			name: "Single page",
			pages: [][]map[string]interface{}{
				{
					{"slug": "repo1", "name": "Repository 1", "full_name": "workspace/repo1"},
					{"slug": "repo2", "name": "Repository 2", "full_name": "workspace/repo2"},
				},
			},
			expectedCount: 2,
			expectedSlugs: []string{"repo1", "repo2"},
		},
		{
			name: "Multiple pages",
			pages: [][]map[string]interface{}{
				{{"slug": "repo1", "name": "Repo 1", "full_name": "workspace/repo1"}},
				{{"slug": "repo2", "name": "Repo 2", "full_name": "workspace/repo2"}},
				{{"slug": "repo3", "name": "Repo 3", "full_name": "workspace/repo3"}},
			},
			expectedCount: 3,
			expectedSlugs: []string{"repo1", "repo2", "repo3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pageIndex := 0

			var server *httptest.Server
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var nextURL string
				if pageIndex < len(tt.pages)-1 {
					nextURL = server.URL + "/2.0/repositories/workspace?pagelen=100&page=2"
				}

				response := mockPaginatedResponse(tt.pages[pageIndex], nextURL)
				pageIndex++

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			client := &RestClient{
				workspace: "workspace",
				client:    server.Client(),
				baseURL:   server.URL + "/2.0",
			}

			repos, err := client.ListRepositories()
			if err != nil {
				t.Fatalf("ListRepositories returned error: %v", err)
			}

			if len(repos) != tt.expectedCount {
				t.Errorf("Expected %d repositories, got %d", tt.expectedCount, len(repos))
			}

			for _, expectedSlug := range tt.expectedSlugs {
				found := false
				for _, repo := range repos {
					if repo.Slug == expectedSlug {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected slug %s not found in results", expectedSlug)
				}
			}
		})
	}
}

// TestListWorkspaceVariablesPagination tests that ListWorkspaceVariables handles pagination correctly
func TestListWorkspaceVariablesPagination(t *testing.T) {
	tests := []struct {
		name          string
		pages         [][]map[string]interface{}
		expectedCount int
	}{
		{
			name: "Single page",
			pages: [][]map[string]interface{}{
				{
					{"key": "WS_VAR1", "value": "value1", "secured": false},
				},
			},
			expectedCount: 1,
		},
		{
			name: "Multiple pages",
			pages: [][]map[string]interface{}{
				{{"key": "WS_VAR1", "value": "v1", "secured": false}},
				{{"key": "WS_VAR2", "value": "v2", "secured": false}},
			},
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pageIndex := 0

			var server *httptest.Server
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var nextURL string
				if pageIndex < len(tt.pages)-1 {
					nextURL = server.URL + "/2.0/workspaces/workspace/pipelines-config/variables?pagelen=100&page=2"
				}

				response := mockPaginatedResponse(tt.pages[pageIndex], nextURL)
				pageIndex++

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			client := &RestClient{
				workspace: "workspace",
				client:    server.Client(),
				baseURL:   server.URL + "/2.0",
			}

			variables, err := client.ListWorkspaceVariables()
			if err != nil {
				t.Fatalf("ListWorkspaceVariables returned error: %v", err)
			}

			if len(variables) != tt.expectedCount {
				t.Errorf("Expected %d variables, got %d", tt.expectedCount, len(variables))
			}
		})
	}
}

// TestListDeploymentEnvironmentsPagination tests that ListDeploymentEnvironments handles pagination correctly
func TestListDeploymentEnvironmentsPagination(t *testing.T) {
	tests := []struct {
		name          string
		pages         [][]map[string]interface{}
		expectedCount int
	}{
		{
			name: "Single page",
			pages: [][]map[string]interface{}{
				{
					{"name": "Test", "uuid": "{env-1}", "slug": "test"},
					{"name": "Production", "uuid": "{env-2}", "slug": "production"},
				},
			},
			expectedCount: 2,
		},
		{
			name: "Multiple pages",
			pages: [][]map[string]interface{}{
				{{"name": "Test", "uuid": "{env-1}", "slug": "test"}},
				{{"name": "Staging", "uuid": "{env-2}", "slug": "staging"}},
				{{"name": "Production", "uuid": "{env-3}", "slug": "production"}},
			},
			expectedCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pageIndex := 0

			var server *httptest.Server
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var nextURL string
				if pageIndex < len(tt.pages)-1 {
					nextURL = server.URL + "/2.0/repositories/workspace/repo/environments/?pagelen=100&page=2"
				}

				response := mockPaginatedResponse(tt.pages[pageIndex], nextURL)
				pageIndex++

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			client := &RestClient{
				workspace: "workspace",
				client:    server.Client(),
				baseURL:   server.URL + "/2.0",
			}

			envs, err := client.ListDeploymentEnvironments("repo")
			if err != nil {
				t.Fatalf("ListDeploymentEnvironments returned error: %v", err)
			}

			if len(envs) != tt.expectedCount {
				t.Errorf("Expected %d environments, got %d", tt.expectedCount, len(envs))
			}
		})
	}
}

// TestListDeployKeysPagination tests that ListDeployKeys handles pagination correctly
func TestListDeployKeysPagination(t *testing.T) {
	tests := []struct {
		name          string
		pages         [][]map[string]interface{}
		expectedCount int
	}{
		{
			name: "Single page",
			pages: [][]map[string]interface{}{
				{
					{"id": 1, "label": "key1", "key": "ssh-ed25519 AAAA..."},
					{"id": 2, "label": "key2", "key": "ssh-ed25519 BBBB..."},
				},
			},
			expectedCount: 2,
		},
		{
			name: "Multiple pages",
			pages: [][]map[string]interface{}{
				{{"id": 1, "label": "key1", "key": "ssh-ed25519 AAAA..."}},
				{{"id": 2, "label": "key2", "key": "ssh-ed25519 BBBB..."}},
			},
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pageIndex := 0

			var server *httptest.Server
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var nextURL string
				if pageIndex < len(tt.pages)-1 {
					nextURL = server.URL + "/2.0/repositories/workspace/repo/deploy-keys?pagelen=100&page=2"
				}

				response := mockPaginatedResponse(tt.pages[pageIndex], nextURL)
				pageIndex++

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			client := &RestClient{
				workspace: "workspace",
				client:    server.Client(),
				baseURL:   server.URL + "/2.0",
			}

			keys, err := client.ListDeployKeys("repo")
			if err != nil {
				t.Fatalf("ListDeployKeys returned error: %v", err)
			}

			if len(keys) != tt.expectedCount {
				t.Errorf("Expected %d keys, got %d", tt.expectedCount, len(keys))
			}
		})
	}
}

// TestGetPipelineStepsPagination tests that GetPipelineSteps handles pagination correctly
func TestGetPipelineStepsPagination(t *testing.T) {
	tests := []struct {
		name          string
		pages         [][]map[string]interface{}
		expectedCount int
	}{
		{
			name: "Single page",
			pages: [][]map[string]interface{}{
				{
					{"uuid": "{step-1}", "name": "Build", "state": map[string]interface{}{"name": "COMPLETED"}},
					{"uuid": "{step-2}", "name": "Test", "state": map[string]interface{}{"name": "COMPLETED"}},
				},
			},
			expectedCount: 2,
		},
		{
			name: "Multiple pages",
			pages: [][]map[string]interface{}{
				{{"uuid": "{step-1}", "name": "Build", "state": map[string]interface{}{"name": "COMPLETED"}}},
				{{"uuid": "{step-2}", "name": "Test", "state": map[string]interface{}{"name": "COMPLETED"}}},
				{{"uuid": "{step-3}", "name": "Deploy", "state": map[string]interface{}{"name": "COMPLETED"}}},
			},
			expectedCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pageIndex := 0
			pipelineUUID := "{pipeline-uuid}"

			var server *httptest.Server
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var nextURL string
				if pageIndex < len(tt.pages)-1 {
					nextURL = server.URL + "/2.0/repositories/workspace/repo/pipelines/" + pipelineUUID + "/steps/?pagelen=100&page=2"
				}

				response := mockPaginatedResponse(tt.pages[pageIndex], nextURL)
				pageIndex++

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			client := &RestClient{
				workspace: "workspace",
				client:    server.Client(),
				baseURL:   server.URL + "/2.0",
			}

			steps, err := client.GetPipelineSteps("repo", pipelineUUID)
			if err != nil {
				t.Fatalf("GetPipelineSteps returned error: %v", err)
			}

			if len(steps) != tt.expectedCount {
				t.Errorf("Expected %d steps, got %d", tt.expectedCount, len(steps))
			}
		})
	}
}

// TestListProjectsPagination tests that ListProjects handles pagination correctly
func TestListProjectsPagination(t *testing.T) {
	tests := []struct {
		name          string
		pages         [][]map[string]interface{}
		expectedCount int
	}{
		{
			name: "Single page",
			pages: [][]map[string]interface{}{
				{
					{"key": "PROJ1", "name": "Project 1"},
					{"key": "PROJ2", "name": "Project 2"},
				},
			},
			expectedCount: 2,
		},
		{
			name: "Multiple pages",
			pages: [][]map[string]interface{}{
				{{"key": "PROJ1", "name": "Project 1"}},
				{{"key": "PROJ2", "name": "Project 2"}},
				{{"key": "PROJ3", "name": "Project 3"}},
			},
			expectedCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pageIndex := 0

			var server *httptest.Server
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var nextURL string
				if pageIndex < len(tt.pages)-1 {
					nextURL = server.URL + "/2.0/workspaces/workspace/projects?pagelen=100&page=2"
				}

				response := mockPaginatedResponse(tt.pages[pageIndex], nextURL)
				pageIndex++

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			client := &RestClient{
				workspace: "workspace",
				client:    server.Client(),
				baseURL:   server.URL + "/2.0",
			}

			projects, err := client.ListProjects()
			if err != nil {
				t.Fatalf("ListProjects returned error: %v", err)
			}

			if len(projects) != tt.expectedCount {
				t.Errorf("Expected %d projects, got %d", tt.expectedCount, len(projects))
			}
		})
	}
}
