//go:build integration

package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPackagedLambdaRespondsToEmptyInvocation proves the end-to-end wiring:
// the main binary loads config, resolves SSM parameters, signs a JWT, exchanges
// it for an installation token, and returns a well-formed response to the
// Lambda Runtime API. All external dependencies are replaced with in-process
// stubs.
//
// The test builds a host-native binary from cmd/github-token-broker so it can
// be exec'd directly on any platform supported by `go build`. The release zip
// is a separate artifact (cross-compiled to linux/arm64 for Lambda) whose
// format is trivial; its business logic is identical to the host build
// exercised here.
func TestPackagedLambdaRespondsToEmptyInvocation(t *testing.T) {
	bootstrap := buildHostBootstrap(t)

	privateKey := generateTestPrivateKeyPEM(t)

	runtimeStub := newRuntimeStub(t)
	t.Cleanup(runtimeStub.Close)

	ssmStub := newSSMStub(t, map[string]string{
		"/test/client-id":       "Iv1.client",
		"/test/installation-id": "123",
		"/test/private-key":     privateKey,
	})
	t.Cleanup(ssmStub.Close)

	githubStub := newGitHubStub(t, "ghs_integration", "2026-04-22T00:00:00Z")
	t.Cleanup(githubStub.Close)

	runtimeHost := strings.TrimPrefix(runtimeStub.server.URL, "http://")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, bootstrap)
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"AWS_LAMBDA_RUNTIME_API=" + runtimeHost,
		"AWS_REGION=us-east-1",
		"AWS_ACCESS_KEY_ID=test",
		"AWS_SECRET_ACCESS_KEY=test",
		"AWS_ENDPOINT_URL_SSM=" + ssmStub.server.URL,
		"GITHUB_TOKEN_BROKER_GITHUB_API_BASE_URL=" + githubStub.server.URL,
		"GITHUB_TOKEN_BROKER_REPOSITORY_OWNER=acme",
		"GITHUB_TOKEN_BROKER_REPOSITORY_NAME=widgets",
		"GITHUB_TOKEN_BROKER_CLIENT_ID_PARAM=/test/client-id",
		"GITHUB_TOKEN_BROKER_INSTALLATION_ID_PARAM=/test/installation-id",
		"GITHUB_TOKEN_BROKER_PRIVATE_KEY_PARAM=/test/private-key",
		"GITHUB_TOKEN_BROKER_LOG_LEVEL=debug",
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	require.NoError(t, cmd.Start(), "failed to start bootstrap binary")
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	})

	select {
	case response := <-runtimeStub.response:
		assert.Contains(t, string(response), `"token":"ghs_integration"`, "response body should carry the minted token")
		assert.Contains(t, string(response), `"acme/widgets"`, "response should report the configured target repository")
		assert.Contains(t, string(response), `"contents":"read"`, "response should echo the default permission set")

		logs := stdout.String()
		assert.NotContains(t, logs, "ghs_integration", "stdout logs must not contain the minted token")

	case <-time.After(15 * time.Second):
		t.Fatalf("timed out waiting for runtime response\nstdout: %s\nstderr: %s", stdout.String(), stderr.String())
	}
}

// buildHostBootstrap compiles the main binary for the host platform and
// returns the path to the resulting executable. The Lambda release build is
// cross-compiled to linux/arm64, which cannot be exec'd on typical dev or CI
// hosts — this helper sidesteps that by producing a host-native binary that
// shares the same code paths.
func buildHostBootstrap(t *testing.T) string {
	t.Helper()

	out := filepath.Join(t.TempDir(), "bootstrap")
	cmd := exec.Command("go", "build", "-tags", "lambda.norpc", "-o", out, "./cmd/github-token-broker")

	repoRoot := findRepoRoot(t)
	cmd.Dir = repoRoot

	combined, err := cmd.CombinedOutput()
	require.NoError(t, err, "go build failed: %s", string(combined))

	info, err := os.Stat(out)
	require.NoError(t, err)
	require.Falsef(t, info.IsDir(), "expected bootstrap file, got directory: %s", out)
	return out
}

func findRepoRoot(t *testing.T) string {
	t.Helper()

	cwd, err := os.Getwd()
	require.NoError(t, err)

	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find go.mod walking up from %s", cwd)
		}
		dir = parent
	}
}

func generateTestPrivateKeyPEM(t *testing.T) string {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	encoded := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	require.NotNil(t, encoded)
	return string(encoded)
}

