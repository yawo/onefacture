//go:build live_pa

package live

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/yawo/onefacture/internal/adapters"
	"github.com/yawo/onefacture/internal/adapters/chorus"
	"github.com/yawo/onefacture/internal/adapters/docaposte"
	"github.com/yawo/onefacture/internal/adapters/pennylane"
	"github.com/yawo/onefacture/internal/core/invoice"
)

func TestLivePASandboxes(t *testing.T) {
	tests := []struct {
		name     string
		baseEnv  string
		tokenEnv string
		adapter  adapters.PAAdapter
	}{
		{"chorus", "ONEFACTURE_CHORUS_BASE_URL", "ONEFACTURE_CHORUS_ACCESS_TOKEN", chorus.New()},
		{"docaposte", "ONEFACTURE_DOCAPOSTE_BASE_URL", "ONEFACTURE_DOCAPOSTE_API_TOKEN", docaposte.New()},
		{"pennylane", "ONEFACTURE_PENNYLANE_BASE_URL", "ONEFACTURE_PENNYLANE_API_TOKEN", pennylane.New()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if os.Getenv(tt.baseEnv) == "" || os.Getenv(tt.tokenEnv) == "" {
				if os.Getenv("ONEFACTURE_REQUIRE_LIVE_PA") == "true" {
					t.Fatalf("%s or %s not configured", tt.baseEnv, tt.tokenEnv)
				}
				t.Skipf("%s or %s not configured", tt.baseEnv, tt.tokenEnv)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			submitted, err := tt.adapter.Submit(ctx, liveInvoice(tt.name))
			require.NoError(t, err)
			require.NotEmpty(t, submitted.PARef)

			status, err := tt.adapter.GetStatus(ctx, submitted.PARef)
			require.NoError(t, err)
			require.NotEmpty(t, status.Status)
		})
	}
}

func liveInvoice(suffix string) *invoice.Invoice {
	now := time.Now().UTC()
	return &invoice.Invoice{
		Profile:   invoice.ProfileEN16931,
		TypeCode:  invoice.TypeCommercialInvoice,
		Number:    "LIVE-" + suffix + "-" + now.Format("20060102150405"),
		Currency:  "EUR",
		IssueDate: now,
		Seller: invoice.Party{
			Name:  "Acme SAS",
			SIREN: "732829320",
			Address: invoice.Address{
				Line1:       "1 rue Cler",
				PostalCode:  "75007",
				City:        "Paris",
				CountryCode: "FR",
			},
		},
		Buyer: invoice.Party{
			Name:  "Globex SAS",
			SIREN: "552120222",
			Address: invoice.Address{
				Line1:       "2 avenue Foch",
				PostalCode:  "75116",
				City:        "Paris",
				CountryCode: "FR",
			},
		},
		Lines: []invoice.Line{{
			Description: "Live sandbox smoke test",
			Quantity:    1,
			UnitCode:    "C62",
			UnitPrice:   1,
			TaxRate:     20,
			TaxCategory: "S",
		}},
	}
}
