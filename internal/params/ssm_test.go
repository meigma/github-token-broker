package params

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadAppConfig(t *testing.T) {
	client := &fakeSSM{
		output: &ssm.GetParametersOutput{
			Parameters: []types.Parameter{
				{Name: aws.String("/client-id"), Value: aws.String("Iv1.client")},
				{Name: aws.String("/installation-id"), Value: aws.String("123")},
				{Name: aws.String("/private-key"), Value: aws.String("private-key")},
			},
		},
	}
	store := NewStore(client, Names{
		ClientID:       "/client-id",
		InstallationID: "/installation-id",
		PrivateKey:     "/private-key",
	})

	cfg, err := store.LoadAppConfig(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []string{"/client-id", "/installation-id", "/private-key"}, client.input.Names)
	assert.True(t, aws.ToBool(client.input.WithDecryption))
	assert.Equal(t, "Iv1.client", cfg.ClientID)
	assert.Equal(t, "123", cfg.InstallationID)
	assert.Equal(t, "private-key", cfg.PrivateKeyPEM)
}

func TestLoadAppConfigRejectsInvalidParameters(t *testing.T) {
	store := NewStore(&fakeSSM{
		output: &ssm.GetParametersOutput{
			InvalidParameters: []string{"/private-key"},
		},
	}, Names{
		ClientID:       "/client-id",
		InstallationID: "/installation-id",
		PrivateKey:     "/private-key",
	})

	_, err := store.LoadAppConfig(context.Background())

	require.Error(t, err)
	assert.ErrorContains(t, err, "missing GitHub App SSM parameters")
}

func TestLoadAppConfigWrapsSSMErrors(t *testing.T) {
	store := NewStore(&fakeSSM{
		err: errors.New("boom"),
	}, Names{
		ClientID:       "/client-id",
		InstallationID: "/installation-id",
		PrivateKey:     "/private-key",
	})

	_, err := store.LoadAppConfig(context.Background())

	require.Error(t, err)
	assert.ErrorContains(t, err, "read GitHub App parameters from SSM")
}

type fakeSSM struct {
	input  *ssm.GetParametersInput
	output *ssm.GetParametersOutput
	err    error
}

func (f *fakeSSM) GetParameters(_ context.Context, input *ssm.GetParametersInput, _ ...func(*ssm.Options)) (*ssm.GetParametersOutput, error) {
	f.input = input
	return f.output, f.err
}
