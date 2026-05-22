package security

import (
	"bytes"
	"context"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHTTPKMSProviderRoundTripAndRotation(t *testing.T) {
	oldKey := bytes.Repeat([]byte{1}, 32)
	newKey := bytes.Repeat([]byte{2}, 32)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		switch r.URL.Path {
		case "/keys/active":
			_, _ = w.Write([]byte(`{"key_id":"v2","key":"` + hex.EncodeToString(newKey) + `"}`))
		case "/keys/v1":
			_, _ = w.Write([]byte(`{"key_id":"v1","key":"` + hex.EncodeToString(oldKey) + `"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	provider := HTTPKMSProvider{BaseURL: server.URL, BearerToken: "test-token", Client: server.Client()}
	keyID, key, err := provider.ActiveKey(context.Background())
	require.NoError(t, err)
	require.Equal(t, "v2", keyID)
	require.Equal(t, newKey, key)

	oldEnvelope, err := NewEncryptor(StaticKeyProvider{KeyID: "v1", Key: oldKey}).Encrypt(context.Background(), []byte("invoice"), []byte("org"))
	require.NoError(t, err)
	plain, err := NewEncryptor(provider).Decrypt(context.Background(), oldEnvelope, []byte("org"))
	require.NoError(t, err)
	require.Equal(t, []byte("invoice"), plain)
}

func TestDecodeAES256KeyRejectsInvalidLength(t *testing.T) {
	_, err := DecodeAES256Key(hex.EncodeToString(bytes.Repeat([]byte{1}, 16)))
	require.Error(t, err)
}
