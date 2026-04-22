// Package params loads GitHub App secrets from AWS SSM Parameter Store.
package params

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"

	"github.com/meigma/github-token-broker/internal/githubapp"
)

// SSMAPI is the SSM client surface used by Store.
type SSMAPI interface {
	GetParameters(ctx context.Context, params *ssm.GetParametersInput, optFns ...func(*ssm.Options)) (*ssm.GetParametersOutput, error)
}

// Names lists the SSM parameters that hold GitHub App configuration.
type Names struct {
	// ClientID is the SSM parameter name for the GitHub App client ID.
	ClientID string
	// InstallationID is the SSM parameter name for the GitHub App installation ID.
	InstallationID string
	// PrivateKey is the SSM SecureString parameter name for the GitHub App private key.
	PrivateKey string
}

// Store loads GitHub App configuration from SSM Parameter Store.
type Store struct {
	client SSMAPI
	names  Names
}

// NewStore constructs an SSM-backed Store.
func NewStore(client SSMAPI, names Names) *Store {
	return &Store{
		client: client,
		names:  names,
	}
}

// LoadAppConfig reads GitHub App configuration from SSM.
//
// LoadAppConfig batches a single GetParameters call with decryption enabled,
// then returns a typed AppConfig. It surfaces missing parameters as an error
// rather than returning empty values.
func (s *Store) LoadAppConfig(ctx context.Context) (githubapp.AppConfig, error) {
	result, err := s.client.GetParameters(ctx, &ssm.GetParametersInput{
		Names: []string{
			s.names.ClientID,
			s.names.InstallationID,
			s.names.PrivateKey,
		},
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		return githubapp.AppConfig{}, fmt.Errorf("read GitHub App parameters from SSM: %w", err)
	}

	if len(result.InvalidParameters) > 0 {
		return githubapp.AppConfig{}, fmt.Errorf("missing GitHub App SSM parameters: %v", result.InvalidParameters)
	}

	values := make(map[string]string, len(result.Parameters))
	for _, parameter := range result.Parameters {
		if parameter.Name == nil || parameter.Value == nil {
			continue
		}

		values[*parameter.Name] = *parameter.Value
	}

	clientID, err := requiredValue(values, s.names.ClientID)
	if err != nil {
		return githubapp.AppConfig{}, err
	}

	installationID, err := requiredValue(values, s.names.InstallationID)
	if err != nil {
		return githubapp.AppConfig{}, err
	}

	privateKey, err := requiredValue(values, s.names.PrivateKey)
	if err != nil {
		return githubapp.AppConfig{}, err
	}

	return githubapp.AppConfig{
		ClientID:       clientID,
		InstallationID: installationID,
		PrivateKeyPEM:  privateKey,
	}, nil
}

func requiredValue(values map[string]string, name string) (string, error) {
	value := values[name]
	if value == "" {
		return "", fmt.Errorf("SSM parameter %s is empty or missing", name)
	}

	return value, nil
}
