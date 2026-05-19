package invoice

import "math"

// ComputeTotals re-derives line nets and the Totals block from raw line inputs.
// Rounding follows EN 16931 BR-CO: half-up to 2 decimals on monetary fields.
func (inv *Invoice) ComputeTotals() {
	taxByKey := map[string]*TaxSubtotal{}
	var lineNet, taxAmount float64

	for i := range inv.Lines {
		l := &inv.Lines[i]
		l.NetAmount = round2(l.Quantity * l.UnitPrice)
		l.TaxAmount = round2(l.NetAmount * l.TaxRate / 100.0)

		lineNet += l.NetAmount
		taxAmount += l.TaxAmount

		key := l.TaxCategory + "|" + ftoa(l.TaxRate)
		st, ok := taxByKey[key]
		if !ok {
			st = &TaxSubtotal{Category: l.TaxCategory, Rate: l.TaxRate}
			taxByKey[key] = st
		}
		st.TaxableBase = round2(st.TaxableBase + l.NetAmount)
		st.TaxAmount = round2(st.TaxAmount + l.TaxAmount)
	}

	inv.Totals.LineNetAmount = round2(lineNet)
	inv.Totals.TaxExclusiveAmount = round2(lineNet)
	inv.Totals.TaxAmount = round2(taxAmount)
	inv.Totals.TaxInclusiveAmount = round2(lineNet + taxAmount)
	inv.Totals.PayableAmount = round2(inv.Totals.TaxInclusiveAmount - inv.Totals.PaidAmount)

	inv.Totals.TaxBreakdown = inv.Totals.TaxBreakdown[:0]
	for _, st := range taxByKey {
		inv.Totals.TaxBreakdown = append(inv.Totals.TaxBreakdown, *st)
	}
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func ftoa(v float64) string {
	// Stable key for tax bucketing.
	return formatFloat(v, 2)
}
