package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDoctorReportsReachabilityAndAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/healthz", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	results := doctor(context.Background(), server.Client(), server.URL, "of_test")

	require.Len(t, results, 3)
	require.True(t, results[0].OK)
	require.True(t, results[1].OK)
	require.True(t, results[2].OK)
}

func TestDoctorReportsMissingAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	results := doctor(context.Background(), server.Client(), server.URL, "")

	require.False(t, results[0].OK)
	require.Contains(t, results[0].Detail, "missing")
	require.True(t, results[1].OK)
}

func TestValidateMinimalInvoicePayload(t *testing.T) {
	result := validateMinimalInvoicePayload()

	require.True(t, result.OK)
	require.Equal(t, "payload_schema", result.Name)
}

func TestFormatDoctorReportShowsClearTerminalStatus(t *testing.T) {
	report, ok := formatDoctorReport([]checkResult{
		{Name: "api_key", OK: true, Detail: "ONEFACTURE_API_KEY is set"},
		{Name: "reachability", OK: false, Detail: "GET http://localhost:8080/healthz returned 503"},
	})

	require.False(t, ok)
	require.Contains(t, report, "[ok] api_key: ONEFACTURE_API_KEY is set")
	require.Contains(t, report, "[fail] reachability: GET http://localhost:8080/healthz returned 503")
}
