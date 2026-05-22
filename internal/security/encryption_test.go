package security

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncryptorRoundTrip(t *testing.T) {
	key := bytes.Repeat([]byte{1}, 32)
	enc := NewEncryptor(StaticKeyProvider{KeyID: "local-v1", Key: key})

	env, err := enc.Encrypt(context.Background(), []byte("secret invoice"), []byte("org-1"))
	require.NoError(t, err)
	require.NotEqual(t, []byte("secret invoice"), env.Ciphertext)

	plain, err := enc.Decrypt(context.Background(), env, []byte("org-1"))
	require.NoError(t, err)
	require.Equal(t, []byte("secret invoice"), plain)
}

func TestEncryptorRejectsWrongAAD(t *testing.T) {
	key := bytes.Repeat([]byte{1}, 32)
	enc := NewEncryptor(StaticKeyProvider{KeyID: "local-v1", Key: key})

	env, err := enc.Encrypt(context.Background(), []byte("secret invoice"), []byte("org-1"))
	require.NoError(t, err)
	_, err = enc.Decrypt(context.Background(), env, []byte("org-2"))

	require.Error(t, err)
}

func TestEncryptorDecryptsOldEnvelopeAfterRotation(t *testing.T) {
	oldKey := bytes.Repeat([]byte{1}, 32)
	newKey := bytes.Repeat([]byte{2}, 32)
	oldEnc := NewEncryptor(StaticKeyringProvider{
		ActiveKeyID: "v1",
		Keys:        map[string][]byte{"v1": oldKey, "v2": newKey},
	})
	env, err := oldEnc.Encrypt(context.Background(), []byte("secret invoice"), []byte("org-1"))
	require.NoError(t, err)

	rotatedEnc := NewEncryptor(StaticKeyringProvider{
		ActiveKeyID: "v2",
		Keys:        map[string][]byte{"v1": oldKey, "v2": newKey},
	})
	plain, err := rotatedEnc.Decrypt(context.Background(), env, []byte("org-1"))

	require.NoError(t, err)
	require.Equal(t, []byte("secret invoice"), plain)
}
