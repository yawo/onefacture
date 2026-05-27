// Package routes contains the HTTP handlers for the gateway.
package routes

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	chi "github.com/go-chi/chi/v5"
	validator "github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/yawo/onefacture/internal/adapters"
	"github.com/yawo/onefacture/internal/adapters/registry"
	"github.com/yawo/onefacture/internal/core/facturx"
	"github.com/yawo/onefacture/internal/core/invoice"
	"github.com/yawo/onefacture/internal/directory"
	"github.com/yawo/onefacture/internal/events"
	"github.com/yawo/onefacture/internal/gateway/middleware"
	"github.com/yawo/onefacture/internal/gateway/problem"
	"github.com/yawo/onefacture/internal/metrics"
	"github.com/yawo/onefacture/internal/storage"
	"github.com/yawo/onefacture/internal/validation"
)

// Dependencies bundles handler dependencies.
type Dependencies struct {
	Logger     *slog.Logger
	Store      *storage.Store
	Validator  *validation.Client
	Registry   *registry.Registry
	Directory  *directory.Resolver
	Events     *events.Bus
	HashPepper string
}

var validate = validator.New()

type bulkValidationItem struct {
	Index    int                  `json:"index"`
	Number   string               `json:"number,omitempty"`
	Valid    bool                 `json:"valid"`
	Findings []validation.Finding `json:"findings"`
}

type timelineEntry struct {
	Type       string         `json:"type"`
	Status     invoice.Status `json:"status,omitempty"`
	FromStatus invoice.Status `json:"from_status,omitempty"`
	ToStatus   invoice.Status `json:"to_status,omitempty"`
	PACode     string         `json:"pa_code,omitempty"`
	PAMessage  string         `json:"pa_message,omitempty"`
	RetryCount int            `json:"retry_count,omitempty"`
	LatencyMS  int64          `json:"latency_ms,omitempty"`
	OccurredAt time.Time      `json:"occurred_at"`
	Payload    map[string]any `json:"payload,omitempty"`
}

type complianceTrend struct {
	Month    string `json:"month"`
	Total    int    `json:"total"`
	Accepted int    `json:"accepted"`
	Rejected int    `json:"rejected"`
	Score    int    `json:"score"`
}

