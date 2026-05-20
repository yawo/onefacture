package invoice

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func sampleInvoice() *Invoice {
	return &Invoice{
		Profile:   ProfileEN16931,
		TypeCode:  TypeCommercialInvoice,
		Number:    "INV-001",
		IssueDate: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		Currency:  "EUR",
		Status:    StatusDraft,
		Seller: Party{
			Name:    "Acme Corp",
			SIREN:   "123456789",
			Address: Address{Line1: "1 rue Cler", PostalCode: "75007", City: "Paris", CountryCode: "FR"},
		},
		Buyer: Party{
			Name:    "Globex Inc",
			SIREN:   "987654321",
			Address: Address{Line1: "2 av Foch", PostalCode: "75116", City: "Paris", CountryCode: "FR"},
		},
		Lines: []Line{
			{Description: "Consulting", Quantity: 10, UnitCode: "HUR", UnitPrice: 150, TaxRate: 20, TaxCategory: "S"},
			{Description: "Travel", Quantity: 1, UnitCode: "C62", UnitPrice: 250, TaxRate: 10, TaxCategory: "S"},
		},
	}
}

func TestComputeTotals(t *testing.T) {
	inv := sampleInvoice()
	inv.ComputeTotals()
	require.Equal(t, 1750.00, inv.Totals.LineNetAmount)
	require.Equal(t, 1750.00, inv.Totals.TaxExclusiveAmount)
	require.Equal(t, 325.00, inv.Totals.TaxAmount)
	require.Equal(t, 2075.00, inv.Totals.TaxInclusiveAmount)
	require.Len(t, inv.Totals.TaxBreakdown, 2)
}

func TestProfileValid(t *testing.T) {
	require.True(t, ProfileEN16931.Valid())
	require.False(t, Profile("FOO").Valid())
}

func TestStateMachine(t *testing.T) {
	inv := sampleInvoice()
	require.True(t, CanTransition(StatusDraft, StatusValidated))
	require.False(t, CanTransition(StatusDraft, StatusPaid))
	require.NoError(t, inv.Transition(StatusValidated))
	require.Equal(t, StatusValidated, inv.Status)
	require.ErrorIs(t, inv.Transition(StatusPaid), ErrInvalidTransition)
}

func TestRejectedCanBeResubmitted(t *testing.T) {
	inv := sampleInvoice()
	inv.Status = StatusRejected
	require.True(t, CanTransition(StatusRejected, StatusSubmitted))
	require.NoError(t, inv.Transition(StatusSubmitted))
	require.Equal(t, StatusSubmitted, inv.Status)
}
