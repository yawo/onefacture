// Package mock is the in-memory PA adapter used for local development and tests.
package mock

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/yawo/onefacture/internal/adapters"
	"github.com/yawo/onefacture/internal/core/invoice"
)

type submission struct {
	inv      *invoice.Invoice
	status   invoice.Status
	updated  time.Time
}

// Adapter implements PAAdapter without leaving the process.
type Adapter struct {
	mu          sync.Mutex
	submissions map[string]*submission
}

// New returns a fresh mock adapter.
func New() *Adapter {
	return &Adapter{submissions: map[string]*submission{}}
}

func (a *Adapter) Name() string { return "mock" }

func (a *Adapter) HealthCheck(_ context.Context) error { return nil }

func (a *Adapter) Submit(_ context.Context, inv *invoice.Invoice) (*adapters.SubmitResult, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	ref := "mock_" + uuid.NewString()
	a.submissions[ref] = &submission{inv: inv, status: invoice.StatusSubmitted, updated: time.Now().UTC()}
	return &adapters.SubmitResult{
		PARef:      ref,
		Status:     invoice.StatusSubmitted,
		AcceptedAt: time.Now().UTC(),
	}, nil
}

func (a *Adapter) GetStatus(_ context.Context, paRef string) (*adapters.LifecycleEvent, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	s, ok := a.submissions[paRef]
	if !ok {
		return nil, fmt.Errorf("unknown pa_ref %q", paRef)
	}
	// Walk a happy-path lifecycle on every poll: SUBMITTED -> RECEIVED -> ACCEPTED.
	switch s.status {
	case invoice.StatusSubmitted:
		s.status = invoice.StatusReceived
	case invoice.StatusReceived:
		s.status = invoice.StatusAccepted
	}
	s.updated = time.Now().UTC()
	return &adapters.LifecycleEvent{
		PARef:      paRef,
		Status:     s.status,
		OccurredAt: s.updated,
	}, nil
}

func (a *Adapter) Webhook(_ context.Context, payload []byte) (*adapters.WebhookEvent, error) {
	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, fmt.Errorf("decode webhook: %w", err)
	}
	ref, _ := raw["pa_ref"].(string)
	statusStr, _ := raw["status"].(string)
	eventType, _ := raw["event_type"].(string)
	if eventType == "" {
		eventType = "invoice.updated"
	}
	return &adapters.WebhookEvent{
		EventType: eventType,
		PARef:     ref,
		Status:    invoice.Status(statusStr),
		Payload:   raw,
	}, nil
}
