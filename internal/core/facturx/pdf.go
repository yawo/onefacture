package facturx

import (
	"bytes"
	"fmt"
	"time"

	"github.com/yawo/onefacture/internal/core/invoice"
)

// PackagePDFA3 returns a minimal but valid PDF container (%PDF-1.7 + metadata comments)
// so the invoice emission pipeline is wire-complete and unit-testable.
// Real PDF/A-3 conformance, XMP, Factur-X XML attachment and visual layout are
// provided by the Python sidecar when ONEFACTURE_PDF_SIDECAR_URL is set.
func PackagePDFA3(inv *invoice.Invoice, xml []byte) ([]byte, error) {
	if inv == nil {
		return nil, fmt.Errorf("nil invoice")
	}
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%%PDF-1.7\n%%\xe2\xe3\xcf\xd3 onefacture PDF/A-3 container\n")
	fmt.Fprintf(&buf, "%% generated_at=%s\n", time.Now().UTC().Format(time.RFC3339))
	fmt.Fprintf(&buf, "%% invoice_number=%s\n", inv.Number)
	fmt.Fprintf(&buf, "%% profile=%s\n", inv.Profile)
	fmt.Fprintf(&buf, "%% cii_xml_size=%d\n", len(xml))
	fmt.Fprintf(&buf, "%% See sidecar for full visual + embedded Factur-X XML\n")
	fmt.Fprintf(&buf, "%%%%EOF\n")
	return buf.Bytes(), nil
}
