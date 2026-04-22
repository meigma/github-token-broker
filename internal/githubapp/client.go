// Package githubapp mints short-lived GitHub App installation tokens by
// signing a JWT and exchanging it at the GitHub REST API.
package githubapp

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const jwtLifetime = 9 * time.Minute

// AppConfig contains the GitHub App material needed to mint installation tokens.
type AppConfig struct {
	// ClientID is the GitHub App client ID used as the JWT issuer.
	ClientID string
	// InstallationID is the GitHub App installation ID for the configured target.
	InstallationID string
	// PrivateKeyPEM is the GitHub App private signing key in PEM format.
	PrivateKeyPEM string
}

// Target describes the installation-token request.
type Target struct {
	// Owner is the GitHub repository owner.
	Owner string
	// Repository is the GitHub repository name.
	Repository string
	// Permissions are the installation-token permissions requested from GitHub.
	Permissions map[string]string
}

// InstallationToken is the short-lived token GitHub returns.
type InstallationToken struct {
	// Token is the GitHub installation token.
	Token string
	// ExpiresAt is the expiration timestamp returned by GitHub.
	ExpiresAt time.Time
}

// HTTPDoer performs HTTP requests.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Clock returns the current time.
type Clock func() time.Time

// Client mints GitHub App installation tokens.
type Client struct {
	httpClient HTTPDoer
	baseURL    *url.URL
	clock      Clock
}

// NewClient constructs a Client.
//
// NewClient validates that baseURL is an absolute URL and defaults httpClient
// to http.DefaultClient and clock to time.Now when either is nil.
func NewClient(httpClient HTTPDoer, baseURL string, clock Clock) (*Client, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse GitHub API base URL: %w", err)
	}

	if parsedBaseURL.Scheme == "" || parsedBaseURL.Host == "" {
		return nil, fmt.Errorf("GitHub API base URL must be absolute")
	}

	if clock == nil {
		clock = time.Now
	}

	return &Client{
		httpClient: httpClient,
		baseURL:    parsedBaseURL,
		clock:      clock,
	}, nil
}

// CreateInstallationToken mints an installation token for the given target.
//
// CreateInstallationToken signs a short-lived RS256 JWT, POSTs it to
// /app/installations/{id}/access_tokens, and returns the token and its
// expiration. Errors never include the App private key material.
func (c *Client) CreateInstallationToken(ctx context.Context, app AppConfig, target Target) (InstallationToken, error) {
	if app.ClientID == "" {
		return InstallationToken{}, fmt.Errorf("GitHub App client ID is required")
	}

	if app.InstallationID == "" {
		return InstallationToken{}, fmt.Errorf("GitHub App installation ID is required")
	}

	if target.Owner == "" {
		return InstallationToken{}, fmt.Errorf("GitHub repository owner is required")
	}

	if target.Repository == "" {
		return InstallationToken{}, fmt.Errorf("GitHub repository name is required")
	}

	jwt, err := c.signJWT(app)
	if err != nil {
		return InstallationToken{}, err
	}

	if err := c.validateRepositoryInstallation(ctx, jwt, app.InstallationID, target); err != nil {
		return InstallationToken{}, err
	}

	requestBody := struct {
		Repositories []string          `json:"repositories"`
		Permissions  map[string]string `json:"permissions"`
	}{
		Repositories: []string{target.Repository},
		Permissions:  target.Permissions,
	}

	encodedRequestBody, err := json.Marshal(requestBody)
	if err != nil {
		return InstallationToken{}, fmt.Errorf("encode GitHub token request: %w", err)
	}

	endpoint := c.baseURL.JoinPath("app", "installations", app.InstallationID, "access_tokens")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(encodedRequestBody))
	if err != nil {
		return InstallationToken{}, fmt.Errorf("create GitHub token request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "github-token-broker")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return InstallationToken{}, fmt.Errorf("request GitHub installation token: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return InstallationToken{}, fmt.Errorf("read GitHub installation-token response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return InstallationToken{}, fmt.Errorf("GitHub installation-token request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	var tokenResponse struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	if err := json.Unmarshal(responseBody, &tokenResponse); err != nil {
		return InstallationToken{}, fmt.Errorf("decode GitHub installation-token response: %w", err)
	}

	if tokenResponse.Token == "" {
		return InstallationToken{}, fmt.Errorf("GitHub installation-token response did not include a token")
	}

	if tokenResponse.ExpiresAt.IsZero() {
		return InstallationToken{}, fmt.Errorf("GitHub installation-token response did not include an expiration")
	}

	return InstallationToken{
		Token:     tokenResponse.Token,
		ExpiresAt: tokenResponse.ExpiresAt,
	}, nil
}

func (c *Client) validateRepositoryInstallation(ctx context.Context, jwt string, installationID string, target Target) error {
	endpoint := c.baseURL.JoinPath("repos", target.Owner, target.Repository, "installation")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return fmt.Errorf("create GitHub repository-installation request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("User-Agent", "github-token-broker")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request GitHub repository installation: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("read GitHub repository-installation response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("GitHub repository-installation request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	var installationResponse struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(responseBody, &installationResponse); err != nil {
		return fmt.Errorf("decode GitHub repository-installation response: %w", err)
	}

	if installationResponse.ID == 0 {
		return fmt.Errorf("GitHub repository-installation response did not include an installation ID")
	}

	if strconv.FormatInt(installationResponse.ID, 10) != installationID {
		return fmt.Errorf("GitHub repository installation %d does not match configured installation %s", installationResponse.ID, installationID)
	}

	return nil
}

func (c *Client) signJWT(app AppConfig) (string, error) {
	privateKey, err := parsePrivateKey(app.PrivateKeyPEM)
	if err != nil {
		return "", err
	}

	now := c.clock().UTC()
	header := map[string]string{
		"alg": "RS256",
		"typ": "JWT",
	}
	claims := map[string]any{
		"iat": now.Add(-60 * time.Second).Unix(),
		"exp": now.Add(jwtLifetime).Unix(),
		"iss": app.ClientID,
	}

	encodedHeader, err := encodeJWTPart(header)
	if err != nil {
		return "", fmt.Errorf("encode JWT header: %w", err)
	}

	encodedClaims, err := encodeJWTPart(claims)
	if err != nil {
		return "", fmt.Errorf("encode JWT claims: %w", err)
	}

	signingInput := encodedHeader + "." + encodedClaims
	digest := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, digest[:])
	if err != nil {
		return "", fmt.Errorf("sign GitHub App JWT: %w", err)
	}

	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func encodeJWTPart(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(encoded), nil
}

func parsePrivateKey(privateKeyPEM string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("decode GitHub App private key PEM")
	}

	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	parsedKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse GitHub App private key")
	}

	privateKey, ok := parsedKey.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("GitHub App private key must be RSA")
	}

	return privateKey, nil
}
