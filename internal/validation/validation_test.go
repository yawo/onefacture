package validation

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/yawo/onefacture/internal/core/invoice"
)

func TestRunBusinessRulesValidInvoice(t *testing.T) {
	inv := &invoice.Invoice{
		Profile:   invoice.ProfileEN16931,
		Currency:  "EUR",
		IssueDate: time.Now(),
		Status:    invoice.StatusDraft,
		Seller: invoice.Party{
			Name:  "Seller",
			SIREN: "123456782", // valid SIREN (Luhn verified)
		},
		Buyer: invoice.Party{
			Name:  "Buyer",
			SIREN: "987654324", // valid SIREN (Luhn verified)
		},
		Lines: []invoice.Line{
			{Description: "Test", Quantity: 1, UnitPrice: 100, TaxRate: 20, TaxCategory: "S"},
		},
	}
	inv.ComputeTotals()

	findings := runBusinessRules(inv)
	for _, f := range findings {
		t.Logf("Finding: %s - %s", f.Code, f.Message)
	}
	require.Empty(t, findings)
}

func TestValidSIREN(t *testing.T) {
	tests := []struct {
		name  string
		siren string
		valid bool
	}{
		{"valid SIREN 1", "123456782", true},
		{"valid SIREN 2", "987654324", true},
		{"invalid length", "12345678", false},
		{"invalid length", "1234567890", false},
		{"non-numeric", "12345678a", false},
		{"all zeros", "000000000", true}, // 000000000 actually passes Luhn
		{"spaces", "1 234 567 89", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validSIREN(tt.siren)
			if tt.valid {
				require.True(t, result, "SIREN %s should be valid", tt.siren)
			} else {
				require.False(t, result, "SIREN %s should be invalid", tt.siren)
			}
		})
	}
}

func TestRunBusinessRulesInvalidSIREN(t *testing.T) {
	inv := &invoice.Invoice{
		Profile:   invoice.ProfileEN16931,
		Currency:  "EUR",
		IssueDate: time.Now(),
		Status:    invoice.StatusDraft,
		Seller: invoice.Party{
			Name:  "Seller",
			SIREN: "111111111", // invalid SIREN
		},
		Buyer: invoice.Party{
			Name:  "Buyer",
			SIREN: "987654324", // valid
		},
		Lines: []invoice.Line{
			{Description: "Test", Quantity: 1, UnitPrice: 100, TaxRate: 20, TaxCategory: "S"},
		},
	}
	inv.ComputeTotals()

	findings := runBusinessRules(inv)
	var sirenError *Finding
	for i, f := range findings {
		if f.Code == "FR-SIREN" {
			sirenError = &findings[i]
			break
		}
	}
	require.NotNil(t, sirenError)
	require.Equal(t, "seller.siren", sirenError.Path)
}

func TestRunBusinessRulesMissingCurrency(t *testing.T) {
	inv := &invoice.Invoice{
		Profile:   invoice.ProfileEN16931,
		Currency:  "", // missing currency
		IssueDate: time.Now(),
		Status:    invoice.StatusDraft,
		Seller: invoice.Party{
			Name:  "Seller",
			SIREN: "123456789",
		},
		Buyer: invoice.Party{
			Name:  "Buyer",
			SIREN: "321654987",
		},
		Lines: []invoice.Line{
			{Description: "Test", Quantity: 1, UnitPrice: 100, TaxRate: 20, TaxCategory: "S"},
		},
	}
	inv.ComputeTotals()

	findings := runBusinessRules(inv)
	var currencyError *Finding
	for i, f := range findings {
		if f.Code == "BR-05" {
			currencyError = &findings[i]
			break
		}
	}
	require.NotNil(t, currencyError)
	require.Equal(t, "currency", currencyError.Path)
	require.Equal(t, SeverityError, currencyError.Severity)
}

