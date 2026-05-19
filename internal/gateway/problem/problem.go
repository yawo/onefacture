// Package problem implements RFC 7807 problem+json responses.
package problem

import (
	"encoding/json"
	"net/http"
)

const baseType = "https://onefacture.io/errors/"

// Problem is the RFC 7807 payload.
type Problem struct {
	Type     string         `json:"type"`
	Title    string         `json:"title"`
	Status   int            `json:"status"`
	Detail   string         `json:"detail,omitempty"`
	Instance string         `json:"instance,omitempty"`
	Errors   []FieldError   `json:"errors,omitempty"`
	Extra    map[string]any `json:"-"`
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
	} else if p.Type[:4] != "http" {
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

// BadRequest is a 400 helper.
func BadRequest(w http.ResponseWriter, r *http.Request, detail string, errs ...FieldError) {
	Write(w, r, Problem{Type: "validation-failed", Title: "Validation Failed", Status: http.StatusBadRequest, Detail: detail, Errors: errs})
}

// Unauthorized is a 401 helper.
func Unauthorized(w http.ResponseWriter, r *http.Request, detail string) {
	Write(w, r, Problem{Type: "unauthorized", Title: "Unauthorized", Status: http.StatusUnauthorized, Detail: detail})
}

// Forbidden is a 403 helper.
func Forbidden(w http.ResponseWriter, r *http.Request, detail string) {
	Write(w, r, Problem{Type: "forbidden", Title: "Forbidden", Status: http.StatusForbidden, Detail: detail})
}

// NotFound is a 404 helper.
func NotFound(w http.ResponseWriter, r *http.Request, detail string) {
	Write(w, r, Problem{Type: "not-found", Title: "Not Found", Status: http.StatusNotFound, Detail: detail})
}

// Conflict is a 409 helper.
func Conflict(w http.ResponseWriter, r *http.Request, detail string) {
	Write(w, r, Problem{Type: "conflict", Title: "Conflict", Status: http.StatusConflict, Detail: detail})
}

// Internal is a 500 helper.
func Internal(w http.ResponseWriter, r *http.Request, detail string) {
	Write(w, r, Problem{Type: "internal", Title: "Internal Server Error", Status: http.StatusInternalServerError, Detail: detail})
}

// TooMany is a 429 helper.
func TooMany(w http.ResponseWriter, r *http.Request, detail string) {
	Write(w, r, Problem{Type: "rate-limited", Title: "Too Many Requests", Status: http.StatusTooManyRequests, Detail: detail})
}

// NotImplemented is a 501 helper.
func NotImplemented(w http.ResponseWriter, r *http.Request, detail string) {
	Write(w, r, Problem{Type: "not-implemented", Title: "Not Implemented", Status: http.StatusNotImplemented, Detail: detail})
}
