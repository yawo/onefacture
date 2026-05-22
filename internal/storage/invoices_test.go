package storage

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/yawo/onefacture/internal/core/invoice"
)

func createTestInvoice() *invoice.Invoice {
	now := time.Now()
	dueDate := now.Add(30 * 24 * time.Hour)
	return &invoice.Invoice{
		Status:    invoice.StatusDraft,
		Profile:   invoice.ProfileEN16931,
		TypeCode:  invoice.TypeCommercialInvoice,
		Number:    "INV-2024-001",
		IssueDate: now,
		DueDate:   &dueDate,
		Currency:  "EUR",
		Seller: invoice.Party{
			Name:  "Seller Corp",
			SIREN: "123456782",
			Address: invoice.Address{
				Line1:       "123 Seller St",
				City:        "Paris",
				PostalCode:  "75001",
				CountryCode: "FR",
			},
		},
		Buyer: invoice.Party{
			Name:  "Buyer Inc",
			SIREN: "987654321",
			Address: invoice.Address{
				Line1:       "456 Buyer Ave",
				City:        "Lyon",
				PostalCode:  "69000",
				CountryCode: "FR",
			},
		},
		Lines: []invoice.Line{
			{
				Description: "Service A",
				Quantity:    1,
				UnitCode:    "C62",
				UnitPrice:   1000.00,
				NetAmount:   1000.00,
				TaxRate:     20,
				TaxCategory: "S",
				TaxAmount:   200.00,
			},
		},
		Totals: invoice.Totals{
			LineNetAmount:      1000.00,
			TaxExclusiveAmount: 1000.00,
			TaxAmount:          200.00,
			TaxInclusiveAmount: 1200.00,
			PayableAmount:      1200.00,
		},
	}
}

func TestInvoiceCreateSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgID := uuid.New()
	inv := createTestInvoice()

	id, err := store.Invoices.Create(ctx, orgID, DirectionOutbound, inv)

	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, id)
	require.NotEmpty(t, inv.ID)
	require.Equal(t, orgID.String(), inv.OrganizationID)
	require.False(t, inv.CreatedAt.IsZero())
	require.False(t, inv.UpdatedAt.IsZero())
}

func TestInvoiceCreateInbound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgID := uuid.New()
	inv := createTestInvoice()
	inv.Status = invoice.StatusReceived

	id, err := store.Invoices.Create(ctx, orgID, DirectionInbound, inv)

	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, id)
}

func TestInvoiceGetSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgID := uuid.New()
	inv := createTestInvoice()

	_, err := store.Invoices.Create(ctx, orgID, DirectionOutbound, inv)
	require.NoError(t, err)

	retrieved, err := store.Invoices.Get(ctx, orgID, uuid.MustParse(inv.ID))

	require.NoError(t, err)
	require.NotNil(t, retrieved)
	require.Equal(t, inv.ID, retrieved.ID)
	require.Equal(t, orgID.String(), retrieved.OrganizationID)
	require.Equal(t, inv.Number, retrieved.Number)
	require.Equal(t, inv.Status, retrieved.Status)
	require.Equal(t, inv.Profile, retrieved.Profile)
	require.Equal(t, inv.Seller.SIREN, retrieved.Seller.SIREN)
	require.Equal(t, inv.Buyer.SIREN, retrieved.Buyer.SIREN)
}

func TestInvoiceGetNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	_, err := store.Invoices.Get(ctx, uuid.New(), uuid.New())

	require.Equal(t, ErrNotFound, err)
}

func TestInvoiceGetWrongOrganization(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgID1 := uuid.New()
	orgID2 := uuid.New()
	inv := createTestInvoice()

	_, err := store.Invoices.Create(ctx, orgID1, DirectionOutbound, inv)
	require.NoError(t, err)

	_, err = store.Invoices.Get(ctx, orgID2, uuid.MustParse(inv.ID))

	require.Equal(t, ErrNotFound, err)
}

func TestInvoiceUpdateStatusSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgID := uuid.New()
	inv := createTestInvoice()

	_, err := store.Invoices.Create(ctx, orgID, DirectionOutbound, inv)
	require.NoError(t, err)

	err = store.Invoices.UpdateStatus(ctx, orgID, uuid.MustParse(inv.ID), invoice.StatusSubmitted)

	require.NoError(t, err)

	retrieved, err := store.Invoices.Get(ctx, orgID, uuid.MustParse(inv.ID))
	require.NoError(t, err)
	require.Equal(t, invoice.StatusSubmitted, retrieved.Status)
}

func TestInvoiceUpdateStatusNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.Invoices.UpdateStatus(ctx, uuid.New(), uuid.New(), invoice.StatusSubmitted)

	require.Equal(t, ErrNotFound, err)
}

func TestInvoiceUpdateStatusWrongOrganization(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgID1 := uuid.New()
	orgID2 := uuid.New()
	inv := createTestInvoice()

	_, err := store.Invoices.Create(ctx, orgID1, DirectionOutbound, inv)
	require.NoError(t, err)

	err = store.Invoices.UpdateStatus(ctx, orgID2, uuid.MustParse(inv.ID), invoice.StatusSubmitted)

	require.Equal(t, ErrNotFound, err)
}

func TestInvoiceListEmpty(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	results, err := store.Invoices.List(ctx, uuid.New(), ListFilter{})

	require.NoError(t, err)
	require.Empty(t, results)
}

func TestInvoiceListSingle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgID := uuid.New()
	inv := createTestInvoice()

	_, err := store.Invoices.Create(ctx, orgID, DirectionOutbound, inv)
	require.NoError(t, err)

	results, err := store.Invoices.List(ctx, orgID, ListFilter{})

	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, inv.Number, results[0].Number)
}

func TestInvoiceListMultiple(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgID := uuid.New()

	for i := 0; i < 5; i++ {
		inv := createTestInvoice()
		inv.Number = "INV-2024-00" + string(rune('0'+i))
		_, err := store.Invoices.Create(ctx, orgID, DirectionOutbound, inv)
		require.NoError(t, err)
	}

	results, err := store.Invoices.List(ctx, orgID, ListFilter{})

	require.NoError(t, err)
	require.Len(t, results, 5)
}

func TestInvoiceListFilterByDirection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgID := uuid.New()
	inv1 := createTestInvoice()
	inv2 := createTestInvoice()
	inv2.Number = "INV-2024-002"

	_, err := store.Invoices.Create(ctx, orgID, DirectionOutbound, inv1)
	require.NoError(t, err)
	_, err = store.Invoices.Create(ctx, orgID, DirectionInbound, inv2)
	require.NoError(t, err)

	outbound, err := store.Invoices.List(ctx, orgID, ListFilter{Direction: DirectionOutbound})
	require.NoError(t, err)
	require.Len(t, outbound, 1)

	inbound, err := store.Invoices.List(ctx, orgID, ListFilter{Direction: DirectionInbound})
	require.NoError(t, err)
	require.Len(t, inbound, 1)
}

func TestInvoiceListFilterByStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgID := uuid.New()
	inv1 := createTestInvoice()
	inv1.Status = invoice.StatusDraft
	inv2 := createTestInvoice()
	inv2.Number = "INV-2024-002"
	inv2.Status = invoice.StatusSubmitted

	_, err := store.Invoices.Create(ctx, orgID, DirectionOutbound, inv1)
	require.NoError(t, err)
	_, err = store.Invoices.Create(ctx, orgID, DirectionOutbound, inv2)
	require.NoError(t, err)

	drafts, err := store.Invoices.List(ctx, orgID, ListFilter{Status: invoice.StatusDraft})
	require.NoError(t, err)
	require.Len(t, drafts, 1)

	submitted, err := store.Invoices.List(ctx, orgID, ListFilter{Status: invoice.StatusSubmitted})
	require.NoError(t, err)
	require.Len(t, submitted, 1)
}