// runtimeStub serves the subset of the AWS Lambda Runtime API that a single
// invocation needs: one `next` long-poll returning an empty payload, then
// capturing the `response` body for the test to assert against.
type runtimeStub struct {
	server   *httptest.Server
	response chan []byte

	mu        sync.Mutex
	served    bool
	requestID string
}

func newRuntimeStub(t *testing.T) *runtimeStub {
	t.Helper()

	stub := &runtimeStub{
		response:  make(chan []byte, 1),
		requestID: "smoke-test-request",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/2018-06-01/runtime/invocation/next", stub.handleNext)
	mux.HandleFunc("/2018-06-01/runtime/invocation/", stub.handleInvocation)
	mux.HandleFunc("/2018-06-01/runtime/init/error", stub.handleInitError)

	stub.server = httptest.NewServer(mux)
	return stub
}

func (s *runtimeStub) Close() { s.server.Close() }

func (s *runtimeStub) handleNext(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	alreadyServed := s.served
	s.served = true
	s.mu.Unlock()

	if alreadyServed {
		// Subsequent polls hang until the connection is torn down. Lambda's
		// runtime client will loop forever otherwise.
		<-r.Context().Done()
		return
	}

	w.Header().Set("Lambda-Runtime-Aws-Request-Id", s.requestID)
	w.Header().Set("Lambda-Runtime-Deadline-Ms", fmt.Sprintf("%d", time.Now().Add(30*time.Second).UnixMilli()))
	w.Header().Set("Lambda-Runtime-Invoked-Function-Arn", "arn:aws:lambda:us-east-1:000000000000:function:smoke")
	w.Header().Set("Lambda-Runtime-Trace-Id", "Root=smoke")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("null"))
}

func (s *runtimeStub) handleInvocation(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/2018-06-01/runtime/invocation/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	switch parts[1] {
	case "response":
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		select {
		case s.response <- body:
		default:
		}
		w.WriteHeader(http.StatusAccepted)
	case "error":
		body, _ := io.ReadAll(r.Body)
		select {
		case s.response <- append([]byte("ERROR:"), body...):
		default:
		}
		w.WriteHeader(http.StatusAccepted)
	default:
		http.Error(w, "not found", http.StatusNotFound)
	}
}

func (s *runtimeStub) handleInitError(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	select {
	case s.response <- append([]byte("INIT-ERROR:"), body...):
	default:
	}
	w.WriteHeader(http.StatusAccepted)
}

// ssmStub serves the AWS SSM JSON-RPC GetParameters call used by the broker.
type ssmStub struct {
	server *httptest.Server
	values map[string]string
}

func newSSMStub(t *testing.T, values map[string]string) *ssmStub {
	t.Helper()

	stub := &ssmStub{values: values}
	stub.server = httptest.NewServer(http.HandlerFunc(stub.handle))
	return stub
}

func (s *ssmStub) Close() { s.server.Close() }

func (s *ssmStub) handle(w http.ResponseWriter, r *http.Request) {
	target := r.Header.Get("X-Amz-Target")
	if target != "AmazonSSM.GetParameters" {
		http.Error(w, "unsupported target: "+target, http.StatusBadRequest)
		return
	}

	var request struct {
		Names          []string `json:"Names"`
		WithDecryption bool     `json:"WithDecryption"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	type parameter struct {
		Name  string `json:"Name"`
		Type  string `json:"Type"`
		Value string `json:"Value"`
	}
	response := struct {
		Parameters        []parameter `json:"Parameters"`
		InvalidParameters []string    `json:"InvalidParameters"`
	}{
		Parameters:        make([]parameter, 0, len(request.Names)),
		InvalidParameters: []string{},
	}

	for _, name := range request.Names {
		value, ok := s.values[name]
		if !ok {
			response.InvalidParameters = append(response.InvalidParameters, name)
			continue
		}
		response.Parameters = append(response.Parameters, parameter{
			Name:  name,
			Type:  "SecureString",
			Value: value,
		})
	}

	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

// githubStub serves the GitHub App installation-token endpoint.
type githubStub struct {
	server    *httptest.Server
	token     string
	expiresAt string
}

func newGitHubStub(t *testing.T, token, expiresAt string) *githubStub {
	t.Helper()

	stub := &githubStub{token: token, expiresAt: expiresAt}
	stub.server = httptest.NewServer(http.HandlerFunc(stub.handle))
	return stub
}

func (s *githubStub) Close() { s.server.Close() }

func (s *githubStub) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/access_tokens") {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = fmt.Fprintf(w, `{"token":%q,"expires_at":%q}`, s.token, s.expiresAt)
}
