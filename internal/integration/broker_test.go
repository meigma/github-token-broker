//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testAWSRegion             = "us-east-1"
	testClientIDParameter     = "/test/client-id"
	testInstallationParameter = "/test/installation-id"
	testPrivateKeyParameter   = "/test/private-key"
	testClientID              = "Iv1.client"
	testInstallationID        = "123"
	testOwner                 = "acme"
	testRepository            = "widgets"
)

func TestBrokerMintsTokenWithDefaultPermissions(t *testing.T) {
	key := generateTestPrivateKey(t)
	ssmEndpoint := startMotoSSM(t, map[string]string{
		testClientIDParameter:     testClientID,
		testInstallationParameter: testInstallationID,
		testPrivateKeyParameter:   key.privatePEM,
	})
	github := newGitHubStub(t, githubStubConfig{
		PublicKey:      &key.privateKey.PublicKey,
		ClientID:       testClientID,
		InstallationID: testInstallationID,
		Owner:          testOwner,
		Repository:     testRepository,
		Permissions:    map[string]string{"contents": "read"},
		Token:          "ghs_default_permissions",
		ExpiresAt:      "2026-04-22T00:00:00Z",
	})

	result := runBrokerInvocation(t, runConfig{
		SSMEndpoint:    ssmEndpoint,
		GitHubEndpoint: github.URL(),
	})

	response := result.requireResponse(t)
	assert.Equal(t, "ghs_default_permissions", response.Token)
	assert.Equal(t, []string{"acme/widgets"}, response.Repositories)
	assert.Equal(t, map[string]string{"contents": "read"}, response.Permissions)
	assert.Equal(t, time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC), response.ExpiresAt)
	assertNoLogLeak(t, result, key.privatePEM, "ghs_default_permissions")
	github.assertRequests(t, 2)
}

func TestBrokerMintsTokenWithCustomPermissions(t *testing.T) {
	key := generateTestPrivateKey(t)
	ssmEndpoint := startMotoSSM(t, map[string]string{
		testClientIDParameter:     testClientID,
		testInstallationParameter: testInstallationID,
		testPrivateKeyParameter:   key.privatePEM,
	})
	github := newGitHubStub(t, githubStubConfig{
		PublicKey:      &key.privateKey.PublicKey,
		ClientID:       testClientID,
		InstallationID: testInstallationID,
		Owner:          testOwner,
		Repository:     testRepository,
		Permissions: map[string]string{
			"contents":      "read",
			"metadata":      "read",
			"pull_requests": "write",
		},
		Token:     "ghs_custom_permissions",
		ExpiresAt: "2026-04-22T00:00:00Z",
	})

	result := runBrokerInvocation(t, runConfig{
		SSMEndpoint:    ssmEndpoint,
		GitHubEndpoint: github.URL(),
		ExtraEnv: map[string]string{
			"GITHUB_TOKEN_BROKER_PERMISSIONS": `{"contents":"read","metadata":"read","pull_requests":"write"}`,
		},
	})

	response := result.requireResponse(t)
	assert.Equal(t, "ghs_custom_permissions", response.Token)
	assert.Equal(t, map[string]string{
		"contents":      "read",
		"metadata":      "read",
		"pull_requests": "write",
	}, response.Permissions)
	assertNoLogLeak(t, result, key.privatePEM, "ghs_custom_permissions")
	github.assertRequests(t, 2)
}

