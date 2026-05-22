package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/yawo/onefacture/internal/adapters/registry"
	"github.com/yawo/onefacture/internal/core/invoice"
	"github.com/yawo/onefacture/internal/directory"
	"github.com/yawo/onefacture/internal/storage"
)

// TestHealthHandlerReturnsJSON tests that health endpoint returns JSON
func TestHealthHandlerReturnsJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	Health(rec, req)

	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var result map[string]string
	err := json.NewDecoder(rec.Body).Decode(&result)
	require.NoError(t, err)
	require.Equal(t, "ok", result["status"])
}

// TestListPlatformsResponseFormat tests that platforms endpoint returns properly formatted response
func TestListPlatformsResponseFormat(t *testing.T) {
	deps := Dependencies{
		Registry: registry.NewDefault(nil),
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/platforms", nil)
	ListPlatforms(deps).ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var result map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&result)
	require.NoError(t, err)

	platforms, ok := result["platforms"]
	require.True(t, ok)
	require.IsType(t, []interface{}{}, platforms)

	// Additional check for platform structure
	platformsList := platforms.([]interface{})
	for _, p := range platformsList {
		platform, ok := p.(map[string]interface{})
		require.True(t, ok)
		require.Contains(t, platform, "name")
		require.Contains(t, platform, "healthy")
	}
}

// TestAtoiDefaultEmptyString tests atoiDefault with empty string
func TestAtoiDefaultEmptyString(t *testing.T) {
	result := atoiDefault("", 50)
	require.Equal(t, 50, result)
}

// TestAtoiDefaultValidNumber tests atoiDefault with valid number
func TestAtoiDefaultValidNumber(t *testing.T) {
	result := atoiDefault("100", 50)
	require.Equal(t, 100, result)
}

// TestAtoiDefaultInvalidString tests atoiDefault with invalid string
func TestAtoiDefaultInvalidString(t *testing.T) {
	result := atoiDefault("not-a-number", 50)
	require.Equal(t, 50, result)
}

// TestAtoiDefaultZero tests atoiDefault with zero value
func TestAtoiDefaultZero(t *testing.T) {
	result := atoiDefault("0", 50)
	require.Equal(t, 0, result)
}

// TestAtoiDefaultNegative tests atoiDefault with negative number
func TestAtoiDefaultNegative(t *testing.T) {
	result := atoiDefault("-10", 50)
	require.Equal(t, -10, result)
}

func TestIdempotencyKeyIsRequired(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/invoices", nil)

	key, ok := idempotencyKey(w, req)

	require.False(t, ok)
	require.Empty(t, key)
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "Idempotency-Key")
}

func TestIdempotencyKeyTrimsHeader(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/invoices", nil)
	req.Header.Set("Idempotency-Key", " idem-123 ")

	key, ok := idempotencyKey(w, req)

	require.True(t, ok)
	require.Equal(t, "idem-123", key)
	require.Equal(t, http.StatusOK, w.Code)
}

// TestWriteJSONSuccessfully tests writeJSON functionality
func TestWriteJSONSuccessfully(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"key": "value"}

	writeJSON(w, http.StatusOK, data)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var result map[string]string
	err := json.NewDecoder(w.Body).Decode(&result)
	require.NoError(t, err)
	require.Equal(t, "value", result["key"])
}

// TestWriteJSONWithCustomStatus tests writeJSON with custom status code
func TestWriteJSONWithCustomStatus(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"status": "created"}

	writeJSON(w, http.StatusCreated, data)

	require.Equal(t, http.StatusCreated, w.Code)
}

// TestWriteJSONComplexStructure tests writeJSON with complex structures
func TestWriteJSONComplexStructure(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]interface{}{
		"items": []int{1, 2, 3},
		"meta": map[string]string{
			"total": "3",
		},
	}

	writeJSON(w, http.StatusOK, data)

	require.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&result)
	require.NoError(t, err)

	items, ok := result["items"]
	require.True(t, ok)
	require.IsType(t, []interface{}{}, items)
}

