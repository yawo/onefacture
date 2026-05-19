package validation

import (
	"fmt"
	"math"
	"strconv"

	"github.com/yawo/onefacture/internal/core/invoice"
)

// runBusinessRules implements layer 6: French / EN16931 business rules that
// don't need XML manipulation. Detected issues should mirror Schematron codes
// where possible (e.g. BR-CO-15).
func runBusinessRules(inv *invoice.Invoice) []Finding {
	var out []Finding

	// BR-CO-15: invoice totals must reconcile.
	expectedNet := 0.0
	for _, l := range inv.Lines {
		expectedNet += round2(l.Quantity * l.UnitPrice)
	}
	if !approxEqual(expectedNet, inv.Totals.LineNetAmount) {
		out = append(out, Finding{
			Layer:    "business",
			Code:     "BR-CO-10",
			Severity: SeverityError,
			Path:     "totals.line_net_amount",
			Message:  fmt.Sprintf("sum of line nets %.2f does not match totals.line_net_amount %.2f", expectedNet, inv.Totals.LineNetAmount),
		})
	}
	if !approxEqual(inv.Totals.TaxExclusiveAmount+inv.Totals.TaxAmount, inv.Totals.TaxInclusiveAmount) {
		out = append(out, Finding{
			Layer: "business", Code: "BR-CO-15", Severity: SeverityError,
			Path:    "totals.tax_inclusive_amount",
			Message: "tax_inclusive_amount must equal tax_exclusive_amount + tax_amount",
		})
	}

	// SIREN check digit (Luhn).
	if inv.Seller.SIREN != "" && !validSIREN(inv.Seller.SIREN) {
		out = append(out, Finding{
			Layer: "business", Code: "FR-SIREN", Severity: SeverityError,
			Path: "seller.siren", Message: "invalid SIREN checksum",
		})
	}
	if inv.Buyer.SIREN != "" && !validSIREN(inv.Buyer.SIREN) {
		out = append(out, Finding{
			Layer: "business", Code: "FR-SIREN", Severity: SeverityError,
			Path: "buyer.siren", Message: "invalid SIREN checksum",
		})
	}

	if inv.Currency == "" {
		out = append(out, Finding{
			Layer: "business", Code: "BR-05", Severity: SeverityError,
			Path: "currency", Message: "currency code is required",
		})
	}

	return out
}

// validSIREN checks the 9-digit French company identifier using the Luhn algorithm.
func validSIREN(s string) bool {
	if len(s) != 9 {
		return false
	}
	sum := 0
	for i, r := range s {
		d, err := strconv.Atoi(string(r))
		if err != nil {
			return false
		}
		// SIREN: even-indexed (from right, 0-based) digits doubled.
		// We iterate left-to-right (i=0..8). Position from right = 8-i.
		// Double when (8-i) is odd, i.e. when i is even (0, 2, 4, 6, 8) — invert: SIREN doubles every 2nd from the right starting at the second.
		if (8-i)%2 == 1 {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
	}
	return sum%10 == 0
}

func approxEqual(a, b float64) bool {
	return math.Abs(a-b) < 0.01
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
