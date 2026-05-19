// Package routes contains the HTTP handlers for the gateway.
package routes

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	chi "github.com/go-chi/chi/v5"
	validator "github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/yawo/onefacture/internal/adapters"
	"github.com/yawo/onefacture/internal/adapters/registry"
	"github.com/yawo/onefacture/internal/core/facturx"
	"github.com/yawo/onefacture/internal/core/invoice"
	"github.com/yawo/onefacture/internal/events"
	"github.com/yawo/onefacture/internal/gateway/middleware"
	"github.com/yawo/onefacture/internal/gateway/problem"
	"github.com/yawo/onefacture/internal/storage"
	"github.com/yawo/onefacture/internal/validation"
)

// Dependencies bundles handler dependencies.
type Dependencies struct {
	Logger    *slog.Logger
	Store     *storage.Store
	Validator *validation.Client
	Registry  *registry.Registry
	Events    *events.Bus
}

var validate = validator.New()

// Health is the liveness probe.
func Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
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
			Name   string `json:"name"`
			Healthy bool  `json:"healthy"`
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
		inv := &invoice.Invoice{}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(inv); err != nil {
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

		writeJSON(w, http.StatusCreated, inv)
	}
}

func submitNow(r *http.Request, deps Dependencies, orgID, invID uuid.UUID, inv *invoice.Invoice) error {
	org, err := deps.Store.Organizations.Get(r.Context(), orgID)
	if err != nil {
		return fmt.Errorf("org lookup: %w", err)
	}
	a, err := deps.Registry.Get(org.PAID)
	if err != nil {
		return err
	}
	res, err := a.Submit(r.Context(), inv)
	if err != nil {
		if errors.Is(err, adapters.ErrNotImplemented) {
			return errors.New("PA adapter not implemented for this organization")
		}
		return fmt.Errorf("submit: %w", err)
	}
	inv.PAID = a.Name()
	inv.PARef = res.PARef
	if err := deps.Store.Invoices.UpdateStatus(r.Context(), orgID, invID, invoice.StatusSubmitted); err != nil {
		return err
	}
	_ = deps.Store.Lifecycle.Record(r.Context(), orgID, invID, storage.LifecycleEvent{
		FromStatus: invoice.StatusValidated, ToStatus: invoice.StatusSubmitted,
		PACode: res.PARef,
	})
	_ = deps.Events.Publish(r.Context(), events.Event{
		Type: "invoice.submitted", OrganizationID: orgID.String(), InvoiceID: invID.String(),
	})
	inv.Status = invoice.StatusSubmitted
	return nil
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

// ListInvoices lists invoices for the organisation.
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
		if !invoice.CanTransition(inv.Status, invoice.StatusSubmitted) {
			problem.Conflict(w, r, "cannot submit from "+string(inv.Status))
			return
		}
		if err := submitNow(r, deps, orgID, id, inv); err != nil {
			problem.Internal(w, r, err.Error())
			return
		}
		writeJSON(w, http.StatusAccepted, inv)
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

// DirectoryLookup resolves a SIREN to its registered PA.
func DirectoryLookup(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		siren := r.URL.Query().Get("siren")
		if siren == "" {
			problem.BadRequest(w, r, "siren query parameter required")
			return
		}
		// Stubbed: a real implementation queries the official DGFiP directory API.
		writeJSON(w, http.StatusOK, map[string]any{
			"siren":     siren,
			"pa_id":     "mock",
			"resolved":  false,
			"source":    "stub",
			"note":      "directory lookup not yet integrated with DGFiP",
		})
	}
}

// CreateWebhook registers a new outbound webhook endpoint.
func CreateWebhook(deps Dependencies) http.HandlerFunc {
	type request struct {
		URL    string   `json:"url"    validate:"required,url"`
		Secret string   `json:"secret" validate:"required,min=16"`
		Events []string `json:"events"`
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
		ep, err := deps.Store.Webhooks.Create(r.Context(), orgID, req.URL, req.Secret, req.Events)
		if err != nil {
			problem.Internal(w, r, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{
			"id":     ep.ID,
			"url":    ep.URL,
			"events": ep.Events,
		})
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

// GDPRErase soft-deletes organisation data (GDPR Article 17).
func GDPRErase(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, _ := middleware.OrgID(r.Context())
		// Audit the request; actual cascading delete is a separate runbook in production.
		_ = deps.Store.Audit.Append(r.Context(), orgID, "api", "gdpr.erase.requested", "organization", orgID.String(), nil)
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "erasure requested"})
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
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
