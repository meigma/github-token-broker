package githubapp

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateInstallationToken(t *testing.T) {
	now := time.Date(2026, 4, 21, 23, 0, 0, 0, time.UTC)
	var requests []string
	httpClient := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests = append(requests, req.Method+" "+req.URL.Path)
		assert.Equal(t, "application/vnd.github+json", req.Header.Get("Accept"))
		assert.Equal(t, "github-token-broker", req.Header.Get("User-Agent"))

		authorization := req.Header.Get("Authorization")
		require.True(t, strings.HasPrefix(authorization, "Bearer "))
		assertJWTClaims(t, strings.TrimPrefix(authorization, "Bearer "), map[string]any{
			"iss": "Iv1.client",
			"iat": float64(now.Add(-60 * time.Second).Unix()),
			"exp": float64(now.Add(jwtLifetime).Unix()),
		})

		switch req.URL.String() {
		case "https://api.github.test/repos/acme/widgets/installation":
			assert.Equal(t, http.MethodGet, req.Method)
			return jsonResponse(http.StatusOK, `{"id":123}`), nil
		case "https://api.github.test/app/installations/123/access_tokens":
			assert.Equal(t, http.MethodPost, req.Method)
			assert.Equal(t, "application/json", req.Header.Get("Content-Type"))

			var body struct {
				Repositories []string          `json:"repositories"`
				Permissions  map[string]string `json:"permissions"`
			}
			require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
			assert.Equal(t, []string{"widgets"}, body.Repositories)
			assert.Equal(t, map[string]string{"contents": "read"}, body.Permissions)

			return jsonResponse(http.StatusCreated, `{"token":"ghs_test","expires_at":"2026-04-22T00:00:00Z"}`), nil
		default:
			t.Fatalf("unexpected GitHub request: %s %s", req.Method, req.URL.String())
			return nil, nil
		}
	})
	client, err := NewClient(httpClient, "https://api.github.test", func() time.Time { return now })
	require.NoError(t, err)

	token, err := client.CreateInstallationToken(context.Background(), AppConfig{
		ClientID:       "Iv1.client",
		InstallationID: "123",
		PrivateKeyPEM:  testPrivateKeyPEM(t),
	}, Target{
		Owner:      "acme",
		Repository: "widgets",
		Permissions: map[string]string{
			"contents": "read",
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "ghs_test", token.Token)
	assert.Equal(t, time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC), token.ExpiresAt)
	assert.Equal(t, []string{
		"GET /repos/acme/widgets/installation",
		"POST /app/installations/123/access_tokens",
	}, requests)
}

func TestCreateInstallationTokenAcceptsPKCS8Key(t *testing.T) {
	privateKey := testPrivateKeyPKCS8PEM(t)
	httpClient := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodGet {
			return jsonResponse(http.StatusOK, `{"id":123}`), nil
		}
		return jsonResponse(http.StatusCreated, `{"token":"ghs_pkcs8","expires_at":"2026-04-22T00:00:00Z"}`), nil
	})
	client, err := NewClient(httpClient, "https://api.github.test", nil)
	require.NoError(t, err)

	token, err := client.CreateInstallationToken(context.Background(), AppConfig{
		ClientID:       "Iv1.client",
		InstallationID: "123",
		PrivateKeyPEM:  privateKey,
	}, Target{
		Owner:      "acme",
		Repository: "widgets",
		Permissions: map[string]string{
			"contents": "read",
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "ghs_pkcs8", token.Token)
}

func TestCreateInstallationTokenRejectsInstallationMismatch(t *testing.T) {
	privateKey := testPrivateKeyPEM(t)
	var tokenRequestIssued bool
	httpClient := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodPost {
			tokenRequestIssued = true
		}
		return jsonResponse(http.StatusOK, `{"id":456}`), nil
	})
	client, err := NewClient(httpClient, "https://api.github.test", nil)
	require.NoError(t, err)

	_, err = client.CreateInstallationToken(context.Background(), AppConfig{
		ClientID:       "Iv1.client",
		InstallationID: "123",
		PrivateKeyPEM:  privateKey,
	}, Target{
		Owner:      "acme",
		Repository: "widgets",
		Permissions: map[string]string{
			"contents": "read",
		},
	})

	require.Error(t, err)
	assert.ErrorContains(t, err, "does not match configured installation 123")
	assert.False(t, tokenRequestIssued, "token request should not be issued after an installation mismatch")
	assert.NotContains(t, err.Error(), privateKey)
}

