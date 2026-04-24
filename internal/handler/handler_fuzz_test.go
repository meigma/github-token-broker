package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/meigma/github-token-broker/internal/broker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func FuzzHandle(f *testing.F) {
	seeds := [][]byte{
		nil,
		[]byte("null"),
		[]byte("   "),
		[]byte("{}"),
		[]byte("[]"),
		[]byte(`"hi"`),
		[]byte("Null"),
		[]byte("NULL"),
		[]byte("nul"),
		[]byte("nulll"),
		[]byte("null\x00"),
		[]byte(" null "),
		[]byte("\tnull\n"),
		[]byte("\xef\xbb\xbf"),
		[]byte(`{"foo":1}`),
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, payload []byte) {
		h := New(&fakeBroker{
			response: broker.Response{
				Token:        "ghs_fuzz",
				ExpiresAt:    time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC),
				Repositories: []string{"acme/widgets"},
				Permissions:  map[string]string{"contents": "read"},
			},
		}, slog.New(slog.NewJSONHandler(io.Discard, nil)))

		_, err := h.Handle(context.Background(), json.RawMessage(payload))

		trimmed := bytes.TrimSpace(payload)
		shouldAccept := len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null"))

		if shouldAccept {
			require.NoError(t, err, "payload=%q should have been accepted", payload)
			return
		}
		require.Error(t, err, "payload=%q should have been rejected", payload)
		assert.ErrorContains(t, err, "does not accept invocation input")
	})
}