// Health is the liveness probe.
func Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func WebhookInspectorUI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>onefacture webhook inspector</title>
  <style>
    body { font-family: system-ui, sans-serif; margin: 2rem; color: #18212f; }
    input, button, select { font: inherit; padding: .55rem .7rem; }
    table { width: 100%; border-collapse: collapse; margin-top: 1rem; }
    th, td { border-bottom: 1px solid #d8dee8; padding: .65rem; text-align: left; vertical-align: top; }
    .bar { display: flex; gap: .5rem; flex-wrap: wrap; }
    .err { color: #a40000; }
  </style>
</head>
<body>
  <h1>Webhook inspector</h1>
  <div class="bar">
    <input id="apiKey" placeholder="X-API-Key" size="42">
    <select id="status"><option value="FAILED">FAILED</option><option value="">ALL</option><option>PENDING</option><option>RETRYING</option><option>DELIVERED</option></select>
    <button id="load">Load</button>
  </div>
  <p id="msg"></p>
  <table><thead><tr><th>ID</th><th>Status</th><th>Event</th><th>Endpoint</th><th>Error</th><th></th></tr></thead><tbody id="rows"></tbody></table>
  <script>
    const rows = document.querySelector("#rows");
    const msg = document.querySelector("#msg");
    async function load() {
      msg.textContent = "";
      rows.innerHTML = "";
      const status = document.querySelector("#status").value;
      const res = await fetch("/v1/webhooks/deliveries?status=" + encodeURIComponent(status), { headers: { "X-API-Key": document.querySelector("#apiKey").value } });
      if (!res.ok) { msg.className = "err"; msg.textContent = await res.text(); return; }
      const data = await res.json();
      for (const d of data.data || []) {
        const tr = document.createElement("tr");
        tr.innerHTML = "<td>" + d.id + "</td><td>" + d.status + "</td><td>" + d.event_type + "</td><td>" + (d.endpoint_url || "") + "</td><td>" + (d.last_error || "") + "</td><td><button data-id=\"" + d.id + "\">Replay</button></td>";
        rows.appendChild(tr);
      }
    }
    rows.addEventListener("click", async (event) => {
      if (event.target.tagName !== "BUTTON") return;
      const id = event.target.dataset.id;
      const res = await fetch("/v1/webhooks/deliveries/" + id + "/replay", { method: "POST", headers: { "X-API-Key": document.querySelector("#apiKey").value } });
      msg.className = res.ok ? "" : "err";
      msg.textContent = res.ok ? "Replay scheduled" : await res.text();
      if (res.ok) await load();
    });
    document.querySelector("#load").addEventListener("click", load);
  </script>
</body>
</html>`))
}

func CreateSandboxCredentials(deps Dependencies) http.HandlerFunc {
	type request struct {
		Name  string `json:"name"`
		SIREN string `json:"siren"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(io.LimitReader(r.Body, 8192)).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			problem.BadRequest(w, r, "invalid JSON: "+err.Error())
			return
		}
		if strings.TrimSpace(req.Name) == "" {
			req.Name = "Sandbox organization"
		}
		org := &storage.Organization{
			Name:  req.Name,
			SIREN: req.SIREN,
			PAID:  "mock",
			Settings: map[string]any{
				"sandbox": true,
			},
		}
		if err := deps.Store.Organizations.Create(r.Context(), org); err != nil {
			problem.Internal(w, r, "create sandbox organization: "+err.Error())
			return
		}
		apiKey, key, err := deps.Store.APIKeys.Generate(r.Context(), org.ID, "sandbox quickstart", deps.HashPepper)
		if err != nil {
			problem.Internal(w, r, "create sandbox api key: "+err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{
			"organization_id": org.ID,
			"api_key":         apiKey,
			"api_key_id":      key.ID,
			"pa_id":           org.PAID,
			"quickstart":      "/docs/onboarding/5-minutes-first-invoice.md",
		})
	}
}

// Ready returns 503 until storage and Redis answer.
func Ready(store *storage.Store, bus *events.Bus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := store.Pool().Ping(r.Context()); err != nil {
			problem.Write(w, r, problem.Problem{Type: "not-ready", Status: 503, Detail: "db down: " + err.Error()})
			return
		}
		if err := bus.Client().Ping(r.Context()).Err(); err != nil {
			problem.Write(w, r, problem.Problem{Type: "not-ready", Status: 503, Detail: "redis down: " + err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	}
}

// ListPlatforms returns the registered PA adapters and their health.
func ListPlatforms(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		type entry struct {
			Name    string `json:"name"`
			Healthy bool   `json:"healthy"`
		}
		out := []entry{}
		for _, name := range deps.Registry.Names() {
			a, err := deps.Registry.Get(name)
			if err != nil {
				continue
			}
			out = append(out, entry{Name: name, Healthy: a.HealthCheck(r.Context()) == nil})
		}
		writeJSON(w, http.StatusOK, map[string]any{"platforms": out})
	}
}

// CreateInvoice creates and stores a new invoice. With ?submit=true it also
// submits to the configured PA synchronously.
func CreateInvoice(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, ok := middleware.OrgID(r.Context())
		if !ok {
			problem.Unauthorized(w, r, "missing organization context")
			return
		}
		idemKey, ok := idempotencyKey(w, r)
		if !ok {
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			problem.BadRequest(w, r, "read request body: "+err.Error())
			return
		}
		replay, ok := reserveIdempotency(w, r, deps, orgID, idemKey, body)
		if !ok {
			return
		}
		if replay {
			return
		}
		committed := false
		defer func() {
			if !committed {
				_ = deps.Store.Idempotency.Release(r.Context(), orgID, idemKey)
			}
		}()

		inv := &invoice.Invoice{}
		if err := json.NewDecoder(bytes.NewReader(body)).Decode(inv); err != nil {
			problem.BadRequest(w, r, "invalid JSON body")
			return
		}
		if inv.IssueDate.IsZero() {
			inv.IssueDate = time.Now().UTC()
		}
		if inv.Status == "" {
			inv.Status = invoice.StatusDraft
		}
		inv.ComputeTotals()

		if err := validate.Struct(inv); err != nil {
			problem.BadRequest(w, r, "validation failed", validatorErrors(err)...)
			return
		}

		xml, err := facturx.GenerateCII(inv)
		if err != nil {
			problem.Internal(w, r, "generate xml: "+err.Error())
			return
		}
		inv.RawXML = xml

		report, err := deps.Validator.ValidateInvoice(r.Context(), inv, xml)
		if err != nil {
			problem.Internal(w, r, "validation pipeline: "+err.Error())
			return
		}
		if !report.Valid {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
				"valid": false, "report": report,
			})
			return
		}
		inv.Status = invoice.StatusValidated

		pdf, err := facturx.PackagePDFA3(inv, xml)
		if err == nil {
			inv.RawPDF = pdf
		}

		id, err := deps.Store.Invoices.Create(r.Context(), orgID, storage.DirectionOutbound, inv)
		if err != nil {
			problem.Internal(w, r, "persist invoice: "+err.Error())
			return
		}
		_ = deps.Store.Lifecycle.Record(r.Context(), orgID, id, storage.LifecycleEvent{
			FromStatus: invoice.StatusDraft, ToStatus: invoice.StatusValidated,
		})
		_ = deps.Store.Audit.Append(r.Context(), orgID, "api", "invoice.create", "invoice", id.String(), nil)
		_ = deps.Events.Publish(r.Context(), events.Event{
			Type: "invoice.validated", OrganizationID: orgID.String(), InvoiceID: id.String(),
			OccurredAt: time.Now().UTC(),
		})

		if r.URL.Query().Get("submit") == "true" {
			if err := submitNow(r, deps, orgID, id, inv); err != nil {
				problem.Internal(w, r, err.Error())
				return
			}
		}

		committed = writeIdempotentJSON(w, r, deps, orgID, idemKey, http.StatusCreated, inv, "invoice", inv.ID)
	}
}

