//go:build integration

package integration

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type githubStubConfig struct {
	PublicKey      *rsa.PublicKey
	ClientID       string
	InstallationID string
	Owner          string
	Repository     string
	Permissions    map[string]string
	Token          string
	ExpiresAt      string
	Status         int
	ErrorBody      string
	ReportedID     string
}

type githubStub struct {
	server *httptest.Server
	cfg    githubStubConfig

	mu      sync.Mutex
	calls   int
	errors  []error
	seenJWT string
}

func newGitHubStub(t *testing.T, cfg githubStubConfig) *githubStub {
	t.Helper()

	if cfg.Status == 0 {
		cfg.Status = http.StatusCreated
	}
	if cfg.ExpiresAt == "" {
		cfg.ExpiresAt = "2026-04-22T00:00:00Z"
	}
	if cfg.ReportedID == "" {
		cfg.ReportedID = cfg.InstallationID
	}

	stub := &githubStub{cfg: cfg}
	stub.server = httptest.NewServer(http.HandlerFunc(stub.handle))
	t.Cleanup(stub.server.Close)
	return stub
}

func (s *githubStub) URL() string {
	return s.server.URL
}

func (s *githubStub) handle(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.calls++
	s.mu.Unlock()

	if err := s.validateCommon(r); err != nil {
		s.recordError(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	switch r.Method + " " + r.URL.Path {
	case http.MethodGet + " " + s.repositoryInstallationPath():
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{"id":%s}`, s.cfg.ReportedID)
		return
	case http.MethodPost + " " + s.accessTokenPath():
		if err := s.validateAccessTokenRequest(r); err != nil {
			s.recordError(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	default:
		err := fmt.Errorf("unexpected GitHub request %s %s", r.Method, r.URL.Path)
		s.recordError(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(s.cfg.Status)
	if s.cfg.Status < 200 || s.cfg.Status > 299 {
		_, _ = io.WriteString(w, s.cfg.ErrorBody)
		return
	}

	_, _ = fmt.Fprintf(w, `{"token":%q,"expires_at":%q}`, s.cfg.Token, s.cfg.ExpiresAt)
}

func (s *githubStub) validateCommon(r *http.Request) error {
	if got := r.Header.Get("Accept"); got != "application/vnd.github+json" {
		return fmt.Errorf("expected GitHub Accept header, got %q", got)
	}
	if got := r.Header.Get("User-Agent"); got != "github-token-broker" {
		return fmt.Errorf("expected github-token-broker User-Agent, got %q", got)
	}
	if got := r.Header.Get("X-GitHub-Api-Version"); got != "2022-11-28" {
		return fmt.Errorf("expected GitHub API version header, got %q", got)
	}

	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok || token == "" {
		return fmt.Errorf("expected bearer JWT authorization")
	}
	if err := verifyGitHubJWT(token, s.cfg.PublicKey, s.cfg.ClientID, time.Now().UTC()); err != nil {
		return err
	}

	s.mu.Lock()
	s.seenJWT = token
	s.mu.Unlock()
	return nil
}

func (s *githubStub) validateAccessTokenRequest(r *http.Request) error {
	if got := r.Header.Get("Content-Type"); got != "application/json" {
		return fmt.Errorf("expected JSON Content-Type header, got %q", got)
	}

	var body struct {
		Repositories []string          `json:"repositories"`
		Permissions  map[string]string `json:"permissions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return fmt.Errorf("decode GitHub token request: %w", err)
	}
	if !reflect.DeepEqual(body.Repositories, []string{s.cfg.Repository}) {
		return fmt.Errorf("expected repositories [%s], got %v", s.cfg.Repository, body.Repositories)
	}
	if !reflect.DeepEqual(body.Permissions, s.cfg.Permissions) {
		return fmt.Errorf("expected permissions %v, got %v", s.cfg.Permissions, body.Permissions)
	}

	return nil
}

func (s *githubStub) repositoryInstallationPath() string {
	return "/repos/" + s.cfg.Owner + "/" + s.cfg.Repository + "/installation"
}

func (s *githubStub) accessTokenPath() string {
	return "/app/installations/" + s.cfg.InstallationID + "/access_tokens"
}

func verifyGitHubJWT(token string, publicKey *rsa.PublicKey, clientID string, now time.Time) error {
	if publicKey == nil {
		return fmt.Errorf("GitHub stub public key is required")
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return fmt.Errorf("expected three-part JWT, got %d parts", len(parts))
	}

	header, err := decodeJWTPart[struct {
		Algorithm string `json:"alg"`
		Type      string `json:"typ"`
	}](parts[0])
	if err != nil {
		return fmt.Errorf("decode JWT header: %w", err)
	}
	if header.Algorithm != "RS256" || header.Type != "JWT" {
		return fmt.Errorf("unexpected JWT header: %+v", header)
	}

	claims, err := decodeJWTPart[struct {
		Issuer    string `json:"iss"`
		IssuedAt  int64  `json:"iat"`
		ExpiresAt int64  `json:"exp"`
	}](parts[1])
	if err != nil {
		return fmt.Errorf("decode JWT claims: %w", err)
	}
	if claims.Issuer != clientID {
		return fmt.Errorf("expected JWT issuer %q, got %q", clientID, claims.Issuer)
	}

	issuedAt := time.Unix(claims.IssuedAt, 0)
	if issuedAt.Before(now.Add(-3*time.Minute)) || issuedAt.After(now.Add(10*time.Second)) {
		return fmt.Errorf("JWT issued-at is outside expected bounds: %s", issuedAt)
	}

	expiresAt := time.Unix(claims.ExpiresAt, 0)
	if expiresAt.Before(now.Add(7*time.Minute)) || expiresAt.After(now.Add(11*time.Minute)) {
		return fmt.Errorf("JWT expiration is outside expected bounds: %s", expiresAt)
	}
	if expiresAt.Sub(issuedAt) < 9*time.Minute || expiresAt.Sub(issuedAt) > 11*time.Minute {
		return fmt.Errorf("JWT lifetime is outside expected bounds: %s", expiresAt.Sub(issuedAt))
	}

	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return fmt.Errorf("decode JWT signature: %w", err)
	}
	signingInput := parts[0] + "." + parts[1]
	digest := sha256.Sum256([]byte(signingInput))
	if err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, digest[:], signature); err != nil {
		return fmt.Errorf("verify JWT signature: %w", err)
	}

	return nil
}

func decodeJWTPart[T any](part string) (T, error) {
	var value T
	decoded, err := base64.RawURLEncoding.DecodeString(part)
	if err != nil {
		return value, err
	}
	if err := json.Unmarshal(decoded, &value); err != nil {
		return value, err
	}
	return value, nil
}

func (s *githubStub) recordError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errors = append(s.errors, err)
}

func (s *githubStub) assertRequests(t *testing.T, want int) {
	t.Helper()

	s.mu.Lock()
	defer s.mu.Unlock()

	assert.Equal(t, want, s.calls)
	for _, err := range s.errors {
		assert.NoError(t, err)
	}
}

func (s *githubStub) lastJWT() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.seenJWT
}