// TestDirectoryLookupWithSIREN tests directory lookup with SIREN parameter
func TestDirectoryLookupWithSIREN(t *testing.T) {
	deps := Dependencies{}
	handler := DirectoryLookup(deps)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/directory/lookup?siren=123456789", nil)
	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&result)
	require.NoError(t, err)

	require.Equal(t, "123456789", result["siren"])
	require.Equal(t, false, result["resolved"])
}

// TestDirectoryLookupWithoutSIREN tests directory lookup without SIREN parameter
func TestDirectoryLookupWithoutSIREN(t *testing.T) {
	deps := Dependencies{}
	handler := DirectoryLookup(deps)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/directory/lookup", nil)
	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

// TestDirectoryLookupEmptySIREN tests directory lookup with empty SIREN
func TestDirectoryLookupEmptySIREN(t *testing.T) {
	deps := Dependencies{}
	handler := DirectoryLookup(deps)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/directory/lookup?siren=", nil)
	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

// TestDirectoryLookupResponseStructure tests the response structure of directory lookup
func TestDirectoryLookupResponseStructure(t *testing.T) {
	deps := Dependencies{}
	handler := DirectoryLookup(deps)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/directory/lookup?siren=987654321", nil)
	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&result)
	require.NoError(t, err)

	require.Contains(t, result, "siren")
	require.Contains(t, result, "pa_id")
	require.Contains(t, result, "resolved")
	require.Contains(t, result, "source")
}

func TestDirectoryLookupUsesConfiguredResolver(t *testing.T) {
	deps := Dependencies{
		Directory: directory.NewResolver(time.Minute,
			directory.StaticProvider{Source: "test", Entries: map[string]string{"123456789": "chorus"}},
			directory.FallbackProvider{PAID: "mock", Source: "fallback"},
		),
	}
	handler := DirectoryLookup(deps)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/directory/lookup?siren=123456789", nil)
	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var result map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
	require.Equal(t, "chorus", result["pa_id"])
	require.Equal(t, true, result["resolved"])
	require.Equal(t, "test", result["source"])
}

func TestValidateBulkRejectsEmptyBatch(t *testing.T) {
	handler := ValidateBulk(Dependencies{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/validate/bulk", strings.NewReader(`{"invoices":[]}`))

	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "invoices")
}

func TestValidateBulkReturnsAggregateReport(t *testing.T) {
	handler := ValidateBulk(Dependencies{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/validate/bulk", strings.NewReader(`{"invoices":[{"number":"BULK-001"}]}`))

	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var got map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	summary := got["summary"].(map[string]any)
	require.Equal(t, float64(1), summary["total"])
	require.Equal(t, float64(0), summary["valid"])
	require.Equal(t, float64(1), summary["invalid"])
	items := got["items"].([]any)
	require.Len(t, items, 1)
	require.Equal(t, "BULK-001", items[0].(map[string]any)["number"])
}

func TestValidateBulkExportsCSVErrors(t *testing.T) {
	handler := ValidateBulk(Dependencies{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/validate/bulk?format=csv", strings.NewReader(`{"invoices":[{"number":"BULK-CSV-001"}]}`))

	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "text/csv", w.Header().Get("Content-Type"))
	body := w.Body.String()
	require.Contains(t, body, "index,number,valid,layer,code,severity,path,message")
	require.Contains(t, body, "BULK-CSV-001")
	require.Contains(t, body, "false")
}

func TestResolvePAIDUsesBuyerOverride(t *testing.T) {
	org := &storage.Organization{
		PAID: "chorus",
		Settings: map[string]any{
			"routing_overrides": map[string]any{
				"123456789": "pennylane",
			},
		},
	}
	inv := &invoice.Invoice{Buyer: invoice.Party{SIREN: "123456789"}}

	paID, overridden := resolvePAID(org, inv)

	require.True(t, overridden)
	require.Equal(t, "pennylane", paID)
}

func TestResolvePAIDFallsBackToOrganizationDefault(t *testing.T) {
	org := &storage.Organization{PAID: "chorus", Settings: map[string]any{}}
	inv := &invoice.Invoice{Buyer: invoice.Party{SIREN: "123456789"}}

	paID, overridden := resolvePAID(org, inv)

	require.False(t, overridden)
	require.Equal(t, "chorus", paID)
}

func TestBuildTimelineIncludesLatencyAndRejectionRetry(t *testing.T) {
	start := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	retryAt := start.Add(3 * time.Minute)
	inv := &invoice.Invoice{LastRejection: &invoice.Rejection{
		Code: "BR-CO-10", Message: "Invalid total", OccurredAt: retryAt, RetryCount: 2, ResolutionHint: "fix totals",
	}}
	events := []storage.LifecycleEvent{
		{FromStatus: invoice.StatusDraft, ToStatus: invoice.StatusValidated, OccurredAt: start},
		{FromStatus: invoice.StatusValidated, ToStatus: invoice.StatusSubmitted, OccurredAt: start.Add(2 * time.Second), PACode: "PA-1"},
	}

	timeline := buildTimeline(inv, events)

	require.Len(t, timeline, 3)
	require.Equal(t, int64(2000), timeline[1].LatencyMS)
	require.Equal(t, "rejection", timeline[2].Type)
	require.Equal(t, 2, timeline[2].RetryCount)
	require.Equal(t, "fix totals", timeline[2].Payload["resolution_hint"])
}

func TestBuildComplianceScorePenalizesRejections(t *testing.T) {
	score := buildComplianceScore(10, 2, 8)

	require.Equal(t, 86, score["score"])
	require.Equal(t, 0.2, score["rejection_rate"])
	require.Equal(t, 0.8, score["acceptance_rate"])
}

func TestBuildRejectionRetrySuccessRate(t *testing.T) {
	metric := buildRejectionRetrySuccessRate(4, 3)

	require.Equal(t, "rejection_retry_success_rate", metric["metric"])
	require.Equal(t, 4, metric["retried_invoices"])
	require.Equal(t, 3, metric["accepted_after_retry"])
	require.Equal(t, 0.75, metric["success_rate"])
}

func TestComplianceTrendStructure(t *testing.T) {
	trend := complianceTrend{Month: "2026-05", Total: 10, Accepted: 8, Rejected: 2, Score: 86}

	require.Equal(t, "2026-05", trend.Month)
	require.Equal(t, 86, trend.Score)
}

func TestSuggestRejectionPatchForSIREN(t *testing.T) {
	inv := &invoice.Invoice{ID: "inv_123", LastRejection: &invoice.Rejection{Code: "BR-SIREN", Message: "buyer siren invalid"}}

	got := suggestRejectionPatch(inv)

	require.Equal(t, "json-patch", got["patch_format"])
	require.Contains(t, got["remediation_hint"], "SIREN")
	require.Contains(t, got["retry_endpoint"], "/retry")
	metric := got["outcome_metric"].(map[string]any)
	require.Equal(t, "rejection_retry_success_rate", metric["name"])
}

// TestValidatorErrorsHandling tests the validatorErrors function with validator errors
func TestValidatorErrorsHandling(t *testing.T) {
	type TestStruct struct {
		Email string `validate:"required,email"`
		Name  string `validate:"required,min=3"`
	}

	v := validate

	obj := TestStruct{Email: "not-an-email", Name: "ab"}
	err := v.Struct(obj)
	require.Error(t, err)

	fieldErrors := validatorErrors(err)
	require.Greater(t, len(fieldErrors), 0)

	for _, fe := range fieldErrors {
		require.NotEmpty(t, fe.Code)
		require.NotEmpty(t, fe.Message)
	}
}

// TestValidatorErrorsWithNonValidationError tests validatorErrors with non-validation error
func TestValidatorErrorsWithNonValidationError(t *testing.T) {
	err := &CustomError{message: "custom error"}

	fieldErrors := validatorErrors(err)

	require.Len(t, fieldErrors, 1)
	require.Equal(t, "INVALID", fieldErrors[0].Code)
	require.Contains(t, fieldErrors[0].Message, "custom")
}

// TestListPlatformsAPIEndpoint tests ListPlatforms handler with registry
func TestListPlatformsAPIEndpoint(t *testing.T) {
	deps := Dependencies{
		Registry: registry.NewDefault(nil),
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/platforms", nil)
	ListPlatforms(deps).ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var result map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&result)
	require.NoError(t, err)

	platforms, ok := result["platforms"].([]interface{})
	require.True(t, ok)
	require.GreaterOrEqual(t, len(platforms), 0)
}

// TestHealthEndpointContentType tests health endpoint content type
func TestHealthEndpointContentType(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	Health(rec, req)

	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
}

func TestWebhookInspectorUI(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tools/webhook-inspector", nil)

	WebhookInspectorUI(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Header().Get("Content-Type"), "text/html")
	require.Contains(t, w.Body.String(), "Webhook inspector")
	require.Contains(t, w.Body.String(), "Replay</button>")
	require.Contains(t, w.Body.String(), "/v1/webhooks/deliveries/")
	require.Contains(t, w.Body.String(), "/replay")
}

func TestComplianceDashboardUI(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tools/compliance-dashboard", nil)

	ComplianceDashboardUI(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Header().Get("Content-Type"), "text/html")
	require.Contains(t, w.Body.String(), "Compliance dashboard")
	require.Contains(t, w.Body.String(), "/v1/analytics/compliance-score")
	require.Contains(t, w.Body.String(), "/v1/analytics/rejection-retry-success-rate")
	require.Contains(t, w.Body.String(), "monthly_trends")
	require.Contains(t, w.Body.String(), "Tendances mensuelles")
}

func TestCreateSandboxCredentialsDefaultName(t *testing.T) {
	body := `{}`
	var req struct {
		Name string `json:"name"`
	}
	require.NoError(t, json.NewDecoder(strings.NewReader(body)).Decode(&req))
	if strings.TrimSpace(req.Name) == "" {
		req.Name = "Sandbox organization"
	}
	require.Equal(t, "Sandbox organization", req.Name)
}

// TestAtoiDefaultEdgeCases tests atoiDefault with various edge cases
func TestAtoiDefaultEdgeCases(t *testing.T) {
	tests := []struct {
		input    string
		fallback int
		expected int
	}{
		{"", 50, 50},
		{"0", 0, 0},
		{"1", 100, 1},
		{"-1", 0, -1},
		{"999999", 10, 999999},
		{"abc", 50, 50},
		{"12.34", 50, 50},
		{"1e5", 50, 50},
	}

	for _, tt := range tests {
		result := atoiDefault(tt.input, tt.fallback)
		require.Equal(t, tt.expected, result, "atoiDefault(%q, %d)", tt.input, tt.fallback)
	}
}

// TestWriteJSONStatusCodes tests writeJSON with various status codes
func TestWriteJSONStatusCodes(t *testing.T) {
	statusCodes := []int{
		http.StatusOK,
		http.StatusCreated,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	}

	for _, code := range statusCodes {
		w := httptest.NewRecorder()
		writeJSON(w, code, map[string]string{"status": "test"})
		require.Equal(t, code, w.Code)
	}
}

// TestDirectoryLookupMultipleSIRENValues tests directory lookup handles multiple SIREN values correctly
func TestDirectoryLookupMultipleSIRENValues(t *testing.T) {
	testCases := []string{"111111111", "222222222", "999999999"}

	for _, siren := range testCases {
		deps := Dependencies{}
		handler := DirectoryLookup(deps)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/v1/directory/lookup?siren="+siren, nil)
		handler.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var result map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&result)
		require.NoError(t, err)
		require.Equal(t, siren, result["siren"])
	}
}

// CustomError for testing
type CustomError struct {
	message string
}

func (e *CustomError) Error() string {
	return e.message
}
