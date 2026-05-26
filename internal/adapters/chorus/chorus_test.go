package chorus

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yawo/onefacture/internal/adapters"
	"github.com/yawo/onefacture/internal/adapters/sandbox"
	"github.com/yawo/onefacture/internal/core/invoice"
)

func TestNew(t *testing.T) {
	// Clear env vars
	os.Unsetenv("ONEFACTURE_CHORUS_CLIENT_ID")
	os.Unsetenv("ONEFACTURE_CHORUS_CLIENT_SECRET")
	os.Unsetenv("ONEFACTURE_CHORUS_BASE_URL")

	adapter := New()
	require.NotNil(t, adapter)
	require.Equal(t, "", adapter.clientID)
	require.Equal(t, "", adapter.clientSecret)
	require.Equal(t, "https://sandbox-api.piste.gouv.fr/cpro", adapter.baseURL)
}

func TestNewWithEnv(t *testing.T) {
	os.Setenv("ONEFACTURE_CHORUS_CLIENT_ID", "test-client-id")
	os.Setenv("ONEFACTURE_CHORUS_CLIENT_SECRET", "test-client-secret")
	os.Setenv("ONEFACTURE_CHORUS_BASE_URL", "https://custom-url.example.com")
	defer func() {
		os.Unsetenv("ONEFACTURE_CHORUS_CLIENT_ID")
		os.Unsetenv("ONEFACTURE_CHORUS_CLIENT_SECRET")
		os.Unsetenv("ONEFACTURE_CHORUS_BASE_URL")
	}()

	adapter := New()
	require.Equal(t, "test-client-id", adapter.clientID)
	require.Equal(t, "test-client-secret", adapter.clientSecret)
	require.Equal(t, "https://custom-url.example.com", adapter.baseURL)
	require.Equal(t, "test-client-id", adapter.client.Auth.ClientID)
	require.Equal(t, "test-client-secret", adapter.client.Auth.ClientSecret)
	require.NotEmpty(t, adapter.client.Auth.TokenURL)
}

func TestName(t *testing.T) {
	adapter := New()
	require.Equal(t, "chorus", adapter.Name())
}

func TestHealthCheckWithoutCredentials(t *testing.T) {
	os.Unsetenv("ONEFACTURE_CHORUS_CLIENT_ID")
	os.Unsetenv("ONEFACTURE_CHORUS_CLIENT_SECRET")

	adapter := New()
	err := adapter.HealthCheck(context.Background())
	require.Equal(t, adapters.ErrNotImplemented, err)
}

func TestHealthCheckWithCredentials(t *testing.T) {
	os.Setenv("ONEFACTURE_CHORUS_CLIENT_ID", "test-id")
	os.Setenv("ONEFACTURE_CHORUS_CLIENT_SECRET", "test-secret")
	defer func() {
		os.Unsetenv("ONEFACTURE_CHORUS_CLIENT_ID")
		os.Unsetenv("ONEFACTURE_CHORUS_CLIENT_SECRET")
	}()

	adapter := New()
	err := adapter.HealthCheck(context.Background())
	require.NoError(t, err)
}

func TestHealthCheckPartialCredentials(t *testing.T) {
	os.Setenv("ONEFACTURE_CHORUS_CLIENT_ID", "test-id")
	os.Unsetenv("ONEFACTURE_CHORUS_CLIENT_SECRET")
	defer func() {
		os.Unsetenv("ONEFACTURE_CHORUS_CLIENT_ID")
	}()

	adapter := New()
	err := adapter.HealthCheck(context.Background())
	require.Equal(t, adapters.ErrNotImplemented, err)
}

func TestSubmit(t *testing.T) {
	adapter := New()
	inv := &invoice.Invoice{}
	result, err := adapter.Submit(context.Background(), inv)
	require.Nil(t, result)
	require.Equal(t, adapters.ErrNotImplemented, err)
}

func TestGetStatus(t *testing.T) {
	adapter := New()
	event, err := adapter.GetStatus(context.Background(), "test-ref")
	require.Nil(t, event)
	require.Equal(t, adapters.ErrNotImplemented, err)
}

func TestWebhook(t *testing.T) {
	adapter := New()
	event, err := adapter.Webhook(context.Background(), []byte("{}"))
	require.Nil(t, event)
	require.Equal(t, adapters.ErrNotImplemented, err)
}

