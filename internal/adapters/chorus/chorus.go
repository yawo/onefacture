// Package chorus is the Chorus Pro / PPF adapter (PISTE OAuth2).
//
// The real implementation lives in chorus_real.go (build tag: piste). Without
// that tag, the adapter operates in stub mode: every call returns
// ErrNotImplemented so callers can surface a graceful 501.
package chorus

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/yawo/onefacture/internal/adapters"
	"github.com/yawo/onefacture/internal/adapters/sandbox"
	"github.com/yawo/onefacture/internal/core/invoice"
)

type Adapter struct {
	clientID     string
	clientSecret string
	baseURL      string
	client       sandbox.Client
}

func New() *Adapter {
	a := &Adapter{
		clientID:     os.Getenv("ONEFACTURE_CHORUS_CLIENT_ID"),
		clientSecret: os.Getenv("ONEFACTURE_CHORUS_CLIENT_SECRET"),
		baseURL:      envOr("ONEFACTURE_CHORUS_BASE_URL", "https://sandbox-api.piste.gouv.fr/cpro"),
	}
	a.client = sandbox.Client{
		Name:       "chorus",
		BaseURL:    a.baseURL,
		SubmitPath: envOr("ONEFACTURE_CHORUS_SUBMIT_PATH", "/invoices"),
		StatusPath: envOr("ONEFACTURE_CHORUS_STATUS_PATH", "/invoices/{pa_ref}/status"),
		WebhookKey: os.Getenv("ONEFACTURE_CHORUS_WEBHOOK_KEY"),
		Auth: sandbox.Auth{
			Scheme:       "Bearer",
			Token:        os.Getenv("ONEFACTURE_CHORUS_ACCESS_TOKEN"),
			TokenURL:     envOr("ONEFACTURE_CHORUS_TOKEN_URL", "https://sandbox-oauth.piste.gouv.fr/api/oauth/token"),
			ClientID:     a.clientID,
			ClientSecret: a.clientSecret,
			Scope:        os.Getenv("ONEFACTURE_CHORUS_SCOPE"),
		},
	}
	if strings.Contains(a.baseURL, "api.piste.gouv.fr") && !strings.Contains(strings.ToLower(a.baseURL), "sandbox") {
		if a.client.StatusMethod == "" {
			a.client.StatusMethod = "POST"
		}
		if a.client.StatusBodyTemplate == "" {
			a.client.StatusBodyTemplate = `{"identifiantFactureCPP":"{pa_ref}"}`
		}
		if a.client.SubmitPath == "/invoices" {
			a.client.SubmitPath = "/cpro/factures/v1/soumettre"
		}
		if a.client.StatusPath == "/invoices/{pa_ref}/status" {
			a.client.StatusPath = "/cpro/factures/v1/consulter/fournisseur"
		}
	}
	return a
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func (a *Adapter) Name() string { return "chorus" }

func (a *Adapter) HealthCheck(ctx context.Context) error {
	return a.client.HealthCheck(ctx)
}

func (a *Adapter) Submit(ctx context.Context, inv *invoice.Invoice) (*adapters.SubmitResult, error) {
	res, err := a.client.Submit(ctx, inv)
	if err != nil {
		return nil, a.mapChorusError("submit", err)
	}
	if res != nil {
		res.Status = invoice.Status(NormalizeLifecycleStatus(string(res.Status)))
	}
	return res, nil
}

func (a *Adapter) GetStatus(ctx context.Context, paRef string) (*adapters.LifecycleEvent, error) {
	ev, err := a.client.GetStatus(ctx, paRef)
	if err != nil {
		return nil, a.mapChorusError("get_status", err)
	}
	if ev != nil {
		ev.Status = invoice.Status(NormalizeLifecycleStatus(string(ev.Status)))
	}
	return ev, nil
}

func (a *Adapter) Webhook(ctx context.Context, payload []byte) (*adapters.WebhookEvent, error) {
	return a.client.Webhook(ctx, payload)
}

func (a *Adapter) mapChorusError(operation string, err error) error {
	var paErr *adapters.PAError
	if errors.As(err, &paErr) {
		switch paErr.Code {
		case "20001", "GDP_MSG_01.001", "20002", "GDP_MSG_01.002":
			paErr.Code = "CHORUS_INVALID_FORMAT"
			paErr.Message = "Format de flux invalide (PDF/A-3 ou XML CII requis)"
			paErr.Remediation = "Vérifiez le conteneur Factur-X, la signature et la taille (<10Mo)"
		case "20003", "20004", "20005":
			paErr.Code = "CHORUS_SIZE_OR_ATTACHMENT_ERROR"
			paErr.Remediation = "Réduisez la taille du fichier ou des pièces jointes"
		case "401", "403", "4":
			paErr.Code = "CHORUS_AUTH_ERROR"
			paErr.Remediation = "Vérifiez les variables ONEFACTURE_CHORUS_* (client_id/secret ou cpro-account)"
		case "429":
			paErr.Code = "CHORUS_RATE_LIMIT"
			paErr.Remediation = "Respectez les quotas PISTE ; retry avec backoff"
		case "500", "502", "503", "504":
			paErr.Code = "CHORUS_SERVER_ERROR"
			paErr.Remediation = "Erreur technique Chorus Pro ; réessayez plus tard"
		}
		paErr.Platform = "chorus"
		paErr.Operation = operation
		return paErr
	}
	return err
}

func NormalizeLifecycleStatus(chorusStatus string) string {
	s := strings.ToUpper(strings.TrimSpace(chorusStatus))
	switch s {
	case "DEPOSEE", "DEPOSITED", "SOUMISE", "ENREGISTREE":
		return "SUBMITTED"
	case "MISE_A_DISPOSITION", "MIS_A_DISPO", "ACCEPTED", "VALIDATED", "MISEADISPO":
		return "ACCEPTED"
	case "REJETEE", "REJECTED", "REFUSEE", "REJET":
		return "REJECTED"
	case "EN_COURS", "EN_TRAITEMENT", "PENDING", "SUSPENDUE":
		return "SUBMITTED"
	default:
		return "SUBMITTED"
	}
}
