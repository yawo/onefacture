package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type checkResult struct {
	Name   string
	OK     bool
	Detail string
}

func main() {
	if len(os.Args) < 2 || os.Args[1] != "doctor" {
		fmt.Fprintln(os.Stderr, "usage: onefacture doctor")
		os.Exit(2)
	}
	baseURL := strings.TrimRight(env("ONEFACTURE_BASE_URL", "http://localhost:8080"), "/")
	apiKey := os.Getenv("ONEFACTURE_API_KEY")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results := doctor(ctx, http.DefaultClient, baseURL, apiKey)
	report, ok := formatDoctorReport(results)
	fmt.Print(report)
	if !ok {
		os.Exit(1)
	}
}

func formatDoctorReport(results []checkResult) (string, bool) {
	var out strings.Builder
	ok := true
	for _, result := range results {
		status := "ok"
		if !result.OK {
			status = "fail"
			ok = false
		}
		fmt.Fprintf(&out, "[%s] %s: %s\n", status, result.Name, result.Detail)
	}
	return out.String(), ok
}

func doctor(ctx context.Context, client *http.Client, baseURL, apiKey string) []checkResult {
	results := []checkResult{
		{Name: "api_key", OK: strings.TrimSpace(apiKey) != "", Detail: "ONEFACTURE_API_KEY is set"},
		validateMinimalInvoicePayload(),
	}
	if strings.TrimSpace(apiKey) == "" {
		results[0].Detail = "ONEFACTURE_API_KEY is missing"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/healthz", nil)
	if err != nil {
		return append(results, checkResult{Name: "reachability", OK: false, Detail: err.Error()})
	}
	resp, err := client.Do(req)
	if err != nil {
		return append(results, checkResult{Name: "reachability", OK: false, Detail: err.Error()})
	}
	defer func() { _ = resp.Body.Close() }()
	results = append(results, checkResult{
		Name:   "reachability",
		OK:     resp.StatusCode >= 200 && resp.StatusCode < 300,
		Detail: fmt.Sprintf("GET %s/healthz returned %d", baseURL, resp.StatusCode),
	})
	return results
}

func validateMinimalInvoicePayload() checkResult {
	var payload map[string]any
	if err := json.Unmarshal([]byte(minimalInvoicePayload), &payload); err != nil {
		return checkResult{Name: "payload_schema", OK: false, Detail: "minimal payload is invalid JSON: " + err.Error()}
	}
	required := []string{"profile", "type_code", "number", "currency", "issue_date", "seller", "buyer", "lines"}
	for _, field := range required {
		if _, ok := payload[field]; !ok {
			return checkResult{Name: "payload_schema", OK: false, Detail: "minimal payload missing " + field}
		}
	}
	lines, ok := payload["lines"].([]any)
	if !ok || len(lines) == 0 {
		return checkResult{Name: "payload_schema", OK: false, Detail: "minimal payload requires at least one line"}
	}
	return checkResult{Name: "payload_schema", OK: true, Detail: "minimal invoice payload shape is valid"}
}

const minimalInvoicePayload = `{
  "profile": "EN16931",
  "type_code": "380",
  "number": "INV-DOCTOR-001",
  "currency": "EUR",
  "issue_date": "2026-05-22T00:00:00Z",
  "seller": {"name": "Acme SAS", "address": {"line1": "1 rue Cler", "postal_code": "75007", "city": "Paris", "country_code": "FR"}},
  "buyer": {"name": "Globex SAS", "address": {"line1": "2 avenue Foch", "postal_code": "75116", "city": "Paris", "country_code": "FR"}},
  "lines": [{"description": "Diagnostic", "quantity": 1, "unit_code": "C62", "unit_price": 1, "tax_rate": 20, "tax_category": "S"}]
}`

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