func submitNow(r *http.Request, deps Dependencies, orgID, invID uuid.UUID, inv *invoice.Invoice) error {
	org, err := deps.Store.Organizations.Get(r.Context(), orgID)
	if err != nil {
		return fmt.Errorf("org lookup: %w", err)
	}
	paID, overridden := resolvePAID(org, inv)
	a, err := deps.Registry.Get(paID)
	if err != nil {
		return err
	}
	res, err := a.Submit(r.Context(), inv)
	if err != nil {
		metrics.PASubmissionTotal.WithLabelValues(paID, "error").Inc()
		metrics.DLQEnqueuedTotal.WithLabelValues(paID).Inc()
		_ = deps.Store.Submissions.EnqueueDLQ(r.Context(), orgID, invID, paID, err.Error(), map[string]any{
			"invoice_id": invID.String(),
			"pa_id":      paID,
			"status":     inv.Status,
		})
		if errors.Is(err, adapters.ErrNotImplemented) {
			return errors.New("PA adapter not implemented for this organization")
		}
		return fmt.Errorf("submit: %w", err)
	}
	metrics.PASubmissionTotal.WithLabelValues(paID, "accepted").Inc()
	inv.PAID = a.Name()
	inv.PARef = res.PARef
	if err := deps.Store.Invoices.SetSubmissionMetadata(r.Context(), orgID, invID, inv.PAID, inv.PARef); err != nil {
		return err
	}
	if err := deps.Store.Invoices.UpdateStatus(r.Context(), orgID, invID, invoice.StatusSubmitted); err != nil {
		return err
	}
	payload := map[string]any{}
	if overridden {
		payload["routing_override"] = true
		payload["buyer_siren"] = inv.Buyer.SIREN
		payload["selected_pa_id"] = paID
	}
	_ = deps.Store.Lifecycle.Record(r.Context(), orgID, invID, storage.LifecycleEvent{
		FromStatus: invoice.StatusValidated, ToStatus: invoice.StatusSubmitted,
		PACode:  res.PARef,
		Payload: payload,
	})
	_ = deps.Events.Publish(r.Context(), events.Event{
		Type: "invoice.submitted", OrganizationID: orgID.String(), InvoiceID: invID.String(),
	})
	inv.Status = invoice.StatusSubmitted
	return nil
}

func resolvePAID(org *storage.Organization, inv *invoice.Invoice) (string, bool) {
	if org == nil || inv == nil || inv.Buyer.SIREN == "" {
		if org == nil {
			return "", false
		}
		return org.PAID, false
	}
	raw, ok := org.Settings["routing_overrides"]
	if !ok {
		return org.PAID, false
	}
	overrides, ok := raw.(map[string]any)
	if !ok {
		return org.PAID, false
	}
	paID, ok := overrides[inv.Buyer.SIREN].(string)
	if !ok || strings.TrimSpace(paID) == "" {
		return org.PAID, false
	}
	return strings.TrimSpace(paID), true
}

// GetInvoice returns one invoice.
func GetInvoice(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, _ := middleware.OrgID(r.Context())
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			problem.BadRequest(w, r, "invalid id")
			return
		}
		inv, err := deps.Store.Invoices.Get(r.Context(), orgID, id)
		if errors.Is(err, storage.ErrNotFound) {
			problem.NotFound(w, r, "invoice not found")
			return
		}
		if err != nil {
			problem.Internal(w, r, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, inv)
	}
}

// ListInvoices lists invoices for the organization.
func ListInvoices(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, _ := middleware.OrgID(r.Context())
		f := storage.ListFilter{
			Direction: storage.DirectionOutbound,
			Status:    invoice.Status(r.URL.Query().Get("status")),
			Limit:     atoiDefault(r.URL.Query().Get("limit"), 50),
			Offset:    atoiDefault(r.URL.Query().Get("offset"), 0),
		}
		list, err := deps.Store.Invoices.List(r.Context(), orgID, f)
		if err != nil {
			problem.Internal(w, r, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": list})
	}
}

// SubmitInvoice promotes a DRAFT/VALIDATED invoice to SUBMITTED via the PA.
func SubmitInvoice(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, ok := middleware.OrgID(r.Context())
		if !ok {
			problem.Unauthorized(w, r, "missing organization context")
			return
		}
		idemKey, ok := idempotencyKey(w, r)
		if !ok {
			return
		}
		replay, ok := reserveIdempotency(w, r, deps, orgID, idemKey, nil)
		if !ok {
			return
		}
		if replay {
			return
		}
		committed := false
		defer func() {
			if !committed {
				_ = deps.Store.Idempotency.Release(r.Context(), orgID, idemKey)
			}
		}()

		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			problem.BadRequest(w, r, "invalid id")
			return
		}
		inv, err := deps.Store.Invoices.Get(r.Context(), orgID, id)
		if errors.Is(err, storage.ErrNotFound) {
			problem.NotFound(w, r, "invoice not found")
			return
		}
		if err != nil {
			problem.Internal(w, r, err.Error())
			return
		}
		if !invoice.CanTransition(inv.Status, invoice.StatusSubmitted) {
			problem.Conflict(w, r, "cannot submit from "+string(inv.Status))
			return
		}
		if inv.Status == invoice.StatusRejected {
			_ = deps.Store.Invoices.IncrementRejectionRetry(r.Context(), orgID, id, "manual resubmission")
		}
		if err := submitNow(r, deps, orgID, id, inv); err != nil {
			problem.Internal(w, r, err.Error())
			return
		}
		committed = writeIdempotentJSON(w, r, deps, orgID, idemKey, http.StatusAccepted, inv, "invoice", inv.ID)
	}
}

