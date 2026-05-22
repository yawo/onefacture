package routes

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	validator "github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/require"
)

func TestAtoiDefault(t *testing.T) {
	tests := []struct {
		input    string
		fallback int
		expected int
	}{
		{"42", 0, 42},
		{"0", 100, 0},
		{"999", 1, 999},
		{"", 50, 50},
		{"invalid", 99, 99},
		{"12.5", 10, 10},
		{"-10", 0, -10},
	}

	for _, tt := range tests {
		result := atoiDefault(tt.input, tt.fallback)
		require.Equal(t, tt.expected, result, "atoiDefault(%q, %d)", tt.input, tt.fallback)
	}
}

func TestValidatorErrors(t *testing.T) {
	type TestStruct struct {
		Email string `validate:"required,email"`
		Name  string `validate:"required,min=3"`
	}

	validator := validator.New()

	// Test with validation errors
	obj := TestStruct{Email: "not-an-email", Name: "ab"}
	err := validator.Struct(obj)
	require.Error(t, err)

	fieldErrors := validatorErrors(err)
	require.Greater(t, len(fieldErrors), 0)

	// Check that errors contain expected fields
	var hasEmailError, hasNameError bool
	for _, fe := range fieldErrors {
		if fe.Field == "TestStruct.Email" {
			hasEmailError = true
			require.Equal(t, "email", fe.Code)
		}
		if fe.Field == "TestStruct.Name" {
			hasNameError = true
			require.Equal(t, "min", fe.Code)
		}
	}
	require.True(t, hasEmailError || hasNameError)
}

func TestValidatorErrorsNonValidationError(t *testing.T) {
	// Test with non-validation error
	err := errors.New("some other error")
	fieldErrors := validatorErrors(err)

	require.Len(t, fieldErrors, 1)
	require.Equal(t, "", fieldErrors[0].Field)
	require.Equal(t, "INVALID", fieldErrors[0].Code)
	require.Equal(t, "some other error", fieldErrors[0].Message)
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()

	data := map[string]interface{}{
		"status": "ok",
		"value":  42,
	}

	writeJSON(w, http.StatusOK, data)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var result map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&result)
	require.NoError(t, err)
	require.Equal(t, "ok", result["status"])
	require.Equal(t, float64(42), result["value"]) // JSON decodes numbers as float64
}

func TestWriteJSONComplexStruct(t *testing.T) {
	type Response struct {
		ID    string   `json:"id"`
		Count int      `json:"count"`
		Items []string `json:"items"`
	}

	w := httptest.NewRecorder()
	resp := Response{
		ID:    "test-123",
		Count: 3,
		Items: []string{"a", "b", "c"},
	}

	writeJSON(w, http.StatusCreated, resp)

	require.Equal(t, http.StatusCreated, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var result Response
	err := json.NewDecoder(w.Body).Decode(&result)
	require.NoError(t, err)
	require.Equal(t, "test-123", result.ID)
	require.Equal(t, 3, result.Count)
	require.Len(t, result.Items, 3)
}

func TestDirectoryLookup(t *testing.T) {
	// Test the DirectoryLookup handler behavior
	deps := Dependencies{} // minimal deps for this test
	handler := DirectoryLookup(deps)

	// Test with SIREN query
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/directory/lookup?siren=123456789", nil)
	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var result map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&result)
	require.NoError(t, err)
	require.Equal(t, "123456789", result["siren"])
	require.Equal(t, false, result["resolved"]) // It's a stub
}

func TestDirectoryLookupMissingSIREN(t *testing.T) {
	deps := Dependencies{}
	handler := DirectoryLookup(deps)

	// Test without SIREN query
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/directory/lookup", nil)
	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestValidatorErrorsStructure(t *testing.T) {
	type TestInput struct {
		Email string `validate:"required,email"`
	}

	v := validator.New()
	obj := TestInput{Email: "invalid"}
	err := v.Struct(obj)
	require.Error(t, err)

	fieldErrors := validatorErrors(err)
	require.NotEmpty(t, fieldErrors)

	// Each error should have the expected structure
	for _, fe := range fieldErrors {
		require.NotEmpty(t, fe.Code)
		require.NotEmpty(t, fe.Message)
	}
}