func TestInvoiceListPagination(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgID := uuid.New()

	for i := 0; i < 15; i++ {
		inv := createTestInvoice()
		inv.Number = "INV-2024-" + string(rune('0'+(i/10))) + string(rune('0'+(i%10)))
		_, err := store.Invoices.Create(ctx, orgID, DirectionOutbound, inv)
		require.NoError(t, err)
	}

	page1, err := store.Invoices.List(ctx, orgID, ListFilter{Limit: 10, Offset: 0})
	require.NoError(t, err)
	require.Len(t, page1, 10)

	page2, err := store.Invoices.List(ctx, orgID, ListFilter{Limit: 10, Offset: 10})
	require.NoError(t, err)
	require.Len(t, page2, 5)
}

func TestInvoiceListLimitDefaults(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgID := uuid.New()

	for i := 0; i < 5; i++ {
		inv := createTestInvoice()
		inv.Number = "INV-2024-00" + string(rune('0'+i))
		_, err := store.Invoices.Create(ctx, orgID, DirectionOutbound, inv)
		require.NoError(t, err)
	}

	results, err := store.Invoices.List(ctx, orgID, ListFilter{Limit: 0})
	require.NoError(t, err)
	require.Len(t, results, 5)
}

func TestInvoiceListLimitCapped(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgID := uuid.New()

	for i := 0; i < 5; i++ {
		inv := createTestInvoice()
		inv.Number = "INV-2024-00" + string(rune('0'+i))
		_, err := store.Invoices.Create(ctx, orgID, DirectionOutbound, inv)
		require.NoError(t, err)
	}

	results, err := store.Invoices.List(ctx, orgID, ListFilter{Limit: 300})
	require.NoError(t, err)
	require.Len(t, results, 5)
}

func TestInvoiceCreateWithRawArtifacts(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgID := uuid.New()
	inv := createTestInvoice()
	inv.RawXML = []byte("<cii>test xml</cii>")
	inv.RawPDF = []byte("%PDF-1.4 test pdf")

	_, err := store.Invoices.Create(ctx, orgID, DirectionOutbound, inv)

	require.NoError(t, err)

	retrieved, err := store.Invoices.Get(ctx, orgID, uuid.MustParse(inv.ID))
	require.NoError(t, err)
	require.NotNil(t, retrieved)
}

func TestInvoiceCreateMultipleOrganizations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	org1 := uuid.New()
	org2 := uuid.New()

	inv1 := createTestInvoice()
	inv2 := createTestInvoice()
	inv2.Number = "INV-2024-002"

	_, err := store.Invoices.Create(ctx, org1, DirectionOutbound, inv1)
	require.NoError(t, err)
	_, err = store.Invoices.Create(ctx, org2, DirectionOutbound, inv2)
	require.NoError(t, err)

	list1, err := store.Invoices.List(ctx, org1, ListFilter{})
	require.NoError(t, err)
	require.Len(t, list1, 1)

	list2, err := store.Invoices.List(ctx, org2, ListFilter{})
	require.NoError(t, err)
	require.Len(t, list2, 1)
}

func TestInvoiceListOrdered(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgID := uuid.New()

	for i := 0; i < 5; i++ {
		inv := createTestInvoice()
		inv.Number = "INV-2024-00" + string(rune('0'+i))
		_, err := store.Invoices.Create(ctx, orgID, DirectionOutbound, inv)
		require.NoError(t, err)
		time.Sleep(50 * time.Millisecond)
	}

	results, err := store.Invoices.List(ctx, orgID, ListFilter{})
	require.NoError(t, err)
	require.Len(t, results, 5)

	for i := 0; i < len(results)-1; i++ {
		require.True(t, results[i].CreatedAt.After(results[i+1].CreatedAt) || results[i].CreatedAt.Equal(results[i+1].CreatedAt),
			"results should be ordered by created_at DESC")
	}
}
