//go:build integration

package integration

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/require"
)

type testPrivateKey struct {
	privateKey *rsa.PrivateKey
	privatePEM string
}

func generateTestPrivateKey(t *testing.T) testPrivateKey {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	encoded := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	require.NotNil(t, encoded)

	return testPrivateKey{
		privateKey: privateKey,
		privatePEM: string(encoded),
	}
}