func TestBrokerReportsMissingSSMParameter(t *testing.T) {
	key := generateTestPrivateKey(t)
	ssmEndpoint := startMotoSSM(t, map[string]string{
		testClientIDParameter:     testClientID,
		testInstallationParameter: testInstallationID,
	})
	github := newGitHubStub(t, githubStubConfig{
		PublicKey:      &key.privateKey.PublicKey,
		ClientID:       testClientID,
		InstallationID: testInstallationID,
		Owner:          testOwner,
		Repository:     testRepository,
		Permissions:    map[string]string{"contents": "read"},
		Token:          "ghs_never_minted",
		ExpiresAt:      "2026-04-22T00:00:00Z",
	})

	result := runBrokerInvocation(t, runConfig{
		SSMEndpoint:    ssmEndpoint,
		GitHubEndpoint: github.URL(),
	})

	body := result.requireError(t)
	assert.Contains(t, body, "missing GitHub App SSM parameters")
	assert.Contains(t, body, testPrivateKeyParameter)
	assertNoLogLeak(t, result, key.privatePEM, "ghs_never_minted")
	github.assertRequests(t, 0)
}

func TestBrokerRejectsRepositoryInstallationMismatch(t *testing.T) {
	key := generateTestPrivateKey(t)
	ssmEndpoint := startMotoSSM(t, map[string]string{
		testClientIDParameter:     testClientID,
		testInstallationParameter: testInstallationID,
		testPrivateKeyParameter:   key.privatePEM,
	})
	github := newGitHubStub(t, githubStubConfig{
		PublicKey:      &key.privateKey.PublicKey,
		ClientID:       testClientID,
		InstallationID: testInstallationID,
		Owner:          testOwner,
		Repository:     testRepository,
		Permissions:    map[string]string{"contents": "read"},
		Token:          "ghs_never_minted",
		ReportedID:     "456",
	})

	result := runBrokerInvocation(t, runConfig{
		SSMEndpoint:    ssmEndpoint,
		GitHubEndpoint: github.URL(),
	})

	body := result.requireError(t)
	assert.Contains(t, body, "does not match configured installation")
	assertNoLogLeak(t, result, key.privatePEM, "ghs_never_minted")
	github.assertRequests(t, 1)
}

func TestBrokerReportsGitHubTokenRejection(t *testing.T) {
	key := generateTestPrivateKey(t)
	ssmEndpoint := startMotoSSM(t, map[string]string{
		testClientIDParameter:     testClientID,
		testInstallationParameter: testInstallationID,
		testPrivateKeyParameter:   key.privatePEM,
	})
	github := newGitHubStub(t, githubStubConfig{
		PublicKey:      &key.privateKey.PublicKey,
		ClientID:       testClientID,
		InstallationID: testInstallationID,
		Owner:          testOwner,
		Repository:     testRepository,
		Permissions:    map[string]string{"contents": "read"},
		Status:         403,
		ErrorBody:      `{"message":"denied"}`,
	})

	result := runBrokerInvocation(t, runConfig{
		SSMEndpoint:    ssmEndpoint,
		GitHubEndpoint: github.URL(),
	})

	body := result.requireError(t)
	assert.Contains(t, body, "status 403")
	assert.NotContains(t, body, "denied")
	assertNoLogLeak(t, result, key.privatePEM, github.lastJWT())
	github.assertRequests(t, 2)
}

func TestBrokerReportsMalformedPrivateKeyBeforeCallingGitHub(t *testing.T) {
	key := generateTestPrivateKey(t)
	malformedKey := "-----BEGIN RSA PRIVATE KEY-----\nbm90LWEta2V5\n-----END RSA PRIVATE KEY-----\n"
	ssmEndpoint := startMotoSSM(t, map[string]string{
		testClientIDParameter:     testClientID,
		testInstallationParameter: testInstallationID,
		testPrivateKeyParameter:   malformedKey,
	})
	github := newGitHubStub(t, githubStubConfig{
		PublicKey:      &key.privateKey.PublicKey,
		ClientID:       testClientID,
		InstallationID: testInstallationID,
		Owner:          testOwner,
		Repository:     testRepository,
		Permissions:    map[string]string{"contents": "read"},
		Token:          "ghs_never_minted",
		ExpiresAt:      "2026-04-22T00:00:00Z",
	})

	result := runBrokerInvocation(t, runConfig{
		SSMEndpoint:    ssmEndpoint,
		GitHubEndpoint: github.URL(),
	})

	body := result.requireError(t)
	assert.Contains(t, body, "parse GitHub App private key")
	assertNoLogLeak(t, result, malformedKey, "ghs_never_minted")
	github.assertRequests(t, 0)
}

