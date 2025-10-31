package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// GetTokenFilePath returns the path to the OAuth token storage file
func GetTokenFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".eiscli", "tokens.json"), nil
}

// Config holds the application configuration
type Config struct {
	Bitbucket  BitbucketConfig  `mapstructure:"bitbucket"`
	Deployment DeploymentConfig `mapstructure:"deployment"`
	AWS        AWSConfig        `mapstructure:"aws"`
}

// BitbucketConfig holds Bitbucket-specific configuration
type BitbucketConfig struct {
	// OAuth 2.0 configuration (recommended)
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	UseOAuth     bool   `mapstructure:"use_oauth"`

	// Basic Auth configuration (legacy)
	Username    string `mapstructure:"username"`
	AppPassword string `mapstructure:"app_password"`

	Workspace string `mapstructure:"workspace"`
}

// DeploymentConfig holds deployment-related configuration
type DeploymentConfig struct {
	AutoCreateEnvironments bool   `mapstructure:"auto_create_environments"`
	DefaultEnvironmentType string `mapstructure:"default_environment_type"`
}

// AWSConfig holds AWS-specific configuration
type AWSConfig struct {
	DefaultProfile string `mapstructure:"default_profile"`
	NonProdProfile string `mapstructure:"nonprod_profile"`
	Region         string `mapstructure:"region"`
}

var globalConfig *Config

// Load reads configuration from file and environment variables
func Load() (*Config, error) {
	if globalConfig != nil {
		return globalConfig, nil
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// Add config search paths
	home, err := os.UserHomeDir()
	if err == nil {
		viper.AddConfigPath(filepath.Join(home, ".eiscli"))
	}
	viper.AddConfigPath(".")

	// Environment variable overrides
	viper.SetEnvPrefix("EISCLI")
	viper.AutomaticEnv()

	// Bind specific environment variables
	_ = viper.BindEnv("bitbucket.username", "EISCLI_BITBUCKET_USERNAME")
	_ = viper.BindEnv("bitbucket.app_password", "EISCLI_BITBUCKET_APP_PASSWORD")
	_ = viper.BindEnv("bitbucket.workspace", "EISCLI_BITBUCKET_WORKSPACE")
	_ = viper.BindEnv("bitbucket.client_id", "EISCLI_BITBUCKET_CLIENT_ID")
	_ = viper.BindEnv("bitbucket.client_secret", "EISCLI_BITBUCKET_CLIENT_SECRET")
	_ = viper.BindEnv("bitbucket.use_oauth", "EISCLI_BITBUCKET_USE_OAUTH")

	// Bind AWS environment variables
	_ = viper.BindEnv("aws.default_profile", "AWS_PROFILE")
	_ = viper.BindEnv("aws.region", "AWS_REGION")

	// Try to read config file (optional)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found - create a default one
			if err := createDefaultConfigFile(); err != nil {
				// If creation fails, that's OK - we'll use env vars and build-time defaults
				// Just log it but don't fail
			} else {
				// Try reading again after creating default config
				_ = viper.ReadInConfig()
			}
		} else {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Apply build-time defaults for OAuth credentials if not set in config or env
	// This allows distributing a binary with OAuth consumer credentials baked in
	if config.Bitbucket.ClientID == "" && getBuildTimeClientID != nil {
		config.Bitbucket.ClientID = getBuildTimeClientID()
	}
	if config.Bitbucket.ClientSecret == "" && getBuildTimeClientSecret != nil {
		config.Bitbucket.ClientSecret = getBuildTimeClientSecret()
	}

	// If build-time OAuth defaults are available, default to OAuth mode
	if config.Bitbucket.ClientID != "" && config.Bitbucket.ClientSecret != "" {
		// Only set use_oauth to true if not explicitly configured
		if !viper.IsSet("bitbucket.use_oauth") {
			config.Bitbucket.UseOAuth = true
		}
	}

	// Set defaults for AWS config
	if config.AWS.Region == "" {
		config.AWS.Region = "eu-central-1"
	}
	if config.AWS.DefaultProfile == "" {
		config.AWS.DefaultProfile = "default"
	}
	if config.AWS.NonProdProfile == "" {
		config.AWS.NonProdProfile = "staging"
	}

	globalConfig = &config
	return globalConfig, nil
}

// Build-time OAuth default providers (set by bitbucket package init)
var (
	getBuildTimeClientID     func() string
	getBuildTimeClientSecret func() string
)

// RegisterBuildTimeOAuthDefaults is called by the bitbucket package to register build-time defaults
func RegisterBuildTimeOAuthDefaults(clientIDFunc, clientSecretFunc func() string) {
	getBuildTimeClientID = clientIDFunc
	getBuildTimeClientSecret = clientSecretFunc
}

// createDefaultConfigFile creates a default config file if none exists
func createDefaultConfigFile() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".eiscli")
	configFile := filepath.Join(configDir, "config.yaml")

	// Create .eiscli directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Check if config file already exists
	if _, err := os.Stat(configFile); err == nil {
		// File already exists, don't overwrite
		return nil
	}

	// Create default config content
	defaultConfig := `# EIS CLI Configuration
# This file was automatically created with default settings.
# You can modify these values as needed.

bitbucket:
  # Your Bitbucket workspace (organization slug)
  workspace: "cover42"

# AWS Configuration (optional)
# Uncomment and modify if you need custom AWS profiles
#aws:
#  default_profile: "default"     # For production/testing environments
#  nonprod_profile: "staging"     # For staging/dev environments
#  region: "eu-central-1"

# Deployment Configuration (optional)
# Uncomment and modify if you want to change default behavior
#deployment:
#  auto_create_environments: false
#  default_environment_type: "Test"
`

	// Write config file
	if err := os.WriteFile(configFile, []byte(defaultConfig), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Get returns the global configuration
func Get() *Config {
	if globalConfig == nil {
		// Try to load, but don't fail if it doesn't exist
		_, _ = Load()
	}
	return globalConfig
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Bitbucket.Workspace == "" {
		return fmt.Errorf("bitbucket workspace is required (set EISCLI_BITBUCKET_WORKSPACE or add to config file)")
	}

	// Check if OAuth or Basic Auth credentials are provided
	if c.Bitbucket.UseOAuth {
		if c.Bitbucket.ClientID == "" {
			return fmt.Errorf("bitbucket OAuth client_id is required when use_oauth is true (set EISCLI_BITBUCKET_CLIENT_ID or add to config file)")
		}
		if c.Bitbucket.ClientSecret == "" {
			return fmt.Errorf("bitbucket OAuth client_secret is required when use_oauth is true (set EISCLI_BITBUCKET_CLIENT_SECRET or add to config file)")
		}
	} else {
		if c.Bitbucket.Username == "" {
			return fmt.Errorf("bitbucket username is required (set EISCLI_BITBUCKET_USERNAME or add to config file)")
		}
		if c.Bitbucket.AppPassword == "" {
			return fmt.Errorf("bitbucket app password is required (set EISCLI_BITBUCKET_APP_PASSWORD or add to config file)")
		}
	}

	// Validate deployment config if default environment type is set
	if c.Deployment.DefaultEnvironmentType != "" {
		validTypes := []string{"Test", "Staging", "Production"}
		isValid := false
		for _, validType := range validTypes {
			if c.Deployment.DefaultEnvironmentType == validType {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("invalid default_environment_type: must be one of Test, Staging, or Production")
		}
	}

	return nil
}
