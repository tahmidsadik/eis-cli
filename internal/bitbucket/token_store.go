package bitbucket

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"bitbucket.org/cover42/eiscli/internal/config"
)

// TokenStore handles OAuth token storage and retrieval
type TokenStore struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
	Scopes       string    `json:"scopes,omitempty"`
}

// Load reads tokens from the token file
func (ts *TokenStore) Load() error {
	tokenPath, err := config.GetTokenFilePath()
	if err != nil {
		return fmt.Errorf("failed to get token file path: %w", err)
	}

	data, err := os.ReadFile(tokenPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("token file not found: please run 'eiscli auth login' first")
		}
		return fmt.Errorf("failed to read token file: %w", err)
	}

	if err := json.Unmarshal(data, ts); err != nil {
		return fmt.Errorf("failed to parse token file: %w", err)
	}

	return nil
}

// Save writes tokens to the token file with secure permissions
func (ts *TokenStore) Save() error {
	tokenPath, err := config.GetTokenFilePath()
	if err != nil {
		return fmt.Errorf("failed to get token file path: %w", err)
	}

	// Ensure the directory exists
	tokenDir := filepath.Dir(tokenPath)
	if err := os.MkdirAll(tokenDir, 0o700); err != nil {
		return fmt.Errorf("failed to create token directory: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(ts, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tokens: %w", err)
	}

	// Write with secure permissions (0o600 = read/write for owner only)
	if err := os.WriteFile(tokenPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	return nil
}

// IsExpired checks if the access token has expired or will expire soon
func (ts *TokenStore) IsExpired() bool {
	// Consider expired if less than 5 minutes remaining
	return time.Now().Add(5 * time.Minute).After(ts.ExpiresAt)
}

// Clear removes the token file (for logout)
func (ts *TokenStore) Clear() error {
	tokenPath, err := config.GetTokenFilePath()
	if err != nil {
		return fmt.Errorf("failed to get token file path: %w", err)
	}

	if err := os.Remove(tokenPath); err != nil {
		if os.IsNotExist(err) {
			return nil // Already cleared
		}
		return fmt.Errorf("failed to remove token file: %w", err)
	}

	return nil
}

// IsValid checks if the token store has valid tokens
func (ts *TokenStore) IsValid() bool {
	return ts.AccessToken != "" && ts.RefreshToken != "" && !ts.IsExpired()
}

// NeedsRefresh checks if the token needs to be refreshed
func (ts *TokenStore) NeedsRefresh() bool {
	return ts.AccessToken != "" && ts.RefreshToken != "" && ts.IsExpired()
}
