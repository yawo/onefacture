package facturx

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yawo/onefacture/internal/core/invoice"
)

func TestPackagePDFA3_DelegatesToSidecar(t *testing.T) {
	// Fake sidecar that returns a minimal PDF
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Return a fake base64 PDF
		fakePDF := base64.StdEncoding.EncodeToString([]byte("%PDF-1.7\nfake sidecar pdf"))
		w.Write([]byte(`{"pdf_base64":"` + fakePDF + `","filename":"test.pdf"}`))
	}))
	defer ts.Close()

	// Set env to trigger delegation
	os.Setenv("ONEFACTURE_PDF_SIDECAR_URL", ts.URL)
	defer os.Unsetenv("ONEFACTURE_PDF_SIDECAR_URL")

	inv := &invoice.Invoice{
		Number:  "INV-SIDECAR-001",
		Profile: invoice.ProfileEN16931,
		Seller:  invoice.Party{Name: "Seller"},
		Buyer:   invoice.Party{Name: "Buyer"},
	}
	xml := []byte(`<?xml version="1.0"?><CrossIndustryInvoice></CrossIndustryInvoice>`)

	pdf, err := PackagePDFA3(inv, xml)
	require.NoError(t, err)
	require.Contains(t, string(pdf), "%PDF-1.7")
	require.Contains(t, string(pdf), "fake sidecar pdf")
}
