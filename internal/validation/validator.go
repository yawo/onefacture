// Package validation orchestrates the 6-layer Factur-X validation pipeline.
// Layers 1-3 (PDF/A-3 container, XML extraction, XSD) and 4-5 (Schematron EN16931 + AFNOR)
// are delegated to the Python sidecar (lxml/saxon-style); layer 6 (business rules) is local.
package validation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/yawo/onefacture/internal/config"
	"github.com/yawo/onefacture/internal/core/invoice"
)

// Severity of a validation finding.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// Finding is a single rule violation.
type Finding struct {
	Layer    string   `json:"layer"`
	Code     string   `json:"code"`
	Severity Severity `json:"severity"`
	Path     string   `json:"path,omitempty"`
	Message  string   `json:"message"`
}

// Report aggregates all findings.
type Report struct {
	Valid    bool      `json:"valid"`
	Profile  string    `json:"profile,omitempty"`
	Findings []Finding `json:"findings"`
}

// Client talks to the Python validation sidecar and runs local business rules.
type Client struct {
	http *http.Client
	base string
}

// NewClient returns a validation client.
func NewClient(cfg config.SidecarConfig) *Client {
	return &Client{
		http: &http.Client{Timeout: cfg.Timeout},
		base: cfg.BaseURL,
	}
}

// ValidateInvoice runs the local business-rule layer plus the sidecar XSD/Schematron
// validation against a generated XML for the given invoice.
func (c *Client) ValidateInvoice(ctx context.Context, inv *invoice.Invoice, xml []byte) (*Report, error) {
	report := &Report{Findings: []Finding{}}
	report.Findings = append(report.Findings, runBusinessRules(inv)...)

	sidecarReport, err := c.ValidateXML(ctx, xml, string(inv.Profile))
	if err != nil {
		// Sidecar unavailable: surface a non-fatal warning rather than blocking the call.
		report.Findings = append(report.Findings, Finding{
			Layer:    "sidecar",
			Code:     "SIDECAR_UNAVAILABLE",
			Severity: SeverityWarning,
			Message:  fmt.Sprintf("validation sidecar unreachable: %v", err),
		})
	} else {
		report.Findings = append(report.Findings, sidecarReport.Findings...)
	}

	report.Profile = string(inv.Profile)
	report.Valid = !hasErrors(report.Findings)
	return report, nil
}

// ValidateXML sends a raw XML document to the sidecar for XSD + Schematron validation.
func (c *Client) ValidateXML(ctx context.Context, xml []byte, profile string) (*Report, error) {
	if c.base == "" {
		return nil, fmt.Errorf("sidecar not configured")
	}
	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	if profile != "" {
		_ = mw.WriteField("profile", profile)
	}
	fw, err := mw.CreateFormFile("file", "invoice.xml")
	if err != nil {
		return nil, fmt.Errorf("multipart: %w", err)
	}
	if _, err := io.Copy(fw, bytes.NewReader(xml)); err != nil {
		return nil, fmt.Errorf("multipart copy: %w", err)
	}
	if err := mw.Close(); err != nil {
		return nil, fmt.Errorf("multipart close: %w", err)
	}

	reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, c.base+"/v1/validate/xml", body)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sidecar request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("sidecar status %d: %s", resp.StatusCode, string(raw))
	}
	var r Report
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("decode report: %w", err)
	}
	return &r, nil
}

func hasErrors(findings []Finding) bool {
	for _, f := range findings {
		if f.Severity == SeverityError {
			return true
		}
	}
	return false
}
