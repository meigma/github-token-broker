//go:build integration

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	motoImage = "motoserver/moto:5.1.22"
	motoPort  = "5000/tcp"
)

func startMotoSSM(t *testing.T, values map[string]string) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	container, err := testcontainers.Run(ctx,
		motoImage,
		testcontainers.WithExposedPorts(motoPort),
		testcontainers.WithWaitStrategy(wait.ForHTTP("/moto-api/").WithPort(motoPort).WithStartupTimeout(90*time.Second)),
	)
	require.NoError(t, err)
	testcontainers.CleanupContainer(t, container)

	host, err := container.Host(ctx)
	require.NoError(t, err)

	port, err := container.MappedPort(ctx, motoPort)
	require.NoError(t, err)

	endpoint := fmt.Sprintf("http://%s:%s", host, port.Port())
	seedSSMParameters(t, ctx, endpoint, values)
	return endpoint
}

func seedSSMParameters(t *testing.T, ctx context.Context, endpoint string, values map[string]string) {
	t.Helper()

	client := ssm.NewFromConfig(aws.Config{
		Region:      testAWSRegion,
		Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider("test", "test", "")),
	}, func(options *ssm.Options) {
		options.BaseEndpoint = aws.String(endpoint)
	})

	for name, value := range values {
		_, err := client.PutParameter(ctx, &ssm.PutParameterInput{
			Name:      aws.String(name),
			Value:     aws.String(value),
			Type:      types.ParameterTypeSecureString,
			Overwrite: aws.Bool(true),
		})
		require.NoError(t, err, "seed SSM parameter %s", name)
	}
}