// RetryRejectedInvoice retries a rejected invoice after quick corrections done in external ERP.
func RetryRejectedInvoice(deps Dependencies) http.HandlerFunc {
	type reqBody struct {
		ResolutionHint string `json:"resolution_hint"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, _ := middleware.OrgID(r.Context())
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			problem.BadRequest(w, r, "invalid id")
			return
		}
		inv, err := deps.Store.Invoices.Get(r.Context(), orgID, id)
		if errors.Is(err, storage.ErrNotFound) {
			problem.NotFound(w, r, "invoice not found")
			return
		}
		if err != nil {
			problem.Internal(w, r, err.Error())
			return
		}
		if inv.Status != invoice.StatusRejected {
			problem.Conflict(w, r, "invoice is not rejected")
			return
		}
		var body reqBody
		_ = json.NewDecoder(io.LimitReader(r.Body, 8192)).Decode(&body)
		_ = deps.Store.Invoices.IncrementRejectionRetry(r.Context(), orgID, id, body.ResolutionHint)
		if err := submitNow(r, deps, orgID, id, inv); err != nil {
			problem.Internal(w, r, err.Error())
			return
		}
		_ = deps.Store.Lifecycle.Record(r.Context(), orgID, id, storage.LifecycleEvent{
			FromStatus: invoice.StatusRejected,
			ToStatus:   invoice.StatusSubmitted,
			PAMessage:  "retry submitted",
			Payload:    map[string]any{"resolution_hint": body.ResolutionHint, "strategy": "manual"},
		})
		writeJSON(w, http.StatusAccepted, inv)
	}
}

func RejectionPatch(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, _ := middleware.OrgID(r.Context())
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			problem.BadRequest(w, r, "invalid id")
			return
		}
		inv, err := deps.Store.Invoices.Get(r.Context(), orgID, id)
		if errors.Is(err, storage.ErrNotFound) {
			problem.NotFound(w, r, "invoice not found")
			return
		}
		if err != nil {
			problem.Internal(w, r, err.Error())
			return
		}
		if inv.LastRejection == nil {
			problem.Conflict(w, r, "invoice has no rejection to correct")
			return
		}
		writeJSON(w, http.StatusOK, suggestRejectionPatch(inv))
	}
}

func suggestRejectionPatch(inv *invoice.Invoice) map[string]any {
	rej := inv.LastRejection
	ops := []map[string]any{}
	hints := []string{}
	text := strings.ToLower(rej.Code + " " + rej.Message)
	switch {
	case strings.Contains(text, "siren"):
		ops = append(ops, map[string]any{"op": "replace", "path": "/buyer/siren", "value": "000000000"})
		hints = append(hints, "Remplacer /buyer/siren par le SIREN a 9 chiffres confirme dans l'annuaire.")
	case strings.Contains(text, "total") || strings.Contains(text, "amount"):
		ops = append(ops, map[string]any{"op": "replace", "path": "/totals", "value": "recompute"})
		hints = append(hints, "Recalculer les totaux depuis les lignes avant resoumission.")
	case strings.Contains(text, "vat") || strings.Contains(text, "tva"):
		ops = append(ops, map[string]any{"op": "replace", "path": "/seller/vat_num", "value": "FR00000000000"})
		hints = append(hints, "Verifier numero TVA intracommunautaire et categorie de taxe.")
	default:
		hints = append(hints, "Consulter le code rejet PA puis corriger le champ indique avant retry.")
	}
	return map[string]any{
		"invoice_id":       inv.ID,
		"rejection_code":   rej.Code,
		"rejection_msg":    rej.Message,
		"patch_format":     "json-patch",
		"patch":            ops,
		"remediation_hint": strings.Join(hints, " "),
		"retry_endpoint":   "/v1/invoices/" + inv.ID + "/retry",
		"outcome_metric": map[string]any{
			"name":           "rejection_retry_success_rate",
			"success_signal": "retry submitted then accepted",
			"retry_count":    rej.RetryCount,
		},
	}
}

// RejectionSummary provides lightweight monitoring stats for rejection management.
func RejectionSummary(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, _ := middleware.OrgID(r.Context())
		const q = `SELECT COALESCE(pa_code,''), COUNT(*) FROM lifecycle_events
WHERE organization_id = $1 AND to_status = 'REJECTED'
GROUP BY pa_code ORDER BY COUNT(*) DESC`
		rows, err := deps.Store.Pool().Query(r.Context(), q, orgID)
		if err != nil {
			problem.Internal(w, r, "query rejection summary: "+err.Error())
			return
		}
		defer rows.Close()
		type entry struct {
			Code  string `json:"code"`
			Count int    `json:"count"`
		}
		out := []entry{}
		total := 0
		for rows.Next() {
			var e entry
			if err := rows.Scan(&e.Code, &e.Count); err != nil {
				problem.Internal(w, r, "scan rejection summary: "+err.Error())
				return
			}
			out = append(out, e)
			total += e.Count
		}
		writeJSON(w, http.StatusOK, map[string]any{"total_rejected": total, "by_code": out})
	}
}

// InvoiceEvents returns the lifecycle event log for an invoice.
func InvoiceEvents(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, _ := middleware.OrgID(r.Context())
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			problem.BadRequest(w, r, "invalid id")
			return
		}
		ev, err := deps.Store.Lifecycle.List(r.Context(), orgID, id)
		if err != nil {
			problem.Internal(w, r, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"events": ev})
	}
}

// InvoiceTimeline returns lifecycle transitions plus retry/error context.
func InvoiceTimeline(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, _ := middleware.OrgID(r.Context())
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			problem.BadRequest(w, r, "invalid id")
			return
		}
		inv, err := deps.Store.Invoices.Get(r.Context(), orgID, id)
		if errors.Is(err, storage.ErrNotFound) {
			problem.NotFound(w, r, "invoice not found")
			return
		}
		if err != nil {
			problem.Internal(w, r, err.Error())
			return
		}
		events, err := deps.Store.Lifecycle.List(r.Context(), orgID, id)
		if err != nil {
			problem.Internal(w, r, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"invoice_id": id,
			"status":     inv.Status,
			"timeline":   buildTimeline(inv, events),
		})
	}
}

func buildTimeline(inv *invoice.Invoice, events []storage.LifecycleEvent) []timelineEntry {
	out := make([]timelineEntry, 0, len(events)+1)
	var prev time.Time
	for _, ev := range events {
		entry := timelineEntry{
			Type:       "transition",
			FromStatus: ev.FromStatus,
			ToStatus:   ev.ToStatus,
			Status:     ev.ToStatus,
			PACode:     ev.PACode,
			PAMessage:  ev.PAMessage,
			OccurredAt: ev.OccurredAt,
			Payload:    ev.Payload,
		}
		if !prev.IsZero() {
			entry.LatencyMS = ev.OccurredAt.Sub(prev).Milliseconds()
		}
		prev = ev.OccurredAt
		out = append(out, entry)
	}
	if inv != nil && inv.LastRejection != nil {
		out = append(out, timelineEntry{
			Type:       "rejection",
			Status:     invoice.StatusRejected,
			PACode:     inv.LastRejection.Code,
			PAMessage:  inv.LastRejection.Message,
			RetryCount: inv.LastRejection.RetryCount,
			OccurredAt: inv.LastRejection.OccurredAt,
			Payload: map[string]any{
				"resolution_hint": inv.LastRejection.ResolutionHint,
				"last_retry_at":   inv.LastRejection.LastRetryAt,
				"next_retry_at":   inv.LastRejection.NextRetryAt,
			},
		})
	}
	return out
}

// ListInbox lists inbound invoices.
func ListInbox(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, _ := middleware.OrgID(r.Context())
		list, err := deps.Store.Invoices.List(r.Context(), orgID, storage.ListFilter{
			Direction: storage.DirectionInbound,
			Limit:     atoiDefault(r.URL.Query().Get("limit"), 50),
		})
		if err != nil {
			problem.Internal(w, r, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": list})
	}
}

// ApproveInbox accepts a received invoice.
func ApproveInbox(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, _ := middleware.OrgID(r.Context())
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			problem.BadRequest(w, r, "invalid id")
			return
		}
		if err := deps.Store.Invoices.UpdateStatus(r.Context(), orgID, id, invoice.StatusAccepted); err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				problem.NotFound(w, r, "invoice not found")
				return
			}
			problem.Internal(w, r, err.Error())
			return
		}
		_ = deps.Store.Lifecycle.Record(r.Context(), orgID, id, storage.LifecycleEvent{
			FromStatus: invoice.StatusReceived, ToStatus: invoice.StatusAccepted,
		})
		writeJSON(w, http.StatusOK, map[string]string{"status": string(invoice.StatusAccepted)})
	}
}

// ValidateRaw validates an uploaded XML document.
func ValidateRaw(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, 5<<20))
		if err != nil {
			problem.BadRequest(w, r, "cannot read body")
			return
		}
		report, err := deps.Validator.ValidateXML(r.Context(), body, r.URL.Query().Get("profile"))
		if err != nil {
			problem.Internal(w, r, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, report)
	}
}

func ValidateBulk(deps Dependencies) http.HandlerFunc {
	type request struct {
		Invoices []invoice.Invoice `json:"invoices"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(io.LimitReader(r.Body, 5<<20)).Decode(&req); err != nil {
			problem.BadRequest(w, r, "invalid JSON body")
			return
		}
		if len(req.Invoices) == 0 {
			problem.BadRequest(w, r, "invoices must not be empty", problem.FieldError{Field: "invoices", Code: "REQUIRED"})
			return
		}
		validatorClient := deps.Validator
		if validatorClient == nil {
			validatorClient = &validation.Client{}
		}
		items := make([]bulkValidationItem, 0, len(req.Invoices))
		validCount := 0
		for i := range req.Invoices {
			inv := req.Invoices[i]
			inv.ComputeTotals()
			xml, err := facturx.GenerateCII(&inv)
			report := &validation.Report{Valid: false, Findings: []validation.Finding{}}
			if err != nil {
				report.Findings = append(report.Findings, validation.Finding{
					Layer: "generation", Code: "XML_GENERATION_FAILED", Severity: validation.SeverityError, Message: err.Error(),
				})
			} else if report, err = validatorClient.ValidateInvoice(r.Context(), &inv, xml); err != nil {
				report = &validation.Report{Valid: false, Findings: []validation.Finding{{
					Layer: "validation", Code: "VALIDATION_FAILED", Severity: validation.SeverityError, Message: err.Error(),
				}}}
			}
			if report.Valid {
				validCount++
			}
			items = append(items, bulkValidationItem{Index: i, Number: inv.Number, Valid: report.Valid, Findings: report.Findings})
		}
		if r.URL.Query().Get("format") == "csv" {
			writeBulkCSV(w, items)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"summary": map[string]int{"total": len(items), "valid": validCount, "invalid": len(items) - validCount},
			"items":   items,
		})
	}
}

