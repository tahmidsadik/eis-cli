package bitbucket

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"bitbucket.org/cover42/eiscli/internal/config"
)

const (
	authURL  = "https://bitbucket.org/site/oauth2/authorize"
	tokenURL = "https://bitbucket.org/site/oauth2/access_token"
)

// Build-time injected OAuth credentials (set via -ldflags during build)
// These are shared across all users and baked into the binary
var (
	// DefaultClientID is the OAuth consumer client ID (injected at build time)
	DefaultClientID string

	// DefaultClientSecret is the OAuth consumer client secret (injected at build time)
	DefaultClientSecret string
)

func init() {
	// Register build-time defaults with the config package
	// This avoids import cycles
	config.RegisterBuildTimeOAuthDefaults(func() string { return DefaultClientID }, func() string { return DefaultClientSecret })
}

// OAuth scopes required for the CLI
var requiredScopes = []string{
	"account",           // Read account info (for @me in PR filters)
	"repository",        // Read repositories
	"pullrequest",       // Read pull requests
	"pullrequest:write", // Create and manage pull requests
	"pipeline",          // View pipelines
	"pipeline:write",    // Trigger/manage pipelines
	"pipeline:variable", // Manage pipeline variables
	"webhook",           // Manage webhooks
}

// OAuthClient handles OAuth 2.0 authentication flow
type OAuthClient struct {
	ClientID     string
	ClientSecret string
	tokenStore   *TokenStore
}

// NewOAuthClient creates a new OAuth client
func NewOAuthClient(clientID, clientSecret string) *OAuthClient {
	return &OAuthClient{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		tokenStore:   &TokenStore{},
	}
}

// tokenResponse represents the OAuth token response
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scopes       string `json:"scopes"`
}

// StartOAuthFlow initiates the OAuth authorization flow
func (oc *OAuthClient) StartOAuthFlow() (*TokenStore, error) {
	// Generate a random state for CSRF protection
	state, err := generateState()
	if err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}

	// Start local callback server
	port, callbackChan, errChan, cleanup := oc.startCallbackServer(state)
	defer cleanup()

	// Build authorization URL
	authURL := oc.buildAuthURL(port, state)

	// Open browser
	fmt.Printf("Opening browser for authorization...\n")
	fmt.Printf("Callback server listening on port: %d\n", port)
	fmt.Printf("\nIf the browser doesn't open, visit this URL:\n%s\n\n", authURL)
	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Failed to open browser automatically: %v\n", err)
	}

	// Wait for callback or timeout
	select {
	case code := <-callbackChan:
		fmt.Println("Authorization successful! Exchanging code for token...")
		return oc.exchangeCodeForToken(code, port)
	case err := <-errChan:
		fmt.Printf("\n❌ Authorization error from Bitbucket:\n")
		fmt.Printf("   %v\n\n", err)
		fmt.Println("Common issues:")
		fmt.Println("  1. Callback URL mismatch - Check your OAuth consumer settings")
		fmt.Printf("     Expected: http://localhost:%d/callback\n", port)
		fmt.Println("  2. Invalid client credentials - Verify client_id and client_secret")
		fmt.Println("  3. Missing permissions - Ensure all required scopes are granted")
		return nil, fmt.Errorf("authorization failed: %w", err)
	case <-time.After(5 * time.Minute):
		return nil, fmt.Errorf("authorization timed out after 5 minutes")
	}
}

