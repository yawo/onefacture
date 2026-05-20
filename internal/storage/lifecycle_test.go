package storage

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/yawo/onefacture/internal/core/invoice"
)

func TestLifecycleRecordSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	orgID := uuid.New()
	invoiceID := uuid.New()
	ev := LifecycleEvent{
		FromStatus: invoice.StatusDraft,
		ToStatus:   invoice.StatusValidated,
		PACode:     "PA001",
		PAMessage:  "Invoice validated successfully",
		Payload:    map[string]any{"validation": "ok"},
	}

	err := store.Lifecycle.Record(ctx, orgID, invoiceID, ev)

	require.NoError(t, err)
}

func TestLifecycleRecordEmptyFromStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	orgID := uuid.New()
	invoiceID := uuid.New()
	ev := LifecycleEvent{
		FromStatus: "",
		ToStatus:   invoice.StatusValidated,
		PACode:     "PA001",
		PAMessage:  "Transition message",
		Payload:    map[string]any{},
	}

	err := store.Lifecycle.Record(ctx, orgID, invoiceID, ev)

	require.NoError(t, err)
}

func TestLifecycleRecordEmptyPACode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	orgID := uuid.New()
	invoiceID := uuid.New()
	ev := LifecycleEvent{
		FromStatus: invoice.StatusDraft,
		ToStatus:   invoice.StatusValidated,
		PACode:     "",
		PAMessage:  "No PA code",
		Payload:    map[string]any{},
	}

	err := store.Lifecycle.Record(ctx, orgID, invoiceID, ev)

	require.NoError(t, err)
}

func TestLifecycleRecordEmptyPAMessage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	orgID := uuid.New()
	invoiceID := uuid.New()
	ev := LifecycleEvent{
		FromStatus: invoice.StatusDraft,
		ToStatus:   invoice.StatusValidated,
		PACode:     "PA001",
		PAMessage:  "",
		Payload:    map[string]any{},
	}

	err := store.Lifecycle.Record(ctx, orgID, invoiceID, ev)

	require.NoError(t, err)
}

func TestLifecycleRecordNilPayload(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	orgID := uuid.New()
	invoiceID := uuid.New()
	ev := LifecycleEvent{
		FromStatus: invoice.StatusDraft,
		ToStatus:   invoice.StatusValidated,
		PACode:     "PA001",
		PAMessage:  "Message",
		Payload:    nil,
	}

	err := store.Lifecycle.Record(ctx, orgID, invoiceID, ev)

	require.NoError(t, err)
}

func TestLifecycleRecordComplexPayload(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	orgID := uuid.New()
	invoiceID := uuid.New()
	ev := LifecycleEvent{
		FromStatus: invoice.StatusDraft,
		ToStatus:   invoice.StatusSubmitted,
		PACode:     "PA001",
		PAMessage:  "Submitted",
		Payload: map[string]any{
			"pa_ref": "ref-123456",
			"timestamp": "2024-01-01T00:00:00Z",
			"details": map[string]any{
				"seller_siren": "123456782",
				"buyer_siren":  "987654321",
				"amount": 1500.50,
			},
			"status_history": []any{
				map[string]any{"status": "pending", "timestamp": "2024-01-01T00:00:00Z"},
				map[string]any{"status": "submitted", "timestamp": "2024-01-01T00:05:00Z"},
			},
		},
	}

	err := store.Lifecycle.Record(ctx, orgID, invoiceID, ev)

	require.NoError(t, err)
}

func TestLifecycleListEmpty(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	events, err := store.Lifecycle.List(ctx, uuid.New(), uuid.New())

	require.NoError(t, err)
	require.Empty(t, events)
}

func TestLifecycleListSingle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	orgID := uuid.New()
	invoiceID := uuid.New()
	ev := LifecycleEvent{
		FromStatus: invoice.StatusDraft,
		ToStatus:   invoice.StatusValidated,
		PACode:     "PA001",
		PAMessage:  "Validated",
		Payload:    map[string]any{},
	}

	err := store.Lifecycle.Record(ctx, orgID, invoiceID, ev)
	require.NoError(t, err)

	events, err := store.Lifecycle.List(ctx, orgID, invoiceID)

	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, invoice.StatusValidated, events[0].ToStatus)
}

func TestLifecycleListMultiple(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	orgID := uuid.New()
	invoiceID := uuid.New()

	statuses := []struct {
		from invoice.Status
		to   invoice.Status
	}{
		{invoice.StatusDraft, invoice.StatusValidated},
		{invoice.StatusValidated, invoice.StatusSubmitted},
		{invoice.StatusSubmitted, invoice.StatusAccepted},
		{invoice.StatusAccepted, invoice.StatusPaid},
	}

	for _, s := range statuses {
		ev := LifecycleEvent{
			FromStatus: s.from,
			ToStatus:   s.to,
			PACode:     "PA001",
			PAMessage:  "Status changed",
			Payload:    map[string]any{},
		}
		err := store.Lifecycle.Record(ctx, orgID, invoiceID, ev)
		require.NoError(t, err)
	}

	events, err := store.Lifecycle.List(ctx, orgID, invoiceID)

	require.NoError(t, err)
	require.Len(t, events, 4)

	for i, s := range statuses {
		require.Equal(t, s.to, events[i].ToStatus)
	}
}