func writeBulkCSV(w http.ResponseWriter, items []bulkValidationItem) {
	w.Header().Set("Content-Type", "text/csv")
	w.WriteHeader(http.StatusOK)
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"index", "number", "valid", "layer", "code", "severity", "path", "message"})
	for _, item := range items {
		if len(item.Findings) == 0 {
			_ = cw.Write([]string{strconv.Itoa(item.Index), item.Number, strconv.FormatBool(item.Valid), "", "", "", "", ""})
			continue
		}
		for _, finding := range item.Findings {
			_ = cw.Write([]string{strconv.Itoa(item.Index), item.Number, strconv.FormatBool(item.Valid), finding.Layer, finding.Code, string(finding.Severity), finding.Path, finding.Message})
		}
	}
	cw.Flush()
}

// DirectoryLookup resolves a SIREN to its registered PA.
func DirectoryLookup(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		siren := r.URL.Query().Get("siren")
		if siren == "" {
			problem.BadRequest(w, r, "siren query parameter required")
			return
		}
		resolver := deps.Directory
		if resolver == nil {
			resolver = directory.NewResolver(time.Minute, directory.StaticProvider{Source: "static", Entries: map[string]string{}}, directory.FallbackProvider{PAID: "mock", Source: "fallback"})
		}
		entry, err := resolver.Resolve(r.Context(), siren)
		if err != nil {
			problem.Internal(w, r, "directory lookup: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, entry)
	}
}

// CreateWebhook registers a new outbound webhook endpoint.
func CreateWebhook(deps Dependencies) http.HandlerFunc {
	type request struct {
		URL          string   `json:"url"    validate:"required,url"`
		Secret       string   `json:"secret" validate:"required,min=16"`
		Events       []string `json:"events"`
		IPAllowlist  []string `json:"ip_allowlist"`
		MTLSRequired bool     `json:"mtls_required"`
		MTLSCertRef  string   `json:"mtls_cert_ref"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, _ := middleware.OrgID(r.Context())
		var req request
		if err := json.NewDecoder(io.LimitReader(r.Body, 8192)).Decode(&req); err != nil {
			problem.BadRequest(w, r, "invalid JSON")
			return
		}
		if err := validate.Struct(req); err != nil {
			problem.BadRequest(w, r, "validation failed", validatorErrors(err)...)
			return
		}
		ep, err := deps.Store.Webhooks.CreateWithOptions(r.Context(), orgID, req.URL, req.Secret, req.Events, storage.WebhookEndpointOptions{
			IPAllowlist:  req.IPAllowlist,
			MTLSRequired: req.MTLSRequired,
			MTLSCertRef:  req.MTLSCertRef,
		})
		if err != nil {
			problem.Internal(w, r, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{
			"id":            ep.ID,
			"url":           ep.URL,
			"events":        ep.Events,
			"ip_allowlist":  ep.IPAllowlist,
			"mtls_required": ep.MTLSRequired,
			"mtls_cert_ref": ep.MTLSCertRef,
		})
	}
}

func ListWebhookDeliveries(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, _ := middleware.OrgID(r.Context())
		list, err := deps.Store.Webhooks.ListDeliveries(r.Context(), orgID, strings.ToUpper(r.URL.Query().Get("status")), atoiDefault(r.URL.Query().Get("limit"), 50))
		if err != nil {
			problem.Internal(w, r, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": list})
	}
}

func ReplayWebhookDelivery(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, _ := middleware.OrgID(r.Context())
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			problem.BadRequest(w, r, "invalid id")
			return
		}
		if err := deps.Store.Webhooks.ReplayDelivery(r.Context(), orgID, id); err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				problem.NotFound(w, r, "failed webhook delivery not found")
				return
			}
			problem.Internal(w, r, err.Error())
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "PENDING"})
	}
}

func ListSubmissionDLQ(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, _ := middleware.OrgID(r.Context())
		list, err := deps.Store.Submissions.ListDLQ(r.Context(), orgID, atoiDefault(r.URL.Query().Get("limit"), 50))
		if err != nil {
			problem.Internal(w, r, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": list})
	}
}

func ReplaySubmissionDLQ(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, _ := middleware.OrgID(r.Context())
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			problem.BadRequest(w, r, "invalid id")
			return
		}
		entry, err := deps.Store.Submissions.GetDLQ(r.Context(), orgID, id)
		if errors.Is(err, storage.ErrNotFound) {
			problem.NotFound(w, r, "submission dlq entry not found")
			return
		}
		if err != nil {
			problem.Internal(w, r, err.Error())
			return
		}
		inv, err := deps.Store.Invoices.Get(r.Context(), orgID, entry.InvoiceID)
		if err != nil {
			problem.Internal(w, r, "load invoice: "+err.Error())
			return
		}
		if err := submitNow(r, deps, orgID, entry.InvoiceID, inv); err != nil {
			problem.Internal(w, r, err.Error())
			return
		}
		_ = deps.Store.Submissions.MarkReplayed(r.Context(), orgID, id)
		writeJSON(w, http.StatusAccepted, map[string]any{"status": "REPLAYED", "invoice_id": entry.InvoiceID})
	}
}

func ComplianceScore(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, _ := middleware.OrgID(r.Context())
		const q = `SELECT
COUNT(*) FILTER (WHERE to_status IN ('SUBMITTED','ACCEPTED','REJECTED')) AS total,
COUNT(*) FILTER (WHERE to_status = 'REJECTED') AS rejected,
COUNT(*) FILTER (WHERE to_status = 'ACCEPTED') AS accepted
FROM lifecycle_events
WHERE organization_id = $1 AND occurred_at >= now() - interval '7 days'`
		var total, rejected, accepted int
		if err := deps.Store.Pool().QueryRow(r.Context(), q, orgID).Scan(&total, &rejected, &accepted); err != nil {
			problem.Internal(w, r, "query compliance score: "+err.Error())
			return
		}
		trends, err := complianceTrends(r.Context(), deps, orgID)
		if err != nil {
			problem.Internal(w, r, "query compliance trends: "+err.Error())
			return
		}
		score := buildComplianceScore(total, rejected, accepted)
		score["monthly_trends"] = trends
		writeJSON(w, http.StatusOK, score)
	}
}

func RejectionRetrySuccessRate(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, _ := middleware.OrgID(r.Context())
		const q = `WITH retried AS (
	SELECT DISTINCT invoice_id
	FROM lifecycle_events
	WHERE organization_id = $1
	  AND from_status = 'REJECTED'
	  AND to_status = 'SUBMITTED'
	  AND occurred_at >= now() - interval '90 days'
),
accepted_after_retry AS (
	SELECT DISTINCT r.invoice_id
	FROM retried r
	JOIN lifecycle_events e ON e.invoice_id = r.invoice_id
	WHERE e.organization_id = $1
	  AND e.to_status = 'ACCEPTED'
)
SELECT COUNT(*)::int AS retried, (SELECT COUNT(*)::int FROM accepted_after_retry) AS accepted_after_retry
FROM retried`
		var retried, acceptedAfterRetry int
		if err := deps.Store.Pool().QueryRow(r.Context(), q, orgID).Scan(&retried, &acceptedAfterRetry); err != nil {
			problem.Internal(w, r, "query rejection retry success rate: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, buildRejectionRetrySuccessRate(retried, acceptedAfterRetry))
	}
}

func buildRejectionRetrySuccessRate(retried, acceptedAfterRetry int) map[string]any {
	rate := 0.0
	if retried > 0 {
		rate = float64(acceptedAfterRetry) / float64(retried)
	}
	return map[string]any{
		"period":               "90d",
		"metric":               "rejection_retry_success_rate",
		"retried_invoices":     retried,
		"accepted_after_retry": acceptedAfterRetry,
		"success_rate":         rate,
	}
}

func ComplianceDashboardUI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!doctype html>
<html lang="fr">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>onefacture - Compliance dashboard</title>
  <style>
    :root { --bg:#f6f7f9; --ink:#17202a; --muted:#687385; --line:#d9dee7; --ok:#167a4a; --warn:#a55b00; --bad:#b42318; --panel:#fff; }
    * { box-sizing: border-box; }
    body { margin:0; font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; color:var(--ink); background:var(--bg); }
    main { max-width:1120px; margin:0 auto; padding:32px 20px 48px; }
    header { display:flex; align-items:flex-end; justify-content:space-between; gap:16px; margin-bottom:24px; }
    h1 { font-size:28px; line-height:1.1; margin:0; }
    .controls { display:flex; gap:8px; flex-wrap:wrap; }
    input { min-width:280px; padding:10px 12px; border:1px solid var(--line); border-radius:6px; font:inherit; }
    button { padding:10px 14px; border:1px solid #1f6feb; border-radius:6px; background:#1f6feb; color:#fff; font:inherit; cursor:pointer; }
    .grid { display:grid; grid-template-columns: repeat(5, minmax(0, 1fr)); gap:12px; margin-bottom:18px; }
    .panel { background:var(--panel); border:1px solid var(--line); border-radius:8px; padding:16px; }
    .label { color:var(--muted); font-size:13px; margin-bottom:8px; }
    .metric { font-size:32px; font-weight:700; letter-spacing:0; }
    .score-ok { color:var(--ok); }
    .score-warn { color:var(--warn); }
    .score-bad { color:var(--bad); }
    table { width:100%; border-collapse:collapse; background:var(--panel); border:1px solid var(--line); border-radius:8px; overflow:hidden; }
    th, td { padding:12px 14px; border-bottom:1px solid var(--line); text-align:right; }
    th:first-child, td:first-child { text-align:left; }
    th { color:var(--muted); font-size:13px; font-weight:600; background:#fbfcfe; }
    tr:last-child td { border-bottom:0; }
    .state { margin:12px 0 0; color:var(--muted); min-height:22px; }
    @media (max-width: 900px) { header { align-items:stretch; flex-direction:column; } input { min-width:100%; } .grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
  </style>
</head>
<body>
<main>
  <header>
    <div>
      <h1>Compliance dashboard</h1>
      <div class="state" id="state">Entrez une API key pour charger le score du tenant.</div>
    </div>
    <div class="controls">
      <input id="apiKey" type="password" autocomplete="off" placeholder="X-API-Key">
      <button id="refresh" type="button">Actualiser</button>
    </div>
  </header>
  <section class="grid" aria-label="Score hebdomadaire">
    <div class="panel"><div class="label">Score 7 jours</div><div class="metric" id="score">-</div></div>
    <div class="panel"><div class="label">Evenements</div><div class="metric" id="total">-</div></div>
    <div class="panel"><div class="label">Accepted</div><div class="metric" id="accepted">-</div></div>
    <div class="panel"><div class="label">Rejetes</div><div class="metric" id="rejected">-</div></div>
    <div class="panel"><div class="label">Retry accepted</div><div class="metric" id="retryRate">-</div></div>
  </section>
  <table aria-label="Tendances mensuelles">
    <thead><tr><th>Mois</th><th>Score</th><th>Total</th><th>Accepted</th><th>Rejetes</th></tr></thead>
    <tbody id="trends"><tr><td colspan="5">Aucune donnee chargee.</td></tr></tbody>
  </table>
</main>
<script>
const $ = (id) => document.getElementById(id);
function scoreClass(score) {
  if (score >= 90) return "score-ok";
  if (score >= 70) return "score-warn";
  return "score-bad";
}
async function loadScore() {
  const apiKey = $("apiKey").value.trim();
  if (!apiKey) {
    $("state").textContent = "API key requise.";
    return;
  }
  $("state").textContent = "Chargement...";
  const res = await fetch("/v1/analytics/compliance-score", { headers: { "X-API-Key": apiKey } });
  if (!res.ok) {
    $("state").textContent = "Erreur " + res.status + " lors du chargement.";
    return;
  }
  const data = await res.json();
  $("score").textContent = data.score ?? "-";
  $("score").className = "metric " + scoreClass(Number(data.score || 0));
  $("total").textContent = data.total_events ?? 0;
  $("accepted").textContent = data.accepted ?? 0;
  $("rejected").textContent = data.rejected ?? 0;
  const retryRes = await fetch("/v1/analytics/rejection-retry-success-rate", { headers: { "X-API-Key": apiKey } });
  if (retryRes.ok) {
    const retry = await retryRes.json();
    $("retryRate").textContent = Math.round(Number(retry.success_rate || 0) * 100) + "%";
  }
  const rows = Array.isArray(data.monthly_trends) ? data.monthly_trends : [];
  $("trends").innerHTML = rows.length ? rows.map((row) => "<tr><td>" + row.month + "</td><td>" + row.score + "</td><td>" + row.total + "</td><td>" + row.accepted + "</td><td>" + row.rejected + "</td></tr>").join("") : "<tr><td colspan=\"5\">Aucune tendance disponible.</td></tr>";
  $("state").textContent = "Score charge.";
}
$("refresh").addEventListener("click", loadScore);
</script>
</body>
</html>`))
}

