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

// TestGetDefaultReviewersPagination tests that GetDefaultReviewers handles pagination correctly
func TestGetDefaultReviewersPagination(t *testing.T) {
	tests := []struct {
		name          string
		pages         [][]map[string]interface{}
		expectedCount int
		expectedUUIDs []string
	}{
		{
			name: "Single page - two reviewers",
			pages: [][]map[string]interface{}{
				{
					{"uuid": "{user-1}", "display_name": "Alice", "username": "alice"},
					{"uuid": "{user-2}", "display_name": "Bob", "username": "bob"},
				},
			},
			expectedCount: 2,
			expectedUUIDs: []string{"{user-1}", "{user-2}"},
		},
		{
			name: "Multiple pages",
			pages: [][]map[string]interface{}{
				{{"uuid": "{user-1}", "display_name": "Alice"}},
				{{"uuid": "{user-2}", "display_name": "Bob"}},
				{{"uuid": "{user-3}", "display_name": "Charlie"}},
			},
			expectedCount: 3,
			expectedUUIDs: []string{"{user-1}", "{user-2}", "{user-3}"},
		},
		{
			name:          "Empty - no default reviewers",
			pages:         [][]map[string]interface{}{{}},
			expectedCount: 0,
			expectedUUIDs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pageIndex := 0

			var server *httptest.Server
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var nextURL string
				if pageIndex < len(tt.pages)-1 {
					nextURL = server.URL + "/2.0/repositories/workspace/repo/default-reviewers?pagelen=100&page=2"
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

			reviewers, err := client.GetDefaultReviewers("repo")
			if err != nil {
				t.Fatalf("GetDefaultReviewers returned error: %v", err)
			}

			if len(reviewers) != tt.expectedCount {
				t.Errorf("Expected %d reviewers, got %d", tt.expectedCount, len(reviewers))
			}

			for _, expectedUUID := range tt.expectedUUIDs {
				found := false
				for _, r := range reviewers {
					if r["uuid"] == expectedUUID {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected UUID %s not found in results", expectedUUID)
				}
			}
		})
	}
}

// TestCreatePullRequestWithDefaultReviewers tests that CreatePullRequest includes default reviewers
func TestCreatePullRequestWithDefaultReviewers(t *testing.T) {
	tests := []struct {
		name              string
		defaultReviewers  []map[string]interface{}
		currentUserUUID   string
		expectedReviewers []string // UUIDs expected in the POST body
	}{
		{
			name: "Includes default reviewers, excludes current user",
			defaultReviewers: []map[string]interface{}{
				{"uuid": "{user-1}", "display_name": "Alice"},
				{"uuid": "{user-2}", "display_name": "Bob"},
				{"uuid": "{user-3}", "display_name": "Charlie"},
			},
			currentUserUUID:   "{user-2}",
			expectedReviewers: []string{"{user-1}", "{user-3}"},
		},
		{
			name:              "No default reviewers configured",
			defaultReviewers:  []map[string]interface{}{},
			currentUserUUID:   "{user-1}",
			expectedReviewers: nil,
		},
		{
			name: "All reviewers are current user - no reviewers in body",
			defaultReviewers: []map[string]interface{}{
				{"uuid": "{user-1}", "display_name": "Alice"},
			},
			currentUserUUID:   "{user-1}",
			expectedReviewers: nil,
		},
		{
			name: "Single reviewer who is not current user",
			defaultReviewers: []map[string]interface{}{
				{"uuid": "{user-2}", "display_name": "Bob"},
			},
			currentUserUUID:   "{user-1}",
			expectedReviewers: []string{"{user-2}"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedBody map[string]interface{}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")

				switch {
				case strings.Contains(r.URL.Path, "/default-reviewers"):
					// Return default reviewers
					resp := mockPaginatedResponse(tt.defaultReviewers, "")
					json.NewEncoder(w).Encode(resp)

				case r.URL.Path == "/2.0/user" && r.Method == "GET":
					// Return current user
					json.NewEncoder(w).Encode(map[string]interface{}{
						"uuid":     tt.currentUserUUID,
						"username": "currentuser",
					})

				case strings.Contains(r.URL.Path, "/pullrequests") && r.Method == "POST":
					// Capture the request body and return a PR response
					json.NewDecoder(r.Body).Decode(&capturedBody)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"id":    float64(1),
						"title": "Test PR",
						"state": "OPEN",
						"source": map[string]interface{}{
							"branch": map[string]interface{}{"name": "feature"},
						},
						"destination": map[string]interface{}{
							"branch": map[string]interface{}{"name": "master"},
						},
						"links": map[string]interface{}{
							"html": map[string]interface{}{
								"href": "https://bitbucket.org/workspace/repo/pull-requests/1",
							},
						},
					})

				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()

			client := &RestClient{
				workspace: "workspace",
				client:    server.Client(),
				baseURL:   server.URL + "/2.0",
			}

			pr, err := client.CreatePullRequest("repo", "feature", "master", "Test PR", "description")
			if err != nil {
				t.Fatalf("CreatePullRequest returned error: %v", err)
			}

			if pr.ID != 1 {
				t.Errorf("Expected PR ID 1, got %d", pr.ID)
			}

			// Verify reviewers in the request body
			if tt.expectedReviewers == nil {
				if _, ok := capturedBody["reviewers"]; ok {
					t.Error("Expected no reviewers field in request body, but found one")
				}
			} else {
				reviewersRaw, ok := capturedBody["reviewers"].([]interface{})
				if !ok {
					t.Fatal("Expected reviewers array in request body")
				}

				if len(reviewersRaw) != len(tt.expectedReviewers) {
					t.Errorf("Expected %d reviewers, got %d", len(tt.expectedReviewers), len(reviewersRaw))
				}

				for _, expectedUUID := range tt.expectedReviewers {
					found := false
					for _, rv := range reviewersRaw {
						if reviewer, ok := rv.(map[string]interface{}); ok {
							if reviewer["uuid"] == expectedUUID {
								found = true
								break
							}
						}
					}
					if !found {
						t.Errorf("Expected reviewer UUID %s not found in request body", expectedUUID)
					}
				}
			}
		})
	}
}

// TestCreatePullRequestDefaultReviewersFetchFails tests graceful degradation when fetching reviewers fails
func TestCreatePullRequestDefaultReviewersFetchFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.Contains(r.URL.Path, "/default-reviewers"):
			// Return an error for default reviewers
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]interface{}{
					"message": "forbidden",
				},
			})

		case strings.Contains(r.URL.Path, "/pullrequests") && r.Method == "POST":
			// PR creation should still succeed
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":    float64(42),
				"title": "Test PR",
				"state": "OPEN",
				"source": map[string]interface{}{
					"branch": map[string]interface{}{"name": "feature"},
				},
				"destination": map[string]interface{}{
					"branch": map[string]interface{}{"name": "master"},
				},
			})

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := &RestClient{
		workspace: "workspace",
		client:    server.Client(),
		baseURL:   server.URL + "/2.0",
	}

	pr, err := client.CreatePullRequest("repo", "feature", "master", "Test PR", "description")
	if err != nil {
		t.Fatalf("CreatePullRequest should succeed even if default reviewers fetch fails, got: %v", err)
	}

	if pr.ID != 42 {
		t.Errorf("Expected PR ID 42, got %d", pr.ID)
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
