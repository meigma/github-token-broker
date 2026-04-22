// Package broker coordinates loading GitHub App configuration from the
// parameter store and minting short-lived installation tokens.
package broker

import (
	"context"
	"time"

	"github.com/meigma/github-token-broker/internal/githubapp"
)

// AppConfigSource loads GitHub App configuration.
type AppConfigSource interface {
	LoadAppConfig(ctx context.Context) (githubapp.AppConfig, error)
}

// TokenIssuer mints GitHub installation tokens.
type TokenIssuer interface {
	CreateInstallationToken(ctx context.Context, app githubapp.AppConfig, target githubapp.Target) (githubapp.InstallationToken, error)
}

// Response is the Lambda response returned to callers.
type Response struct {
	// Token is the GitHub installation token.
	Token string `json:"token"`
	// ExpiresAt is the GitHub installation token expiration timestamp.
	ExpiresAt time.Time `json:"expires_at"`
	// Repositories lists the repositories covered by the token.
	Repositories []string `json:"repositories"`
	// Permissions summarizes the GitHub permissions requested for the token.
	Permissions map[string]string `json:"permissions"`
}

// Service coordinates parameter loading and GitHub token minting.
type Service struct {
	appConfigSource AppConfigSource
	tokenIssuer     TokenIssuer
	target          githubapp.Target
}

// NewService constructs a Service.
func NewService(appConfigSource AppConfigSource, tokenIssuer TokenIssuer, target githubapp.Target) *Service {
	return &Service{
		appConfigSource: appConfigSource,
		tokenIssuer:     tokenIssuer,
		target:          target,
	}
}

// Mint creates a short-lived installation token for the configured target.
//
// Mint loads GitHub App credentials from the configured source, requests an
// installation token from the issuer, and returns the token alongside the
// repository list and permission set requested.
func (s *Service) Mint(ctx context.Context) (Response, error) {
	appConfig, err := s.appConfigSource.LoadAppConfig(ctx)
	if err != nil {
		return Response{}, err
	}

	token, err := s.tokenIssuer.CreateInstallationToken(ctx, appConfig, s.target)
	if err != nil {
		return Response{}, err
	}

	return Response{
		Token:        token.Token,
		ExpiresAt:    token.ExpiresAt,
		Repositories: []string{s.target.Owner + "/" + s.target.Repository},
		Permissions:  clonePermissions(s.target.Permissions),
	}, nil
}

func clonePermissions(permissions map[string]string) map[string]string {
	cloned := make(map[string]string, len(permissions))
	for key, value := range permissions {
		cloned[key] = value
	}

	return cloned
}
