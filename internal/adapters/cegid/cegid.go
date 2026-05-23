package cegid

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
		Name:       "cegid",
		BaseURL:    os.Getenv("ONEFACTURE_CEGID_BASE_URL"),
		SubmitPath: envOr("ONEFACTURE_CEGID_SUBMIT_PATH", "/invoices"),
		StatusPath: envOr("ONEFACTURE_CEGID_STATUS_PATH", "/invoices/{pa_ref}/status"),
		WebhookKey: os.Getenv("ONEFACTURE_CEGID_WEBHOOK_KEY"),
		Auth:       sandbox.Auth{Scheme: "Bearer", Token: os.Getenv("ONEFACTURE_CEGID_API_TOKEN")},
	}}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func (a *Adapter) Name() string { return "cegid" }

func (a *Adapter) HealthCheck(ctx context.Context) error { return a.client.HealthCheck(ctx) }

func (a *Adapter) Submit(ctx context.Context, inv *invoice.Invoice) (*adapters.SubmitResult, error) {
	res, err := a.client.Submit(ctx, inv)
	if err != nil {
		return nil, a.mapCegidError("submit", err)
	}
	if res != nil {
		res.Status = invoice.Status(NormalizeLifecycleStatus(string(res.Status)))
	}
	return res, nil
}

func (a *Adapter) GetStatus(ctx context.Context, paRef string) (*adapters.LifecycleEvent, error) {
	ev, err := a.client.GetStatus(ctx, paRef)
	if err != nil {
		return nil, a.mapCegidError("get_status", err)
	}
	if ev != nil {
		ev.Status = invoice.Status(NormalizeLifecycleStatus(string(ev.Status)))
	}
	return ev, nil
}

// mapCegidError permet d’enrichir ou de normaliser les erreurs spécifiques à Cegid
// (ex: codes d’erreur métier Cegid, messages en français, hints de correction…).
func (a *Adapter) mapCegidError(operation string, err error) error {
	var paErr *adapters.PAError
	if errors.As(err, &paErr) {
		// Exemple de mapping spécifique Cegid (à enrichir avec les vrais codes)
		switch paErr.Code {
		case "CEG-001", "INVALID_SIREN":
			paErr.Code = "CEGID_INVALID_SIREN"
			paErr.Message = "SIREN du destinataire invalide selon les règles Cegid"
			paErr.Remediation = "Vérifiez le SIREN du buyer et les règles de routage Cegid"
		case "CEG-042":
			paErr.Code = "CEGID_INVOICE_ALREADY_EXISTS"
			paErr.Remediation = "Utilisez un Idempotency-Key différent ou vérifiez le doublon"
		}
		paErr.Platform = "cegid"
		paErr.Operation = operation
		return paErr
	}
	return err
}

// NormalizeLifecycleStatus maps Cegid-specific statuses to onefacture core statuses.
// This is an example of adapter-specific logic.
func NormalizeLifecycleStatus(cegidStatus string) string {
	switch cegidStatus {
	case "ACCEPTE", "VALIDATED", "OK":
		return "ACCEPTED"
	case "REFUSE", "REJECTED", "KO":
		return "REJECTED"
	case "EN_COURS", "PENDING", "IN_PROGRESS":
		return "SUBMITTED"
	default:
		return "SUBMITTED"
	}
}

func (a *Adapter) Webhook(ctx context.Context, payload []byte) (*adapters.WebhookEvent, error) {
	return a.client.Webhook(ctx, payload)
}
