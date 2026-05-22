package security

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

type KeyProvider interface {
	ActiveKey(ctx context.Context) (keyID string, key []byte, err error)
}

type KeyResolver interface {
	ResolveKey(ctx context.Context, keyID string) ([]byte, error)
}

type StaticKeyProvider struct {
	KeyID string
	Key   []byte
}

func (p StaticKeyProvider) ActiveKey(context.Context) (string, []byte, error) {
	if len(p.Key) != 32 {
		return "", nil, fmt.Errorf("static key must be 32 bytes for AES-256")
	}
	key := make([]byte, len(p.Key))
	copy(key, p.Key)
	return p.KeyID, key, nil
}

func (p StaticKeyProvider) ResolveKey(_ context.Context, keyID string) ([]byte, error) {
	if keyID != "" && keyID != p.KeyID {
		return nil, fmt.Errorf("key %q not found", keyID)
	}
	if len(p.Key) != 32 {
		return nil, fmt.Errorf("static key must be 32 bytes for AES-256")
	}
	key := make([]byte, len(p.Key))
	copy(key, p.Key)
	return key, nil
}

type StaticKeyringProvider struct {
	ActiveKeyID string
	Keys        map[string][]byte
}

func (p StaticKeyringProvider) ActiveKey(ctx context.Context) (string, []byte, error) {
	key, err := p.ResolveKey(ctx, p.ActiveKeyID)
	return p.ActiveKeyID, key, err
}

func (p StaticKeyringProvider) ResolveKey(_ context.Context, keyID string) ([]byte, error) {
	key, ok := p.Keys[keyID]
	if !ok {
		return nil, fmt.Errorf("key %q not found", keyID)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("key %q must be 32 bytes for AES-256", keyID)
	}
	out := make([]byte, len(key))
	copy(out, key)
	return out, nil
}

type Envelope struct {
	KeyID      string `json:"key_id"`
	Nonce      []byte `json:"nonce"`
	Ciphertext []byte `json:"ciphertext"`
}

type Encryptor struct {
	provider KeyProvider
}

func NewEncryptor(provider KeyProvider) *Encryptor {
	return &Encryptor{provider: provider}
}

func (e *Encryptor) Encrypt(ctx context.Context, plaintext, aad []byte) (Envelope, error) {
	keyID, key, err := e.provider.ActiveKey(ctx)
	if err != nil {
		return Envelope{}, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return Envelope{}, fmt.Errorf("aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return Envelope{}, fmt.Errorf("gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return Envelope{}, fmt.Errorf("nonce: %w", err)
	}
	return Envelope{KeyID: keyID, Nonce: nonce, Ciphertext: gcm.Seal(nil, nonce, plaintext, aad)}, nil
}

func (e *Encryptor) Decrypt(ctx context.Context, env Envelope, aad []byte) ([]byte, error) {
	var key []byte
	var err error
	if resolver, ok := e.provider.(KeyResolver); ok && env.KeyID != "" {
		key, err = resolver.ResolveKey(ctx, env.KeyID)
	} else {
		_, key, err = e.provider.ActiveKey(ctx)
	}
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}
	plaintext, err := gcm.Open(nil, env.Nonce, env.Ciphertext, aad)
	if err != nil {
		return nil, fmt.Errorf("decrypt envelope %s: %w", env.KeyID, err)
	}
	return plaintext, nil
}
