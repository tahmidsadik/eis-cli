package cmd

import (
	"fmt"
	"time"

	"bitbucket.org/cover42/eiscli/internal/bitbucket"
	"bitbucket.org/cover42/eiscli/internal/config"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication with Bitbucket",
	Long: `Manage authentication with Bitbucket using OAuth 2.0.

This command provides subcommands to:
- Login via OAuth (browser-based)
- Logout and clear stored tokens
- Check authentication status
- Manually refresh access token`,
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to Bitbucket using OAuth 2.0",
	Long: `Login to Bitbucket using OAuth 2.0 authorization flow.

This command will:
1. Open your browser for authorization
2. Start a local callback server
3. Exchange authorization code for access token
4. Save tokens securely to ~/.eiscli/tokens.json

Prerequisites:
- OAuth consumer must be configured in Bitbucket workspace settings
- client_id and client_secret must be set in config file or environment`,
	RunE: runAuthLogin,
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout and clear stored OAuth tokens",
	Long: `Logout and clear stored OAuth tokens.

This will remove the token file (~/.eiscli/tokens.json).
You will need to run 'eiscli auth login' again to use OAuth authentication.`,
	RunE: runAuthLogout,
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current authentication status",
	Long: `Show current authentication status and token information.

Displays:
- Authentication method (OAuth or Basic Auth)
- Token expiration time (if using OAuth)
- Granted scopes (if using OAuth)`,
	RunE: runAuthStatus,
}

var authRefreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Manually refresh the OAuth access token",
	Long: `Manually refresh the OAuth access token using the refresh token.

This is normally done automatically when making API calls, but you can
use this command to manually trigger a token refresh.`,
	RunE: runAuthRefresh,
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(authRefreshCmd)
}

func runAuthLogin(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Check if OAuth is configured
	if !cfg.Bitbucket.UseOAuth {
		return fmt.Errorf("OAuth is not enabled in config. Set use_oauth: true in your config file")
	}

	if cfg.Bitbucket.ClientID == "" || cfg.Bitbucket.ClientSecret == "" {
		return fmt.Errorf("OAuth credentials not configured. Please set client_id and client_secret in your config file")
	}

	// Create OAuth client
	oauthClient := bitbucket.NewOAuthClient(cfg.Bitbucket.ClientID, cfg.Bitbucket.ClientSecret)

	// Start OAuth flow
	green := color.New(color.FgGreen, color.Bold)
	fmt.Println("Starting OAuth login flow...")
	fmt.Println()

	tokenStore, err := oauthClient.StartOAuthFlow()
	if err != nil {
		return fmt.Errorf("OAuth login failed: %w", err)
	}

	// Save tokens
	if err := tokenStore.Save(); err != nil {
		return fmt.Errorf("failed to save tokens: %w", err)
	}

	// Display success
	fmt.Println()
	green.Println("✓ Authentication successful!")
	fmt.Printf("\nToken expires at: %s\n", tokenStore.ExpiresAt.Format(time.RFC1123))
	if tokenStore.Scopes != "" {
		fmt.Printf("Granted scopes: %s\n", tokenStore.Scopes)
	}
	fmt.Println("\nYou can now use eiscli commands with OAuth authentication.")

	return nil
}

func runAuthLogout(cmd *cobra.Command, args []string) error {
	tokenStore := &bitbucket.TokenStore{}

	if err := tokenStore.Clear(); err != nil {
		return fmt.Errorf("failed to logout: %w", err)
	}

	green := color.New(color.FgGreen, color.Bold)
	green.Println("✓ Logged out successfully!")
	fmt.Println("OAuth tokens have been cleared.")
	fmt.Println("Run 'eiscli auth login' to authenticate again.")

	return nil
}

func runAuthStatus(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	green := color.New(color.FgGreen, color.Bold)
	yellow := color.New(color.FgYellow, color.Bold)
	cyan := color.New(color.FgCyan, color.Bold)

	fmt.Println("Authentication Status")
	fmt.Println("====================")
	fmt.Println()

	// Check authentication method
	if cfg.Bitbucket.UseOAuth {
		cyan.Println("Authentication Method: OAuth 2.0")
		fmt.Println()

		// Try to load token
		tokenStore := &bitbucket.TokenStore{}
		if err := tokenStore.Load(); err != nil {
			yellow.Println("⚠ No OAuth token found")
			fmt.Println("Run 'eiscli auth login' to authenticate.")
			return nil
		}

		// Check token validity
		if tokenStore.IsExpired() {
			yellow.Println("⚠ Access token has expired")
			fmt.Println("It will be automatically refreshed on the next API call.")
		} else {
			green.Println("✓ Access token is valid")
		}

		fmt.Println()
		fmt.Printf("Token Type: %s\n", tokenStore.TokenType)
		fmt.Printf("Expires At: %s\n", tokenStore.ExpiresAt.Format(time.RFC1123))

		timeUntilExpiry := time.Until(tokenStore.ExpiresAt)
		if timeUntilExpiry > 0 {
			fmt.Printf("Time Until Expiry: %s\n", timeUntilExpiry.Round(time.Minute))
		}

		if tokenStore.Scopes != "" {
			fmt.Printf("Scopes: %s\n", tokenStore.Scopes)
		}

		// Get token file path
		tokenPath, _ := config.GetTokenFilePath()
		fmt.Println()
		fmt.Printf("Token File: %s\n", tokenPath)
	} else {
		cyan.Println("Authentication Method: Basic Auth (App Password)")
		fmt.Println()
		yellow.Println("⚠ You are using legacy Basic Auth")
		fmt.Println("Consider migrating to OAuth for better security.")
		fmt.Println()
		fmt.Println("To migrate:")
		fmt.Println("1. Set up OAuth consumer in Bitbucket")
		fmt.Println("2. Add client_id and client_secret to config")
		fmt.Println("3. Set use_oauth: true in config")
		fmt.Println("4. Run 'eiscli auth login'")
	}

	return nil
}

func runAuthRefresh(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if !cfg.Bitbucket.UseOAuth {
		return fmt.Errorf("OAuth is not enabled. This command only works with OAuth authentication")
	}

	// Load current token
	tokenStore := &bitbucket.TokenStore{}
	if err := tokenStore.Load(); err != nil {
		return fmt.Errorf("failed to load token: %w\n\nRun 'eiscli auth login' to authenticate first", err)
	}

	// Create OAuth client
	oauthClient := bitbucket.NewOAuthClient(cfg.Bitbucket.ClientID, cfg.Bitbucket.ClientSecret)

	// Refresh token
	fmt.Println("Refreshing access token...")
	newTokenStore, err := oauthClient.RefreshAccessToken(tokenStore.RefreshToken)
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w\n\nYou may need to run 'eiscli auth login' again", err)
	}

	// Save new token
	if err := newTokenStore.Save(); err != nil {
		return fmt.Errorf("failed to save refreshed token: %w", err)
	}

	green := color.New(color.FgGreen, color.Bold)
	green.Println("✓ Token refreshed successfully!")
	fmt.Printf("\nNew token expires at: %s\n", newTokenStore.ExpiresAt.Format(time.RFC1123))

	return nil
}
