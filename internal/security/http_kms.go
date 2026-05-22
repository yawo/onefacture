package security

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type HTTPKMSProvider struct {
	BaseURL     string
	BearerToken string
	Client      *http.Client
}

type httpKMSKeyResponse struct {
	KeyID string `json:"key_id"`
	Key   string `json:"key"`
}

func (p HTTPKMSProvider) ActiveKey(ctx context.Context) (string, []byte, error) {
	return p.fetch(ctx, "/keys/active")
}

func (p HTTPKMSProvider) ResolveKey(ctx context.Context, keyID string) ([]byte, error) {
	if strings.TrimSpace(keyID) == "" {
		return nil, fmt.Errorf("key_id is required")
	}
	_, key, err := p.fetch(ctx, "/keys/"+url.PathEscape(keyID))
	return key, err
}

func (p HTTPKMSProvider) fetch(ctx context.Context, path string) (string, []byte, error) {
	base := strings.TrimRight(p.BaseURL, "/")
	if base == "" {
		return "", nil, fmt.Errorf("kms base url is required")
	}
	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+path, nil)
	if err != nil {
		return "", nil, fmt.Errorf("kms request: %w", err)
	}
	if p.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+p.BearerToken)
	}
	res, err := client.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("kms request: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("kms returned status %d", res.StatusCode)
	}
	var payload httpKMSKeyResponse
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return "", nil, fmt.Errorf("decode kms response: %w", err)
	}
	key, err := DecodeAES256Key(payload.Key)
	if err != nil {
		return "", nil, fmt.Errorf("kms key %q: %w", payload.KeyID, err)
	}
	return payload.KeyID, key, nil
}

func DecodeAES256Key(raw string) ([]byte, error) {
	key, err := hex.DecodeString(raw)
	if err != nil || len(key) != 32 {
		key, err = base64.StdEncoding.DecodeString(raw)
	}
	if err != nil || len(key) != 32 {
		return nil, fmt.Errorf("key must be 32 bytes encoded as hex or base64")
	}
	return key, nil
}
