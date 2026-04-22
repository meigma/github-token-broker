package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/meigma/github-token-broker/internal/broker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleAcceptsOnlyEmptyPayloads(t *testing.T) {
	tests := []struct {
		name    string
		payload json.RawMessage
		wantErr bool
	}{
		{name: "nil", payload: nil},
		{name: "null literal", payload: json.RawMessage("null")},
		{name: "whitespace", payload: json.RawMessage("   ")},
		{name: "object", payload: json.RawMessage("{}"), wantErr: true},
		{name: "array", payload: json.RawMessage("[]"), wantErr: true},
		{name: "string", payload: json.RawMessage(`"hi"`), wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			handler := New(&fakeBroker{
				response: broker.Response{
					Token:        "ghs_test",
					ExpiresAt:    time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC),
					Repositories: []string{"acme/widgets"},
					Permissions:  map[string]string{"contents": "read"},
				},
			}, slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil)))

			_, err := handler.Handle(context.Background(), tt.payload)

			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorContains(t, err, "does not accept invocation input")
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestHandleDoesNotLogToken(t *testing.T) {
	var logs bytes.Buffer
	handler := New(&fakeBroker{
		response: broker.Response{
			Token:        "ghs_secret_token",
			ExpiresAt:    time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC),
			Repositories: []string{"acme/widgets"},
			Permissions:  map[string]string{"contents": "read"},
		},
	}, slog.New(slog.NewJSONHandler(&logs, nil)))

	response, err := handler.Handle(context.Background(), nil)

	require.NoError(t, err)
	assert.Equal(t, "ghs_secret_token", response.Token)
	assert.NotContains(t, logs.String(), "ghs_secret_token")
}

func TestHandlePropagatesBrokerErrorsWithoutLeakingToken(t *testing.T) {
	var logs bytes.Buffer
	handler := New(&fakeBroker{
		response: broker.Response{Token: "ghs_never_returned"},
		err:      errors.New("upstream exploded"),
	}, slog.New(slog.NewJSONHandler(&logs, nil)))

	_, err := handler.Handle(context.Background(), nil)

	require.Error(t, err)
	assert.ErrorContains(t, err, "upstream exploded")
	assert.Contains(t, logs.String(), `"level":"ERROR"`)
	assert.NotContains(t, logs.String(), "ghs_never_returned")
}

type fakeBroker struct {
	response broker.Response
	err      error
}

func (f *fakeBroker) Mint(context.Context) (broker.Response, error) {
	if f.err != nil {
		return broker.Response{}, f.err
	}
	return f.response, nil
}