func TestCreateInstallationTokenSurfacesGitHubErrorsWithoutPrivateKey(t *testing.T) {
	privateKey := testPrivateKeyPEM(t)
	httpClient := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodGet {
			return jsonResponse(http.StatusOK, `{"id":123}`), nil
		}
		return jsonResponse(http.StatusForbidden, `{"message":"bad credentials","jwt":"`+privateKey+`"}`), nil
	})
	client, err := NewClient(httpClient, "https://api.github.test", nil)
	require.NoError(t, err)

	_, err = client.CreateInstallationToken(context.Background(), AppConfig{
		ClientID:       "Iv1.client",
		InstallationID: "123",
		PrivateKeyPEM:  privateKey,
	}, Target{
		Owner:      "acme",
		Repository: "widgets",
		Permissions: map[string]string{
			"contents": "read",
		},
	})

	require.Error(t, err)
	assert.ErrorContains(t, err, "status 403")
	assert.NotContains(t, err.Error(), "bad credentials")
	assert.NotContains(t, err.Error(), privateKey)
}

func TestCreateInstallationTokenRejectsUnsafeTarget(t *testing.T) {
	privateKey := testPrivateKeyPEM(t)
	httpClient := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatalf("HTTP call should not be issued for an unsafe target")
		return nil, nil
	})
	client, err := NewClient(httpClient, "https://api.github.test", nil)
	require.NoError(t, err)

	tests := []struct {
		name    string
		target  Target
		wantErr string
	}{
		{
			name: "owner with slash",
			target: Target{
				Owner:      "acme/other",
				Repository: "widgets",
			},
			wantErr: "GitHub repository owner contains unsupported characters",
		},
		{
			name: "repository with percent escape",
			target: Target{
				Owner:      "acme",
				Repository: "widgets%2fother",
			},
			wantErr: "GitHub repository name contains unsupported characters",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.CreateInstallationToken(context.Background(), AppConfig{
				ClientID:       "Iv1.client",
				InstallationID: "123",
				PrivateKeyPEM:  privateKey,
			}, tt.target)

			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErr)
			assert.NotContains(t, err.Error(), privateKey)
		})
	}
}

func TestCreateInstallationTokenRejectsInvalidPrivateKey(t *testing.T) {
	httpClient := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatalf("HTTP call should not be issued for a malformed key")
		return nil, nil
	})
	client, err := NewClient(httpClient, "https://api.github.test", nil)
	require.NoError(t, err)

	garbage := "-----BEGIN RSA PRIVATE KEY-----\nbm90LWEta2V5\n-----END RSA PRIVATE KEY-----\n"
	_, err = client.CreateInstallationToken(context.Background(), AppConfig{
		ClientID:       "Iv1.client",
		InstallationID: "123",
		PrivateKeyPEM:  garbage,
	}, Target{
		Owner:      "acme",
		Repository: "widgets",
	})

	require.Error(t, err)
	assert.NotContains(t, err.Error(), garbage)
}

func TestNewClientRejectsRelativeBaseURL(t *testing.T) {
	_, err := NewClient(nil, "/relative", nil)

	require.Error(t, err)
	assert.ErrorContains(t, err, "absolute")
}

func TestNewClientValidatesBaseURLScheme(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		wantErr bool
	}{
		{name: "github https", baseURL: "https://api.github.com"},
		{name: "ghes https", baseURL: "https://ghe.example.com/api/v3"},
		{name: "localhost http", baseURL: "http://localhost:8080"},
		{name: "ipv4 loopback http", baseURL: "http://127.0.0.1:8080"},
		{name: "ipv6 loopback http", baseURL: "http://[::1]:8080"},
		{name: "non-loopback http", baseURL: "http://api.github.test", wantErr: true},
		{name: "unsupported scheme", baseURL: "ftp://api.github.test", wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewClient(nil, tt.baseURL, nil)

			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorContains(t, err, "must use https")
				return
			}
			require.NoError(t, err)
		})
	}
}

func assertJWTClaims(t *testing.T, token string, expected map[string]any) {
	t.Helper()

	parts := strings.Split(token, ".")
	require.Len(t, parts, 3)

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)

	var claims map[string]any
	require.NoError(t, json.Unmarshal(payload, &claims))

	for key, value := range expected {
		assert.Equal(t, value, claims[key])
	}
}

func testPrivateKeyPEM(t *testing.T) string {
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

func testPrivateKeyPKCS8PEM(t *testing.T) string {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	pkcs8, err := x509.MarshalPKCS8PrivateKey(privateKey)
	require.NoError(t, err)

	encoded := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: pkcs8,
	})
	require.NotNil(t, encoded)

	return string(encoded)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) Do(req *http.Request) (*http.Response, error) {
	return f(req)
}
