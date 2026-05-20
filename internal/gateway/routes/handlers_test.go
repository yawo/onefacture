package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yawo/onefacture/internal/adapters/registry"
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