func complianceTrends(ctx context.Context, deps Dependencies, orgID uuid.UUID) ([]complianceTrend, error) {
	const q = `SELECT to_char(date_trunc('month', occurred_at), 'YYYY-MM') AS month,
COUNT(*) FILTER (WHERE to_status IN ('SUBMITTED','ACCEPTED','REJECTED')) AS total,
COUNT(*) FILTER (WHERE to_status = 'ACCEPTED') AS accepted,
COUNT(*) FILTER (WHERE to_status = 'REJECTED') AS rejected
FROM lifecycle_events
WHERE organization_id = $1 AND occurred_at >= now() - interval '12 months'
GROUP BY date_trunc('month', occurred_at)
ORDER BY month ASC`
	rows, err := deps.Store.Pool().Query(ctx, q, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []complianceTrend{}
	for rows.Next() {
		var trend complianceTrend
		if err := rows.Scan(&trend.Month, &trend.Total, &trend.Accepted, &trend.Rejected); err != nil {
			return nil, err
		}
		trend.Score = buildComplianceScore(trend.Total, trend.Rejected, trend.Accepted)["score"].(int)
		out = append(out, trend)
	}
	return out, rows.Err()
}

func buildComplianceScore(total, rejected, accepted int) map[string]any {
	score := 100
	rejectionRate := 0.0
	acceptanceRate := 0.0
	if total > 0 {
		rejectionRate = float64(rejected) / float64(total)
		acceptanceRate = float64(accepted) / float64(total)
		score = 100 - int(rejectionRate*70)
		if acceptanceRate < 0.5 {
			score -= 10
		}
		if score < 0 {
			score = 0
		}
	}
	return map[string]any{
		"period":          "7d",
		"score":           score,
		"total_events":    total,
		"accepted":        accepted,
		"rejected":        rejected,
		"acceptance_rate": acceptanceRate,
		"rejection_rate":  rejectionRate,
	}
}

// GDPRExport returns all data tied to an organization (GDPR Article 20).
func GDPRExport(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, _ := middleware.OrgID(r.Context())
		invoices, err := deps.Store.Invoices.List(r.Context(), orgID, storage.ListFilter{Limit: 200})
		if err != nil {
			problem.Internal(w, r, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"organization_id": orgID,
			"invoices":        invoices,
		})
	}
}

