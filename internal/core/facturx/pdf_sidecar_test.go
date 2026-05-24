package facturx

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yawo/onefacture/internal/core/invoice"
)

func TestPackagePDFA3_DelegatesToSidecar(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		var req struct {
			InvoiceNumber string `json:"invoice_number"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if req.InvoiceNumber == "" {
			req.InvoiceNumber = "INV-SIDECAR-001"
		}
		content := "%PDF-1.7\n%% onefacture\n/Title (Facture " + req.InvoiceNumber + ")\n/pdfaid:part 3\n/pdfaid:conformance A\n factur-x.xml\n" + req.InvoiceNumber + "\n%%EOF\n"
		fakePDF := base64.StdEncoding.EncodeToString([]byte(content))
		w.Write([]byte(`{"pdf_base64":"` + fakePDF + `","filename":"test.pdf"}`))
	}))
	defer ts.Close()

	os.Setenv("ONEFACTURE_PDF_SIDECAR_URL", ts.URL)
	defer os.Unsetenv("ONEFACTURE_PDF_SIDECAR_URL")

	inv := &invoice.Invoice{
		Number:  "INV-SIDECAR-001",
		Profile: invoice.ProfileEN16931,
		Seller:  invoice.Party{Name: "Seller"},
		Buyer:   invoice.Party{Name: "Buyer"},
		Lines: []invoice.Line{{Description: "Consulting", Quantity: 10, UnitPrice: 150, NetAmount: 1500}},
		Totals: invoice.Totals{TaxExclusiveAmount: 1500, TaxInclusiveAmount: 1800, TaxBreakdown: []invoice.TaxSubtotal{{Rate: 20, TaxableBase: 1500, TaxAmount: 300}}},
	}
	xml := []byte(`<?xml version="1.0"?><CrossIndustryInvoice></CrossIndustryInvoice>`)

	pdf, err := PackagePDFA3(inv, xml)
	require.NoError(t, err)
	s := string(pdf)
	require.Contains(t, s, "%PDF-1.7")
	require.Contains(t, s, "INV-SIDECAR-001")
	require.Contains(t, s, "pdfaid:part")
	require.Contains(t, s, "factur-x.xml")
}
