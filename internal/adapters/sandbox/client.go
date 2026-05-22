package sandbox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/yawo/onefacture/internal/adapters"
	"github.com/yawo/onefacture/internal/core/invoice"
)

type Auth struct {
	Scheme       string
	Token        string
	TokenURL     string
	ClientID     string
	ClientSecret string
	Scope        string
}

type Client struct {
	Name       string
	BaseURL    string
	SubmitPath string
	StatusPath string
	WebhookKey string
	Auth       Auth
	HTTP       *http.Client
}

func (c Client) Ready() bool {
	hasStaticToken := strings.TrimSpace(c.Auth.Token) != ""
	hasClientCredentials := strings.TrimSpace(c.Auth.TokenURL) != "" && strings.TrimSpace(c.Auth.ClientID) != "" && strings.TrimSpace(c.Auth.ClientSecret) != ""
	return strings.TrimSpace(c.BaseURL) != "" && (hasStaticToken || hasClientCredentials)
}

func (c Client) HealthCheck(ctx context.Context) error {
	if !c.Ready() {
		return adapters.ErrNotImplemented
	}
	if strings.TrimSpace(c.BaseURL) == "" {
		return adapters.ErrNotImplemented
	}
	if strings.TrimSpace(c.SubmitPath) != "" {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(c.BaseURL, "/")+"/health", nil)
	if err != nil {
		return fmt.Errorf("%s health request: %w", c.Name, err)
	}
	if err := c.authorize(ctx, req); err != nil {
		return err
	}
	resp, err := c.http().Do(req)
	if err != nil {
		return fmt.Errorf("%s health: %w", c.Name, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("%s health status %d", c.Name, resp.StatusCode)
	}
	return nil
}

func (c Client) Submit(ctx context.Context, inv *invoice.Invoice) (*adapters.SubmitResult, error) {
	if !c.Ready() {
		return nil, adapters.ErrNotImplemented
	}
	body, err := json.Marshal(inv)
	if err != nil {
		return nil, fmt.Errorf("%s marshal invoice: %w", c.Name, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url(c.SubmitPath), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("%s submit request: %w", c.Name, err)
	}
	req.Header.Set("Content-Type", "application/json")
	if err := c.authorize(ctx, req); err != nil {
		return nil, err
	}
	resp, err := c.http().Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s submit: %w", c.Name, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return nil, c.paError("submit", resp)
	}
	var out adapters.SubmitResult
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("%s submit decode: %w", c.Name, err)
	}
	if out.Status == "" {
		out.Status = invoice.StatusSubmitted
	}
	if out.AcceptedAt.IsZero() {
		out.AcceptedAt = time.Now().UTC()
	}
	return &out, nil
}

func (c Client) GetStatus(ctx context.Context, paRef string) (*adapters.LifecycleEvent, error) {
	if !c.Ready() {
		return nil, adapters.ErrNotImplemented
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url(strings.ReplaceAll(c.StatusPath, "{pa_ref}", paRef)), nil)
	if err != nil {
		return nil, fmt.Errorf("%s status request: %w", c.Name, err)
	}
	if err := c.authorize(ctx, req); err != nil {
		return nil, err
	}
	resp, err := c.http().Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s status: %w", c.Name, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return nil, c.paError("status", resp)
	}
	var out adapters.LifecycleEvent
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("%s status decode: %w", c.Name, err)
	}
	if out.OccurredAt.IsZero() {
		out.OccurredAt = time.Now().UTC()
	}
	return &out, nil
}

func (c Client) Webhook(_ context.Context, payload []byte) (*adapters.WebhookEvent, error) {
	if c.WebhookKey == "" {
		return nil, adapters.ErrNotImplemented
	}
	var out adapters.WebhookEvent
	if err := json.Unmarshal(payload, &out); err != nil {
		return nil, fmt.Errorf("%s webhook decode: %w", c.Name, err)
	}
	return &out, nil
}

func (c Client) url(path string) string {
	return strings.TrimRight(c.BaseURL, "/") + "/" + strings.TrimLeft(path, "/")
}

func (c Client) authorize(ctx context.Context, req *http.Request) error {
	token := c.Auth.Token
	scheme := c.Auth.Scheme
	if token == "" {
		oauthScheme, oauthToken, err := c.clientCredentialsToken(ctx)
		if err != nil {
			return err
		}
		token = oauthToken
		if scheme == "" {
			scheme = oauthScheme
		}
	}
	if scheme == "" {
		scheme = "Bearer"
	}
	req.Header.Set("Authorization", scheme+" "+token)
	return nil
}

func (c Client) clientCredentialsToken(ctx context.Context) (string, string, error) {
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", c.Auth.ClientID)
	form.Set("client_secret", c.Auth.ClientSecret)
	if c.Auth.Scope != "" {
		form.Set("scope", c.Auth.Scope)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Auth.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", fmt.Errorf("%s oauth token request: %w", c.Name, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.http().Do(req)
	if err != nil {
		return "", "", fmt.Errorf("%s oauth token: %w", c.Name, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return "", "", fmt.Errorf("%s oauth token status %d", c.Name, resp.StatusCode)
	}
	var out struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", "", fmt.Errorf("%s oauth token decode: %w", c.Name, err)
	}
	if out.AccessToken == "" {
		return "", "", fmt.Errorf("%s oauth token response missing access_token", c.Name)
	}
	if out.TokenType == "" {
		out.TokenType = "Bearer"
	}
	return out.TokenType, out.AccessToken, nil
}

func (c Client) paError(operation string, resp *http.Response) error {
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	out := &adapters.PAError{
		Platform:   c.Name,
		Operation:  operation,
		StatusCode: resp.StatusCode,
		Message:    http.StatusText(resp.StatusCode),
		Retryable:  resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500,
		Raw:        raw,
	}
	var payload struct {
		Code             string `json:"code"`
		Message          string `json:"message"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
		Retryable        *bool  `json:"retryable"`
	}
	if err := json.Unmarshal(raw, &payload); err == nil {
		if payload.Code != "" {
			out.Code = payload.Code
		} else if payload.Error != "" {
			out.Code = payload.Error
		}
		if payload.Message != "" {
			out.Message = payload.Message
		} else if payload.ErrorDescription != "" {
			out.Message = payload.ErrorDescription
		}
		if payload.Retryable != nil {
			out.Retryable = *payload.Retryable
		}
	}
	return out
}

func (c Client) http() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 30 * time.Second}
}
