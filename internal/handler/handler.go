// Package handler adapts the broker service to the AWS Lambda contract.
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/meigma/github-token-broker/internal/broker"
)

// TokenBroker mints GitHub installation tokens.
type TokenBroker interface {
	Mint(ctx context.Context) (broker.Response, error)
}

// Handler adapts the token broker service to Lambda.
type Handler struct {
	broker TokenBroker
	logger *slog.Logger
}

// New constructs a Handler.
//
// New defaults the logger to slog.Default() when nil so the handler is safe to
// use without explicit wiring in tests.
func New(tokenBroker TokenBroker, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return &Handler{
		broker: tokenBroker,
		logger: logger,
	}
}

// Handle rejects caller-selected input and returns a fixed-scope GitHub token.
//
// Handle accepts only empty or null Lambda payloads; any other input is
// rejected so callers cannot influence the token's scope. The minted token is
// never emitted to logs.
func (h *Handler) Handle(ctx context.Context, payload json.RawMessage) (broker.Response, error) {
	if err := validateEmptyPayload(payload); err != nil {
		h.logger.Warn("rejected non-empty invocation payload")
		return broker.Response{}, err
	}

	response, err := h.broker.Mint(ctx)
	if err != nil {
		h.logger.Error("mint GitHub installation token", "error", err)
		return broker.Response{}, err
	}

	h.logger.Info("minted GitHub installation token", "repositories", response.Repositories, "expires_at", response.ExpiresAt)

	return response, nil
}

func validateEmptyPayload(payload json.RawMessage) error {
	trimmed := bytes.TrimSpace(payload)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil
	}

	return fmt.Errorf("github-token-broker does not accept invocation input")
}
