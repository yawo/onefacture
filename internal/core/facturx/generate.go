// Package facturx generates Cross-Industry Invoice (CII D22B) XML according
// to Factur-X 1.08 / EN 16931. PDF/A-3 packaging is handled by a separate
// component and accepts the XML emitted here as the embedded file.
package facturx

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"time"

	"github.com/yawo/onefacture/internal/core/invoice"
)

// GuidelineSpecifiedDocumentContextParameterID returns the Factur-X URN
// for a given profile.
func GuidelineSpecifiedDocumentContextParameterID(p invoice.Profile) string {
	switch p {
	case invoice.ProfileMinimum:
		return "urn:factur-x.eu:1p0:minimum"
	case invoice.ProfileBasic:
		return "urn:cen.eu:en16931:2017#compliant#urn:factur-x.eu:1p0:basic"
	case invoice.ProfileEN16931:
		return "urn:cen.eu:en16931:2017"
	case invoice.ProfileExtended:
		return "urn:cen.eu:en16931:2017#conformant#urn:factur-x.eu:1p0:extended"
	default:
		return "urn:cen.eu:en16931:2017"
	}
}

// GenerateCII returns the CII XML representation of the invoice.
func GenerateCII(inv *invoice.Invoice) ([]byte, error) {
	doc := buildDocument(inv)
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return nil, fmt.Errorf("encode CII: %w", err)
	}
	if err := enc.Flush(); err != nil {
		return nil, fmt.Errorf("flush CII: %w", err)
	}
	return buf.Bytes(), nil
}

func formatDate(t time.Time) string { return t.UTC().Format("20060102") }
