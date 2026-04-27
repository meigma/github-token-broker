package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("AWS_REGION", "us-west-2")
	t.Setenv("GITHUB_TOKEN_BROKER_REPOSITORY_OWNER", "acme")
	t.Setenv("GITHUB_TOKEN_BROKER_REPOSITORY_NAME", "widgets")
}

func TestLoadDefaults(t *testing.T) {
	setRequiredEnv(t)

	cfg, err := Load()

	require.NoError(t, err)
	assert.Equal(t, "us-west-2", cfg.AWSRegion)
	assert.Equal(t, defaultClientIDParameter, cfg.ClientIDParameter)
	assert.Equal(t, defaultInstallationIDParameter, cfg.InstallationIDParameter)
	assert.Equal(t, defaultPrivateKeyParameter, cfg.PrivateKeyParameter)
	assert.Equal(t, defaultGitHubAPIBaseURL, cfg.GitHubAPIBaseURL)
	assert.Equal(t, "acme", cfg.RepositoryOwner)
	assert.Equal(t, "widgets", cfg.RepositoryName)
	assert.Equal(t, map[string]string{"contents": "read"}, cfg.Permissions)
	assert.Equal(t, defaultLogLevel, cfg.LogLevel)
}

func TestLoadRejectsMissingRegion(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("AWS_REGION", "")

	_, err := Load()

	require.Error(t, err)
	assert.ErrorContains(t, err, "AWS_REGION is required")
}

func TestLoadRejectsMissingOwner(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("GITHUB_TOKEN_BROKER_REPOSITORY_OWNER", "")

	_, err := Load()

	require.Error(t, err)
	assert.ErrorContains(t, err, "GITHUB_TOKEN_BROKER_REPOSITORY_OWNER is required")
}

func TestLoadRejectsMissingRepository(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("GITHUB_TOKEN_BROKER_REPOSITORY_NAME", "")

	_, err := Load()

	require.Error(t, err)
	assert.ErrorContains(t, err, "GITHUB_TOKEN_BROKER_REPOSITORY_NAME is required")
}

func TestLoadAcceptsArbitraryOwnerRepo(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("GITHUB_TOKEN_BROKER_REPOSITORY_OWNER", "example-org")
	t.Setenv("GITHUB_TOKEN_BROKER_REPOSITORY_NAME", "infra")

	cfg, err := Load()

	require.NoError(t, err)
	assert.Equal(t, "example-org", cfg.RepositoryOwner)
	assert.Equal(t, "infra", cfg.RepositoryName)
}

func TestLoadRejectsUnsafeOwnerRepo(t *testing.T) {
	tests := []struct {
		name    string
		envKey  string
		envVar  string
		wantMsg string
	}{
		{
			name:    "owner path escape",
			envKey:  "GITHUB_TOKEN_BROKER_REPOSITORY_OWNER",
			envVar:  "acme/widgets",
			wantMsg: "GITHUB_TOKEN_BROKER_REPOSITORY_OWNER contains unsupported characters",
		},
		{
			name:    "repository percent escape",
			envKey:  "GITHUB_TOKEN_BROKER_REPOSITORY_NAME",
			envVar:  "widgets%2fadmin",
			wantMsg: "GITHUB_TOKEN_BROKER_REPOSITORY_NAME contains unsupported characters",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			setRequiredEnv(t)
			t.Setenv(tt.envKey, tt.envVar)

			_, err := Load()

			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantMsg)
		})
	}
}

func TestLoadRejectsInvalidParameterPath(t *testing.T) {
	tests := []struct {
		name    string
		envVar  string
		envKey  string
		wantMsg string
	}{
		{
			name:    "client id",
			envVar:  "github-token-broker/app/client-id",
			envKey:  "GITHUB_TOKEN_BROKER_CLIENT_ID_PARAM",
			wantMsg: "GITHUB_TOKEN_BROKER_CLIENT_ID_PARAM must be an absolute literal SSM parameter path",
		},
		{
			name:    "installation id",
			envVar:  "/github-token-broker/app/*",
			envKey:  "GITHUB_TOKEN_BROKER_INSTALLATION_ID_PARAM",
			wantMsg: "GITHUB_TOKEN_BROKER_INSTALLATION_ID_PARAM must be an absolute literal SSM parameter path",
		},
		{
			name:    "private key",
			envVar:  "/github-token-broker/app/private key",
			envKey:  "GITHUB_TOKEN_BROKER_PRIVATE_KEY_PARAM",
			wantMsg: "GITHUB_TOKEN_BROKER_PRIVATE_KEY_PARAM must be an absolute literal SSM parameter path",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			setRequiredEnv(t)
			t.Setenv(tt.envKey, tt.envVar)

			_, err := Load()

			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantMsg)
		})
	}
}

func TestLoadParsesPermissionsJSON(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("GITHUB_TOKEN_BROKER_PERMISSIONS", `{"contents":"read","metadata":"read","pull_requests":"write"}`)

	cfg, err := Load()

	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		"contents":      "read",
		"metadata":      "read",
		"pull_requests": "write",
	}, cfg.Permissions)
}

func TestLoadRejectsInvalidPermissionsJSON(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{"not json", "not-json"},
		{"array instead of object", "[]"},
		{"string instead of object", `"contents:read"`},
		{"non-string value", `{"contents": 1}`},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			setRequiredEnv(t)
			t.Setenv("GITHUB_TOKEN_BROKER_PERMISSIONS", tt.raw)

			_, err := Load()

			require.Error(t, err)
			assert.ErrorContains(t, err, "GITHUB_TOKEN_BROKER_PERMISSIONS must be a JSON object of string-to-string entries")
		})
	}
}

func TestLoadRejectsEmptyPermissions(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("GITHUB_TOKEN_BROKER_PERMISSIONS", "{}")

	_, err := Load()

	require.Error(t, err)
	assert.ErrorContains(t, err, "GITHUB_TOKEN_BROKER_PERMISSIONS must request at least one permission")
}

func TestLoadRejectsBlankPermissionEntry(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{"empty key", `{"":"read"}`},
		{"empty value", `{"contents":""}`},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			setRequiredEnv(t)
			t.Setenv("GITHUB_TOKEN_BROKER_PERMISSIONS", tt.raw)

			_, err := Load()

			require.Error(t, err)
			assert.ErrorContains(t, err, "GITHUB_TOKEN_BROKER_PERMISSIONS entries must have non-empty keys and values")
		})
	}
}