type runConfig struct {
	SSMEndpoint    string
	GitHubEndpoint string
	ExtraEnv       map[string]string
}

type brokerRunResult struct {
	runtime runtimeResult
	stdout  string
	stderr  string
}

func runBrokerInvocation(t *testing.T, cfg runConfig) brokerRunResult {
	t.Helper()

	runtime := newRuntimeStub(t)

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, buildHostBootstrap(t))
	cmd.Env = brokerEnv(runtime.Host(), cfg)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	require.NoError(t, cmd.Start(), "failed to start bootstrap binary")

	var stopOnce sync.Once
	stop := func() {
		stopOnce.Do(func() {
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			_ = cmd.Wait()
		})
	}
	t.Cleanup(stop)

	result := runtime.Wait(t, 20*time.Second)
	stop()

	return brokerRunResult{
		runtime: result,
		stdout:  stdout.String(),
		stderr:  stderr.String(),
	}
}

func brokerEnv(runtimeHost string, cfg runConfig) []string {
	env := map[string]string{
		"PATH":                                      os.Getenv("PATH"),
		"AWS_ACCESS_KEY_ID":                         "test",
		"AWS_SECRET_ACCESS_KEY":                     "test",
		"AWS_EC2_METADATA_DISABLED":                 "true",
		"AWS_LAMBDA_RUNTIME_API":                    runtimeHost,
		"AWS_REGION":                                testAWSRegion,
		"AWS_ENDPOINT_URL_SSM":                      cfg.SSMEndpoint,
		"GITHUB_TOKEN_BROKER_CLIENT_ID_PARAM":       testClientIDParameter,
		"GITHUB_TOKEN_BROKER_GITHUB_API_BASE_URL":   cfg.GitHubEndpoint,
		"GITHUB_TOKEN_BROKER_INSTALLATION_ID_PARAM": testInstallationParameter,
		"GITHUB_TOKEN_BROKER_LOG_LEVEL":             "debug",
		"GITHUB_TOKEN_BROKER_PRIVATE_KEY_PARAM":     testPrivateKeyParameter,
		"GITHUB_TOKEN_BROKER_REPOSITORY_NAME":       testRepository,
		"GITHUB_TOKEN_BROKER_REPOSITORY_OWNER":      testOwner,
	}
	for key, value := range cfg.ExtraEnv {
		env[key] = value
	}

	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := make([]string, 0, len(keys))
	for _, key := range keys {
		result = append(result, key+"="+env[key])
	}
	return result
}

func (r brokerRunResult) requireResponse(t *testing.T) brokerResponse {
	t.Helper()

	require.Equal(t, runtimeResultResponse, r.runtime.kind, "stdout: %s\nstderr: %s\nbody: %s", r.stdout, r.stderr, string(r.runtime.body))

	var response brokerResponse
	require.NoError(t, json.Unmarshal(r.runtime.body, &response))
	return response
}

func (r brokerRunResult) requireError(t *testing.T) string {
	t.Helper()

	require.Equal(t, runtimeResultError, r.runtime.kind, "stdout: %s\nstderr: %s\nbody: %s", r.stdout, r.stderr, string(r.runtime.body))
	return string(r.runtime.body)
}

func assertNoLogLeak(t *testing.T, result brokerRunResult, secrets ...string) {
	t.Helper()

	logs := result.stdout + result.stderr
	for _, secret := range secrets {
		if strings.TrimSpace(secret) == "" {
			continue
		}
		assert.NotContains(t, logs, secret)
	}
}

type brokerResponse struct {
	Token        string            `json:"token"`
	ExpiresAt    time.Time         `json:"expires_at"`
	Repositories []string          `json:"repositories"`
	Permissions  map[string]string `json:"permissions"`
}