func TestEnvOr(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		fallback string
		expected string
	}{
		{"value set", "TEST_KEY_SET", "custom-value", "default", "custom-value"},
		{"value not set", "TEST_KEY_UNSET", "", "default", "default"},
		{"empty value", "TEST_KEY_EMPTY", "", "fallback", "fallback"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv(tt.key)
			if tt.value != "" {
				os.Setenv(tt.key, tt.value)
				defer os.Unsetenv(tt.key)
			}
			result := envOr(tt.key, tt.fallback)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestChorusIntegrationNormalizeLifecycle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/invoices" {
			_ = json.NewEncoder(w).Encode(adapters.SubmitResult{PARef: "cpp-123", Status: "DEPOSEE"})
			return
		}
		if strings.HasSuffix(r.URL.Path, "/status") {
			_ = json.NewEncoder(w).Encode(adapters.LifecycleEvent{PARef: "cpp-123", Status: "MISE_A_DISPOSITION"})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	c := sandbox.Client{Name: "chorus", BaseURL: server.URL, SubmitPath: "/invoices", StatusPath: "/invoices/{pa_ref}/status", Auth: sandbox.Auth{Token: "t"}, HTTP: server.Client()}
	a := &Adapter{client: c}
	res, err := a.Submit(context.Background(), &invoice.Invoice{Number: "i1"})
	require.NoError(t, err)
	require.Equal(t, invoice.StatusSubmitted, res.Status)
	ev, err := a.GetStatus(context.Background(), "cpp-123")
	require.NoError(t, err)
	require.Equal(t, invoice.StatusAccepted, ev.Status)

	require.Equal(t, "SUBMITTED", NormalizeLifecycleStatus("DEPOSEE"))
	require.Equal(t, "ACCEPTED", NormalizeLifecycleStatus("MISE_A_DISPOSITION"))
	require.Equal(t, "REJECTED", NormalizeLifecycleStatus("REJETEE"))
	require.Equal(t, "SUBMITTED", NormalizeLifecycleStatus("SUSPENDUE"))
}

func TestChorusIntegrationRealPISTEShapes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/cpro/factures/v1/soumettre" {
			_ = json.NewEncoder(w).Encode(map[string]any{"identifiantFactureCPP": 504114, "statutFacture": "DEPOSEE", "numeroFacture": "INV-001"})
			return
		}
		if r.URL.Path == "/cpro/factures/v1/consulter/fournisseur" {
			_ = json.NewEncoder(w).Encode(map[string]any{"identifiantFactureCPP": 504114, "statutFacture": "MISE_A_DISPOSITION"})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	c := sandbox.Client{Name: "chorus", BaseURL: server.URL, SubmitPath: "/cpro/factures/v1/soumettre", StatusPath: "/cpro/factures/v1/consulter/fournisseur", StatusMethod: "POST", StatusBodyTemplate: `{"identifiantFactureCPP":"{pa_ref}"}`, Auth: sandbox.Auth{Token: "t"}, HTTP: server.Client()}
	a := &Adapter{client: c}
	res, err := a.Submit(context.Background(), &invoice.Invoice{Number: "i1"})
	require.NoError(t, err)
	require.Equal(t, "504114", res.PARef)
	require.Equal(t, invoice.StatusSubmitted, res.Status)
	ev, err := a.GetStatus(context.Background(), "504114")
	require.NoError(t, err)
	require.Equal(t, "504114", ev.PARef)
	require.Equal(t, invoice.StatusAccepted, ev.Status)
}

func TestChorusSubmitWithPDFMultipart(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cpro/factures/v1/soumettre" {
			http.NotFound(w, r)
			return
		}
		contentType := r.Header.Get("Content-Type")
		require.Contains(t, contentType, "multipart/form-data")
		mr, err := r.MultipartReader()
		require.NoError(t, err)
		var invoicePart string
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			require.NoError(t, err)
			if part.FormName() == "invoice" {
				b, _ := io.ReadAll(part)
				invoicePart = string(b)
			}
			if part.FormName() == "file" {
				b, _ := io.ReadAll(part)
				require.Equal(t, []byte("PDFBYTES"), b)
			}
		}
		require.NotEmpty(t, invoicePart)
		var meta map[string]any
		require.NoError(t, json.Unmarshal([]byte(invoicePart), &meta))
		// respond with PISTE shaped JSON
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"identifiantFactureCPP": 9001, "statutFacture": "DEPOSEE"})
	}))
	defer server.Close()
	c := sandbox.Client{Name: "chorus", BaseURL: server.URL, SubmitPath: "/cpro/factures/v1/soumettre", Auth: sandbox.Auth{Token: "t"}, HTTP: server.Client()}
	a := &Adapter{client: c}
	inv := &invoice.Invoice{Number: "i-pdf"}
	inv.RawPDF = []byte("PDFBYTES")
	res, err := a.Submit(context.Background(), inv)
	require.NoError(t, err)
	require.Equal(t, "9001", res.PARef)
	require.Equal(t, invoice.StatusSubmitted, res.Status)
}