func TestLifecycleListOrdered(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	orgID := uuid.New()
	invoiceID := uuid.New()

	for i := 0; i < 5; i++ {
		ev := LifecycleEvent{
			FromStatus: invoice.StatusDraft,
			ToStatus:   invoice.StatusValidated,
			PACode:     "PA001",
			PAMessage:  "Event",
			Payload:    map[string]any{"seq": i},
		}
		err := store.Lifecycle.Record(ctx, orgID, invoiceID, ev)
		require.NoError(t, err)
	}

	events, err := store.Lifecycle.List(ctx, orgID, invoiceID)

	require.NoError(t, err)
	require.Len(t, events, 5)

	for i := 0; i < len(events)-1; i++ {
		require.True(t, events[i].OccurredAt.Before(events[i+1].OccurredAt) || events[i].OccurredAt.Equal(events[i+1].OccurredAt),
			"events should be ordered by occurred_at ASC")
	}
}

func TestLifecycleListIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	orgID1 := uuid.New()
	orgID2 := uuid.New()
	invoiceID1 := uuid.New()
	invoiceID2 := uuid.New()

	ev1 := LifecycleEvent{
		FromStatus: invoice.StatusDraft,
		ToStatus:   invoice.StatusValidated,
		Payload:    map[string]any{},
	}

	err := store.Lifecycle.Record(ctx, orgID1, invoiceID1, ev1)
	require.NoError(t, err)
	err = store.Lifecycle.Record(ctx, orgID2, invoiceID2, ev1)
	require.NoError(t, err)

	events1, err := store.Lifecycle.List(ctx, orgID1, invoiceID1)
	require.NoError(t, err)
	require.Len(t, events1, 1)

	events2, err := store.Lifecycle.List(ctx, orgID2, invoiceID2)
	require.NoError(t, err)
	require.Len(t, events2, 1)

	events12, err := store.Lifecycle.List(ctx, orgID1, invoiceID2)
	require.NoError(t, err)
	require.Empty(t, events12)
}

func TestLifecycleRecordDifferentInvoices(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	orgID := uuid.New()
	invoiceID1 := uuid.New()
	invoiceID2 := uuid.New()

	ev := LifecycleEvent{
		FromStatus: invoice.StatusDraft,
		ToStatus:   invoice.StatusValidated,
		Payload:    map[string]any{},
	}

	err := store.Lifecycle.Record(ctx, orgID, invoiceID1, ev)
	require.NoError(t, err)
	err = store.Lifecycle.Record(ctx, orgID, invoiceID2, ev)
	require.NoError(t, err)

	events1, err := store.Lifecycle.List(ctx, orgID, invoiceID1)
	require.NoError(t, err)
	require.Len(t, events1, 1)

	events2, err := store.Lifecycle.List(ctx, orgID, invoiceID2)
	require.NoError(t, err)
	require.Len(t, events2, 1)
}

func TestLifecycleRecordAllStatuses(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	orgID := uuid.New()
	invoiceID := uuid.New()

	statuses := []invoice.Status{
		invoice.StatusDraft,
		invoice.StatusValidated,
		invoice.StatusSubmitted,
		invoice.StatusReceived,
		invoice.StatusAccepted,
		invoice.StatusRejected,
		invoice.StatusPaid,
		invoice.StatusCancelled,
	}

	for i, s := range statuses {
		var from invoice.Status
		if i > 0 {
			from = statuses[i-1]
		}

		ev := LifecycleEvent{
			FromStatus: from,
			ToStatus:   s,
			Payload:    map[string]any{},
		}
		err := store.Lifecycle.Record(ctx, orgID, invoiceID, ev)
		require.NoError(t, err)
	}

	events, err := store.Lifecycle.List(ctx, orgID, invoiceID)
	require.NoError(t, err)
	require.Len(t, events, len(statuses))
}

func TestLifecycleRecordLongPAMessage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	orgID := uuid.New()
	invoiceID := uuid.New()

	longMessage := "This is a very long PA message that contains detailed information about the status transition and may include error codes and descriptions that could span multiple lines if printed. " +
		"It may include details about the validation process, submission status, and any issues encountered during the processing. " +
		"The message can be quite lengthy to provide comprehensive information about what happened."

	ev := LifecycleEvent{
		FromStatus: invoice.StatusDraft,
		ToStatus:   invoice.StatusValidated,
		PACode:     "PA001",
		PAMessage:  longMessage,
		Payload:    map[string]any{},
	}

	err := store.Lifecycle.Record(ctx, orgID, invoiceID, ev)
	require.NoError(t, err)

	events, err := store.Lifecycle.List(ctx, orgID, invoiceID)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, longMessage, events[0].PAMessage)
}

func TestLifecycleRecordUnicodePayload(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	orgID := uuid.New()
	invoiceID := uuid.New()

	ev := LifecycleEvent{
		FromStatus: invoice.StatusDraft,
		ToStatus:   invoice.StatusValidated,
		Payload: map[string]any{
			"message": "Facture validée avec succès",
			"vendor": "社内ベンダー",
			"details": "Détails: Société Générale français",
		},
	}

	err := store.Lifecycle.Record(ctx, orgID, invoiceID, ev)
	require.NoError(t, err)

	events, err := store.Lifecycle.List(ctx, orgID, invoiceID)
	require.NoError(t, err)
	require.Len(t, events, 1)
}
