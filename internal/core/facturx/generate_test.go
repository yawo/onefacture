package facturx

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/yawo/onefacture/internal/core/invoice"
)

func TestGenerateCII(t *testing.T) {
	inv := &invoice.Invoice{
		Profile:   invoice.ProfileEN16931,
		TypeCode:  invoice.TypeCommercialInvoice,
		Number:    "INV-001",
		IssueDate: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		Currency:  "EUR",
		Seller:    invoice.Party{Name: "Acme", SIREN: "732829320", Address: invoice.Address{Line1: "1 rue", PostalCode: "75007", City: "Paris", CountryCode: "FR"}},
		Buyer:     invoice.Party{Name: "Globex", SIREN: "552120222", Address: invoice.Address{Line1: "2 av", PostalCode: "75116", City: "Paris", CountryCode: "FR"}},
		Lines: []invoice.Line{
			{Description: "Consulting", Quantity: 10, UnitCode: "HUR", UnitPrice: 100, TaxRate: 20, TaxCategory: "S"},
		},
	}
	inv.ComputeTotals()
	xml, err := GenerateCII(inv)
	require.NoError(t, err)
	s := string(xml)
	require.True(t, strings.Contains(s, "CrossIndustryInvoice"))
	require.True(t, strings.Contains(s, "INV-001"))
	require.True(t, strings.Contains(s, "urn:cen.eu:en16931:2017"))
	require.True(t, strings.Contains(s, "Acme"))
	require.True(t, strings.Contains(s, "EUR"))
}