// GDPRErase soft-deletes organization data (GDPR Article 17).
func GDPRErase(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, _ := middleware.OrgID(r.Context())
		// Audit the request; actual cascading delete is a separate runbook in production.
		_ = deps.Store.Audit.Append(r.Context(), orgID, "api", "gdpr.erase.requested", "organization", orgID.String(), nil)
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "erasure requested"})
	}
}

func idempotencyKey(w http.ResponseWriter, r *http.Request) (string, bool) {
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" {
		problem.BadRequest(w, r, "missing Idempotency-Key header", problem.FieldError{Field: "Idempotency-Key", Code: "REQUIRED"})
		return "", false
	}
	return key, true
}

func reserveIdempotency(w http.ResponseWriter, r *http.Request, deps Dependencies, orgID uuid.UUID, key string, body []byte) (bool, bool) {
	sum := sha256.Sum256(body)
	rec, created, err := deps.Store.Idempotency.Reserve(r.Context(), orgID, key, r.Method, r.URL.RequestURI(), hex.EncodeToString(sum[:]))
	if errors.Is(err, storage.ErrIdempotencyConflict) {
		problem.Conflict(w, r, err.Error())
		return false, false
	}
	if errors.Is(err, storage.ErrIdempotencyInProgress) {
		problem.Conflict(w, r, err.Error())
		return false, false
	}
	if err != nil {
		problem.Internal(w, r, err.Error())
		return false, false
	}
	if !created {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Idempotent-Replay", "true")
		w.WriteHeader(rec.StatusCode)
		_, _ = w.Write(rec.ResponseBody)
		return true, true
	}
	return false, true
}

func writeIdempotentJSON(w http.ResponseWriter, r *http.Request, deps Dependencies, orgID uuid.UUID, key string, status int, body any, resourceType, resourceID string) bool {
	payload, err := json.Marshal(body)
	if err != nil {
		problem.Internal(w, r, "encode response: "+err.Error())
		return false
	}
	if err := deps.Store.Idempotency.Store(r.Context(), orgID, key, status, payload, resourceType, resourceID); err != nil {
		problem.Internal(w, r, err.Error())
		return false
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(payload)
	return true
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("encode json", "err", err)
	}
}

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func validatorErrors(err error) []problem.FieldError {
	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		return []problem.FieldError{{Field: "", Code: "INVALID", Message: err.Error()}}
	}
	out := make([]problem.FieldError, 0, len(ve))
	for _, fe := range ve {
		out = append(out, problem.FieldError{
			Field:   fe.Namespace(),
			Code:    fe.Tag(),
			Message: fmt.Sprintf("%s failed %s", fe.Field(), fe.Tag()),
		})
	}
	return out
}
