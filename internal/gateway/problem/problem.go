// Package problem implements RFC 7807 problem+json responses.
package problem

import (
	"encoding/json"
	"net/http"
)

const baseType = "https://onefacture.io/errors/"

// Problem is the RFC 7807 payload.
type Problem struct {
	Type            string         `json:"type"`
	Title           string         `json:"title"`
	Status          int            `json:"status"`
	Detail          string         `json:"detail,omitempty"`
	Instance        string         `json:"instance,omitempty"`
	Errors          []FieldError   `json:"errors,omitempty"`
	RemediationHint string         `json:"remediation_hint,omitempty"`
	DocsURL         string         `json:"docs_url,omitempty"`
	Retryable       *bool          `json:"retryable,omitempty"`
	Extra           map[string]any `json:"-"`
}

type FieldError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message,omitempty"`
}

// Write sends a Problem response.
func Write(w http.ResponseWriter, r *http.Request, p Problem) {
	if p.Type == "" {
		p.Type = baseType + "internal"
	} else if len(p.Type) < 4 || p.Type[:4] != "http" {
		p.Type = baseType + p.Type
	}
	if p.Status == 0 {
		p.Status = http.StatusInternalServerError
	}
	if p.Title == "" {
		p.Title = http.StatusText(p.Status)
	}
	if r != nil {
		p.Instance = r.URL.Path
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(p.Status)
	_ = json.NewEncoder(w).Encode(p)
}

func retryable(v bool) *bool { return &v }

// BadRequest is a 400 helper.
func BadRequest(w http.ResponseWriter, r *http.Request, detail string, errs ...FieldError) {
	Write(w, r, Problem{
		Type:            "validation-failed",
		Title:           "Validation Failed",
		Status:          http.StatusBadRequest,
		Detail:          detail,
		Errors:          errs,
		RemediationHint: "Corrigez les champs en erreur puis renvoyez la requete.",
		DocsURL:         "https://onefacture.io/docs/errors/validation-failed",
		Retryable:       retryable(false),
	})
}

// Unauthorized is a 401 helper.
func Unauthorized(w http.ResponseWriter, r *http.Request, detail string) {
	Write(w, r, Problem{Type: "unauthorized", Title: "Unauthorized", Status: http.StatusUnauthorized, Detail: detail, RemediationHint: "Fournissez une cle API active dans le header X-API-Key.", DocsURL: "https://onefacture.io/docs/auth/api-keys", Retryable: retryable(false)})
}

// Forbidden is a 403 helper.
func Forbidden(w http.ResponseWriter, r *http.Request, detail string) {
	Write(w, r, Problem{Type: "forbidden", Title: "Forbidden", Status: http.StatusForbidden, Detail: detail, RemediationHint: "Verifiez les droits de l'organisation ou la portee de la cle API.", DocsURL: "https://onefacture.io/docs/errors/forbidden", Retryable: retryable(false)})
}

// NotFound is a 404 helper.
func NotFound(w http.ResponseWriter, r *http.Request, detail string) {
	Write(w, r, Problem{Type: "not-found", Title: "Not Found", Status: http.StatusNotFound, Detail: detail, RemediationHint: "Verifiez l'identifiant et l'organisation courante.", DocsURL: "https://onefacture.io/docs/errors/not-found", Retryable: retryable(false)})
}

// Conflict is a 409 helper.
func Conflict(w http.ResponseWriter, r *http.Request, detail string) {
	Write(w, r, Problem{Type: "conflict", Title: "Conflict", Status: http.StatusConflict, Detail: detail, RemediationHint: "Rechargez l'etat courant puis rejouez l'action si elle reste valide.", DocsURL: "https://onefacture.io/docs/errors/conflict", Retryable: retryable(false)})
}

// Internal is a 500 helper.
func Internal(w http.ResponseWriter, r *http.Request, detail string) {
	Write(w, r, Problem{Type: "internal", Title: "Internal Server Error", Status: http.StatusInternalServerError, Detail: detail, RemediationHint: "Reessayez plus tard avec le meme X-Request-ID pour le support.", DocsURL: "https://onefacture.io/docs/errors/internal", Retryable: retryable(true)})
}

// TooMany is a 429 helper.
func TooMany(w http.ResponseWriter, r *http.Request, detail string) {
	Write(w, r, Problem{Type: "rate-limited", Title: "Too Many Requests", Status: http.StatusTooManyRequests, Detail: detail, RemediationHint: "Attendez la fenetre de quota suivante avant de rejouer la requete.", DocsURL: "https://onefacture.io/docs/errors/rate-limited", Retryable: retryable(true)})
}

// NotImplemented is a 501 helper.
func NotImplemented(w http.ResponseWriter, r *http.Request, detail string) {
	Write(w, r, Problem{Type: "not-implemented", Title: "Not Implemented", Status: http.StatusNotImplemented, Detail: detail, RemediationHint: "Utilisez un endpoint supporte ou suivez l'avancement dans le backlog.", DocsURL: "https://onefacture.io/docs/errors/not-implemented", Retryable: retryable(false)})
}
