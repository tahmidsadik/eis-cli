package git

import (
	"testing"
)

func TestExtractSlugFromURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		expected  string
		shouldErr bool
	}{
		{
			name:      "SSH format with .git",
			url:       "git@bitbucket.org:cover42/documentservicev2.git",
			expected:  "documentservicev2",
			shouldErr: false,
		},
		{
			name:      "SSH format without .git",
			url:       "git@bitbucket.org:cover42/documentservicev2",
			expected:  "documentservicev2",
			shouldErr: false,
		},
		{
			name:      "HTTPS format with .git",
			url:       "https://bitbucket.org/cover42/documentservicev2.git",
			expected:  "documentservicev2",
			shouldErr: false,
		},
		{
			name:      "HTTPS format without .git",
			url:       "https://bitbucket.org/cover42/documentservicev2",
			expected:  "documentservicev2",
			shouldErr: false,
		},
		{
			name:      "HTTPS with username",
			url:       "https://user@bitbucket.org/cover42/authservicev2.git",
			expected:  "authservicev2",
			shouldErr: false,
		},
		{
			name:      "HTTP format",
			url:       "http://bitbucket.org/cover42/tenantservice.git",
			expected:  "tenantservice",
			shouldErr: false,
		},
		{
			name:      "Invalid format",
			url:       "not-a-valid-url",
			expected:  "",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractSlugFromURL(tt.url)

			if tt.shouldErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("expected %q but got %q", tt.expected, result)
			}
		})
	}
}

func TestExtractSlugFromPath(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		expected  string
		shouldErr bool
	}{
		{
			name:      "Path with .git",
			path:      "cover42/documentservicev2.git",
			expected:  "documentservicev2",
			shouldErr: false,
		},
		{
			name:      "Path without .git",
			path:      "cover42/documentservicev2",
			expected:  "documentservicev2",
			shouldErr: false,
		},
		{
			name:      "Path with multiple segments",
			path:      "org/team/project/repo.git",
			expected:  "repo",
			shouldErr: false,
		},
		{
			name:      "Invalid path - no slash",
			path:      "documentservicev2",
			expected:  "",
			shouldErr: true,
		},
		{
			name:      "Invalid path - empty",
			path:      "",
			expected:  "",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractSlugFromPath(tt.path)

			if tt.shouldErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("expected %q but got %q", tt.expected, result)
			}
		})
	}
}
