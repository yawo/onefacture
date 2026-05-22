// Package adapters defines the PAAdapter interface implemented by each
// Plateforme Agréée integration (Chorus Pro, Pennylane, Docaposte, etc.).
package adapters

import (
	"context"
	"fmt"
	"time"

	"github.com/yawo/onefacture/internal/core/invoice"
)

// PAAdapter is implemented by every concrete PA integration.
type PAAdapter interface {
	Name() string
	Submit(ctx context.Context, inv *invoice.Invoice) (*SubmitResult, error)
	GetStatus(ctx context.Context, paRef string) (*LifecycleEvent, error)
	Webhook(ctx context.Context, payload []byte) (*WebhookEvent, error)
	HealthCheck(ctx context.Context) error
}

// SubmitResult is the response from a PA after submitting an invoice.
type SubmitResult struct {
	PARef      string         `json:"pa_ref"`
	Status     invoice.Status `json:"status"`
	AcceptedAt time.Time      `json:"accepted_at"`
	Raw        []byte         `json:"-"`
}

// LifecycleEvent is the normalised PA-side status update.
type LifecycleEvent struct {
	PARef      string         `json:"pa_ref"`
	Status     invoice.Status `json:"status"`
	PACode     string         `json:"pa_code,omitempty"`
	PAMessage  string         `json:"pa_message,omitempty"`
	OccurredAt time.Time      `json:"occurred_at"`
}

// WebhookEvent is the normalised inbound event from a PA webhook.
type WebhookEvent struct {
	EventType string         `json:"event_type"`
	PARef     string         `json:"pa_ref"`
	Status    invoice.Status `json:"status,omitempty"`
	Payload   map[string]any `json:"payload,omitempty"`
}

// ErrNotImplemented is returned by stub adapters when a method is not wired up.
var ErrNotImplemented = errStr("adapter: not implemented")

type errStr string

func (e errStr) Error() string { return string(e) }

type PAError struct {
	Platform   string `json:"platform"`
	Operation  string `json:"operation"`
	StatusCode int    `json:"status_code"`
	Code       string `json:"code,omitempty"`
	Message    string `json:"message"`
	Retryable  bool   `json:"retryable"`
	Raw        []byte `json:"-"`
}

func (e *PAError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("%s %s failed: %s (%s)", e.Platform, e.Operation, e.Message, e.Code)
	}
	return fmt.Sprintf("%s %s failed: %s", e.Platform, e.Operation, e.Message)
}