// startCallbackServer starts a local HTTP server to receive the OAuth callback
func (oc *OAuthClient) startCallbackServer(expectedState string) (int, chan string, chan error, func()) {
	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	// Find an available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		errChan <- err
		return 0, codeChan, errChan, func() {}
	}

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		errChan <- fmt.Errorf("failed to get TCP address")
		return 0, codeChan, errChan, func() {}
	}
	port := addr.Port

	// Create HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		// Check for errors from OAuth provider
		if errMsg := r.URL.Query().Get("error"); errMsg != "" {
			errDesc := r.URL.Query().Get("error_description")
			errChan <- fmt.Errorf("OAuth error: %s - %s", errMsg, errDesc)
			http.Error(w, "Authorization failed. You can close this window.", http.StatusBadRequest)
			return
		}

		// Validate state
		state := r.URL.Query().Get("state")
		if state != expectedState {
			errChan <- fmt.Errorf("invalid state parameter")
			http.Error(w, "Invalid state parameter. You can close this window.", http.StatusBadRequest)
			return
		}

		// Get authorization code
		code := r.URL.Query().Get("code")
		if code == "" {
			errChan <- fmt.Errorf("no authorization code received")
			http.Error(w, "No authorization code received. You can close this window.", http.StatusBadRequest)
			return
		}

		// Send success response to browser
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `
			<html>
			<head>
				<meta charset="UTF-8">
				<title>Authorization Successful</title>
			</head>
			<body style="font-family: sans-serif; text-align: center; padding: 50px;">
				<h1 style="color: #0052CC;">✓ Authorization Successful!</h1>
				<p>You can close this window and return to the terminal.</p>
			</body>
			</html>
		`)

		codeChan <- code
	})

	server := &http.Server{
		Handler: mux,
	}

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx) // Ignore error during cleanup
	}

	return port, codeChan, errChan, cleanup
}

// buildAuthURL constructs the OAuth authorization URL
func (oc *OAuthClient) buildAuthURL(port int, state string) string {
	params := url.Values{}
	params.Set("client_id", oc.ClientID)
	params.Set("response_type", "code")
	params.Set("state", state)
	params.Set("scope", strings.Join(requiredScopes, " "))
	// Callback URL must match what's configured in OAuth consumer
	params.Set("redirect_uri", fmt.Sprintf("http://localhost:%d/callback", port))

	return fmt.Sprintf("%s?%s", authURL, params.Encode())
}

// exchangeCodeForToken exchanges the authorization code for access and refresh tokens
func (oc *OAuthClient) exchangeCodeForToken(code string, port int) (*TokenStore, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", fmt.Sprintf("http://localhost:%d/callback", port))

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}

	req.SetBasicAuth(oc.ClientID, oc.ClientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed (status %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp tokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	// Calculate expiration time
	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	tokenStore := &TokenStore{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    expiresAt,
		TokenType:    tokenResp.TokenType,
		Scopes:       tokenResp.Scopes,
	}

	return tokenStore, nil
}

// RefreshAccessToken refreshes the access token using the refresh token
func (oc *OAuthClient) RefreshAccessToken(refreshToken string) (*TokenStore, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh request: %w", err)
	}

	req.SetBasicAuth(oc.ClientID, oc.ClientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read refresh response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed (status %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp tokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse refresh response: %w", err)
	}

	// Calculate expiration time
	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	tokenStore := &TokenStore{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    expiresAt,
		TokenType:    tokenResp.TokenType,
		Scopes:       tokenResp.Scopes,
	}

	return tokenStore, nil
}

// LoadOrRefreshToken loads the token from storage or refreshes if expired
func LoadOrRefreshToken(cfg *config.Config) (*TokenStore, error) {
	tokenStore := &TokenStore{}
	if err := tokenStore.Load(); err != nil {
		return nil, err
	}

	// If token needs refresh, refresh it
	if tokenStore.NeedsRefresh() {
		oauthClient := NewOAuthClient(cfg.Bitbucket.ClientID, cfg.Bitbucket.ClientSecret)
		newTokenStore, err := oauthClient.RefreshAccessToken(tokenStore.RefreshToken)
		if err != nil {
			return nil, fmt.Errorf("failed to refresh token: %w (you may need to run 'eiscli auth login' again)", err)
		}

		// Save the new tokens
		if err := newTokenStore.Save(); err != nil {
			return nil, fmt.Errorf("failed to save refreshed token: %w", err)
		}

		return newTokenStore, nil
	}

	return tokenStore, nil
}

// generateState generates a random state string for CSRF protection
func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// openBrowser opens the specified URL in the default browser
func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "linux":
		cmd = "xdg-open"
		args = []string{url}
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	default:
		return fmt.Errorf("unsupported platform")
	}

	return exec.Command(cmd, args...).Start()
}
