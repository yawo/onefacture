package qonto

import (
	"context"
	"errors"
	"os"

	"github.com/yawo/onefacture/internal/adapters"
	"github.com/yawo/onefacture/internal/adapters/sandbox"
	"github.com/yawo/onefacture/internal/core/invoice"
)

type Adapter struct{ client sandbox.Client }

func New() *Adapter {
	return &Adapter{client: sandbox.Client{
		Name:       "qonto",
		BaseURL:    os.Getenv("ONEFACTURE_QONTO_BASE_URL"),
		SubmitPath: envOr("ONEFACTURE_QONTO_SUBMIT_PATH", "/invoices"),
		StatusPath: envOr("ONEFACTURE_QONTO_STATUS_PATH", "/invoices/{pa_ref}/status"),
		WebhookKey: os.Getenv("ONEFACTURE_QONTO_WEBHOOK_KEY"),
		Auth:       sandbox.Auth{Scheme: "Bearer", Token: os.Getenv("ONEFACTURE_QONTO_API_TOKEN")},
	}}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func (a *Adapter) Name() string { return "qonto" }

func (a *Adapter) HealthCheck(ctx context.Context) error { return a.client.HealthCheck(ctx) }

func (a *Adapter) Submit(ctx context.Context, inv *invoice.Invoice) (*adapters.SubmitResult, error) {
	res, err := a.client.Submit(ctx, inv)
	if err != nil {
		return nil, a.mapQontoError("submit", err)
	}
	return res, nil
}

func (a *Adapter) GetStatus(ctx context.Context, paRef string) (*adapters.LifecycleEvent, error) {
	ev, err := a.client.GetStatus(ctx, paRef)
	if err != nil {
		return nil, a.mapQontoError("get_status", err)
	}
	if ev != nil {
		ev.Status = invoice.Status(NormalizeLifecycleStatus(string(ev.Status)))
	}
	return ev, nil
}

// mapQontoError enrichit les erreurs avec les codes métier spécifiques à Qonto.
func (a *Adapter) mapQontoError(operation string, err error) error {
	var paErr *adapters.PAError
	if errors.As(err, &paErr) {
		switch paErr.Code {
		case "QON-101", "INVALID_IBAN":
			paErr.Code = "QONTO_INVALID_IBAN"
			paErr.Message = "IBAN du fournisseur invalide selon les règles Qonto"
			paErr.Remediation = "Vérifiez l’IBAN du bénéficiaire dans les données de paiement"
		case "QON-204":
			paErr.Code = "QONTO_DUPLICATE_INVOICE"
			paErr.Remediation = "Facture déjà soumise — utilisez un Idempotency-Key"
		}
		paErr.Platform = "qonto"
		paErr.Operation = operation
		return paErr
	}
	return err
}

// NormalizeLifecycleStatus maps Qonto-specific statuses to onefacture core statuses.
func NormalizeLifecycleStatus(qontoStatus string) string {
	switch qontoStatus {
	case "accepted", "validated", "ok":
		return "ACCEPTED"
	case "rejected", "refused", "ko":
		return "REJECTED"
	case "pending", "in_progress":
		return "SUBMITTED"
	default:
		return "SUBMITTED"
	}
}

func (a *Adapter) Webhook(ctx context.Context, payload []byte) (*adapters.WebhookEvent, error) {
	return a.client.Webhook(ctx, payload)
}