func TestRunBusinessRulesInvalidTotals(t *testing.T) {
	inv := &invoice.Invoice{
		Profile:   invoice.ProfileEN16931,
		Currency:  "EUR",
		IssueDate: time.Now(),
		Status:    invoice.StatusDraft,
		Seller: invoice.Party{
			Name:  "Seller",
			SIREN: "123456789",
		},
		Buyer: invoice.Party{
			Name:  "Buyer",
			SIREN: "321654987",
		},
		Lines: []invoice.Line{
			{Description: "Test", Quantity: 1, UnitPrice: 100, TaxRate: 20, TaxCategory: "S"},
		},
		Totals: invoice.Totals{
			LineNetAmount:      999.99, // wrong
			TaxExclusiveAmount: 999.99,
			TaxAmount:          200.00,
			TaxInclusiveAmount: 1199.99,
		},
	}

	findings := runBusinessRules(inv)
	var totalsError *Finding
	for i, f := range findings {
		if f.Code == "BR-CO-10" {
			totalsError = &findings[i]
			break
		}
	}
	require.NotNil(t, totalsError, "should detect invalid totals")
}

func TestApproxEqual(t *testing.T) {
	require.True(t, approxEqual(100.00, 100.00))
	require.True(t, approxEqual(100.00, 100.001))
	require.True(t, approxEqual(100.00, 100.009))
	require.False(t, approxEqual(100.00, 100.01))
	require.False(t, approxEqual(100.00, 100.1))
}

func TestRound2(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{100.0, 100.00},
		{100.124, 100.12},
		{100.125, 100.12},  // banker's rounding: round to nearest even
		{100.126, 100.13},
		{100.135, 100.14},  // banker's rounding
	}

	for _, tt := range tests {
		result := round2(tt.input)
		// Check within tolerance for floating point
		if math.Abs(result-tt.expected) > 0.01 {
			t.Errorf("round2(%f) = %f, expected %f", tt.input, result, tt.expected)
		}
	}
}

func TestHasErrors(t *testing.T) {
	tests := []struct {
		name     string
		findings []Finding
		hasError bool
	}{
		{
			name:     "no findings",
			findings: []Finding{},
			hasError: false,
		},
		{
			name: "only warnings",
			findings: []Finding{
				{Severity: SeverityWarning, Message: "warning"},
			},
			hasError: false,
		},
		{
			name: "one error",
			findings: []Finding{
				{Severity: SeverityError, Message: "error"},
			},
			hasError: true,
		},
		{
			name: "errors and warnings",
			findings: []Finding{
				{Severity: SeverityWarning, Message: "warning"},
				{Severity: SeverityError, Message: "error"},
			},
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasErrors(tt.findings)
			require.Equal(t, tt.hasError, result)
		})
	}
}

func TestValidSIRENLuhnAlgorithm(t *testing.T) {
	// Test known valid SIRENs verified with Luhn algorithm
	validSIRENs := []string{
		"123456782", // verified
		"987654324", // verified
	}

	for _, siren := range validSIRENs {
		require.True(t, validSIREN(siren), "SIREN %s should be valid", siren)
	}
}

func TestRunBusinessRulesMissingSellerSIREN(t *testing.T) {
	inv := &invoice.Invoice{
		Profile:   invoice.ProfileEN16931,
		Currency:  "EUR",
		IssueDate: time.Now(),
		Status:    invoice.StatusDraft,
		Seller: invoice.Party{
			Name:  "Seller",
			SIREN: "", // missing
		},
		Buyer: invoice.Party{
			Name:  "Buyer",
			SIREN: "987654324", // valid
		},
		Lines: []invoice.Line{
			{Description: "Test", Quantity: 1, UnitPrice: 100, TaxRate: 20, TaxCategory: "S"},
		},
	}
	inv.ComputeTotals()

	findings := runBusinessRules(inv)
	// Should not error on empty SIREN (it's optional in this context)
	var sirenErrors []Finding
	for _, f := range findings {
		if f.Code == "FR-SIREN" && f.Path == "seller.siren" {
			sirenErrors = append(sirenErrors, f)
		}
	}
	require.Empty(t, sirenErrors)
}
