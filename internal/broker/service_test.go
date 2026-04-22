package broker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/meigma/github-token-broker/internal/githubapp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMint(t *testing.T) {
	expiresAt := time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC)
	source := &fakeAppConfigSource{
		cfg: githubapp.AppConfig{
			ClientID:       "Iv1.client",
			InstallationID: "123",
			PrivateKeyPEM:  "private-key",
		},
	}
	issuer := &fakeTokenIssuer{
		token: githubapp.InstallationToken{
			Token:     "ghs_test",
			ExpiresAt: expiresAt,
		},
	}
	service := NewService(source, issuer, githubapp.Target{
		Owner:      "acme",
		Repository: "widgets",
		Permissions: map[string]string{
			"contents": "read",
		},
	})

	response, err := service.Mint(context.Background())

	require.NoError(t, err)
	assert.Equal(t, source.cfg, issuer.app)
	assert.Equal(t, "ghs_test", response.Token)
	assert.Equal(t, expiresAt, response.ExpiresAt)
	assert.Equal(t, []string{"acme/widgets"}, response.Repositories)
	assert.Equal(t, map[string]string{"contents": "read"}, response.Permissions)
}

func TestMintPropagatesAppConfigErrors(t *testing.T) {
	source := &fakeAppConfigSource{err: errors.New("ssm exploded")}
	issuer := &fakeTokenIssuer{}
	service := NewService(source, issuer, githubapp.Target{
		Owner:      "acme",
		Repository: "widgets",
	})

	_, err := service.Mint(context.Background())

	require.Error(t, err)
	assert.ErrorContains(t, err, "ssm exploded")
}

func TestMintPropagatesTokenIssuerErrors(t *testing.T) {
	source := &fakeAppConfigSource{
		cfg: githubapp.AppConfig{
			ClientID:       "Iv1.client",
			InstallationID: "123",
			PrivateKeyPEM:  "private-key",
		},
	}
	issuer := &fakeTokenIssuer{err: errors.New("github denied the JWT")}
	service := NewService(source, issuer, githubapp.Target{
		Owner:      "acme",
		Repository: "widgets",
	})

	_, err := service.Mint(context.Background())

	require.Error(t, err)
	assert.ErrorContains(t, err, "github denied the JWT")
}

type fakeAppConfigSource struct {
	cfg githubapp.AppConfig
	err error
}

func (f *fakeAppConfigSource) LoadAppConfig(context.Context) (githubapp.AppConfig, error) {
	if f.err != nil {
		return githubapp.AppConfig{}, f.err
	}
	return f.cfg, nil
}

type fakeTokenIssuer struct {
	app   githubapp.AppConfig
	token githubapp.InstallationToken
	err   error
}

func (f *fakeTokenIssuer) CreateInstallationToken(_ context.Context, app githubapp.AppConfig, _ githubapp.Target) (githubapp.InstallationToken, error) {
	f.app = app
	if f.err != nil {
		return githubapp.InstallationToken{}, f.err
	}
	return f.token, nil
}
