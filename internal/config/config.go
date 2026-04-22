// Package config loads github-token-broker runtime configuration from the
// process environment.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const (
	defaultClientIDParameter       = "/github-token-broker/app/client-id"
	defaultInstallationIDParameter = "/github-token-broker/app/installation-id"
	defaultPrivateKeyParameter     = "/github-token-broker/app/private-key-pem"
	defaultGitHubAPIBaseURL        = "https://api.github.com"
	defaultLogLevel                = "info"
	defaultPermissionName          = "contents"
	defaultPermissionLevel         = "read"
)

// Config is the runtime configuration for github-token-broker.
type Config struct {
	// AWSRegion is the AWS region used for SDK configuration.
	AWSRegion string
	// ClientIDParameter is the SSM parameter that stores the GitHub App client ID.
	ClientIDParameter string
	// InstallationIDParameter is the SSM parameter that stores the GitHub App installation ID.
	InstallationIDParameter string
	// PrivateKeyParameter is the SSM SecureString parameter that stores the GitHub App private key.
	PrivateKeyParameter string
	// GitHubAPIBaseURL is the GitHub API base URL.
	GitHubAPIBaseURL string
	// RepositoryOwner is the GitHub owner for the token request.
	RepositoryOwner string
	// RepositoryName is the GitHub repository for the token request.
	RepositoryName string
	// Permissions is the GitHub permission set requested on each minted installation token.
	Permissions map[string]string
	// LogLevel is the slog level string.
	LogLevel string
}

// Load reads environment variables into a Config.
//
// Load returns an error when required variables are missing, when SSM parameter
// paths are not absolute, or when the permissions environment variable does not
// parse into a non-empty JSON object of string-to-string entries.
func Load() (Config, error) {
	cfg := Config{
		AWSRegion:               os.Getenv("AWS_REGION"),
		ClientIDParameter:       envOrDefault("GITHUB_TOKEN_BROKER_CLIENT_ID_PARAM", defaultClientIDParameter),
		InstallationIDParameter: envOrDefault("GITHUB_TOKEN_BROKER_INSTALLATION_ID_PARAM", defaultInstallationIDParameter),
		PrivateKeyParameter:     envOrDefault("GITHUB_TOKEN_BROKER_PRIVATE_KEY_PARAM", defaultPrivateKeyParameter),
		GitHubAPIBaseURL:        envOrDefault("GITHUB_TOKEN_BROKER_GITHUB_API_BASE_URL", defaultGitHubAPIBaseURL),
		RepositoryOwner:         strings.TrimSpace(os.Getenv("GITHUB_TOKEN_BROKER_REPOSITORY_OWNER")),
		RepositoryName:          strings.TrimSpace(os.Getenv("GITHUB_TOKEN_BROKER_REPOSITORY_NAME")),
		LogLevel:                envOrDefault("GITHUB_TOKEN_BROKER_LOG_LEVEL", defaultLogLevel),
	}

	if cfg.AWSRegion == "" {
		return Config{}, fmt.Errorf("AWS_REGION is required")
	}

	if !strings.HasPrefix(cfg.ClientIDParameter, "/") {
		return Config{}, fmt.Errorf("GITHUB_TOKEN_BROKER_CLIENT_ID_PARAM must be an absolute SSM parameter path")
	}

	if !strings.HasPrefix(cfg.InstallationIDParameter, "/") {
		return Config{}, fmt.Errorf("GITHUB_TOKEN_BROKER_INSTALLATION_ID_PARAM must be an absolute SSM parameter path")
	}

	if !strings.HasPrefix(cfg.PrivateKeyParameter, "/") {
		return Config{}, fmt.Errorf("GITHUB_TOKEN_BROKER_PRIVATE_KEY_PARAM must be an absolute SSM parameter path")
	}

	if cfg.RepositoryOwner == "" {
		return Config{}, fmt.Errorf("GITHUB_TOKEN_BROKER_REPOSITORY_OWNER is required")
	}

	if cfg.RepositoryName == "" {
		return Config{}, fmt.Errorf("GITHUB_TOKEN_BROKER_REPOSITORY_NAME is required")
	}

	if cfg.GitHubAPIBaseURL == "" {
		return Config{}, fmt.Errorf("GITHUB_TOKEN_BROKER_GITHUB_API_BASE_URL must not be empty")
	}

	permissions, err := loadPermissions(os.Getenv("GITHUB_TOKEN_BROKER_PERMISSIONS"))
	if err != nil {
		return Config{}, err
	}
	cfg.Permissions = permissions

	return cfg, nil
}

func loadPermissions(raw string) (map[string]string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return map[string]string{defaultPermissionName: defaultPermissionLevel}, nil
	}

	var permissions map[string]string
	if err := json.Unmarshal([]byte(trimmed), &permissions); err != nil {
		return nil, fmt.Errorf("GITHUB_TOKEN_BROKER_PERMISSIONS must be a JSON object of string-to-string entries: %w", err)
	}

	if len(permissions) == 0 {
		return nil, fmt.Errorf("GITHUB_TOKEN_BROKER_PERMISSIONS must request at least one permission")
	}

	for name, level := range permissions {
		if name == "" || level == "" {
			return nil, fmt.Errorf("GITHUB_TOKEN_BROKER_PERMISSIONS entries must have non-empty keys and values")
		}
	}

	return permissions, nil
}

func envOrDefault(key string, defaultValue string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}

	return defaultValue
}
