package facturx

import (
	"bytes"
	"fmt"
	"time"

	"github.com/yawo/onefacture/internal/core/invoice"
)

// PackagePDFA3 returns a PDF/A-3 byte stream embedding the provided CII XML.
// A production implementation delegates to the Python sidecar (which uses
// pdfa3 + factur-x python lib). Here we emit a minimal placeholder so the
// pipeline is wire-complete and unit-testable end-to-end. The sidecar takes
// over when ONEFACTURE_SIDECAR_URL is set.
func PackagePDFA3(inv *invoice.Invoice, xml []byte) ([]byte, error) {
	if inv == nil {
		return nil, fmt.Errorf("nil invoice")
	}
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%%PDF-1.7\n%% onefacture placeholder PDF/A-3\n")
	fmt.Fprintf(&buf, "%% generated_at=%s\n", time.Now().UTC().Format(time.RFC3339))
	fmt.Fprintf(&buf, "%% invoice_number=%s\n", inv.Number)
	fmt.Fprintf(&buf, "%% profile=%s\n", inv.Profile)
	fmt.Fprintf(&buf, "%% cii_xml_size=%d\n", len(xml))
	// Trailing marker so we can detect this placeholder in tests.
	fmt.Fprintf(&buf, "%%%%EOF\n")
	return buf.Bytes(), nil
}
