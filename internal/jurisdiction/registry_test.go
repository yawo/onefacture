package jurisdiction

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegistryIncludesFranceAndPEPPOLReadyProfile(t *testing.T) {
	reg := NewRegistry()

	fr, err := reg.Get("FR")
	require.NoError(t, err)
	require.Equal(t, "Factur-X EN16931", fr.Name)
	require.Contains(t, fr.Formats, "FACTUR-X")

	eu, err := reg.Get("EU")
	require.NoError(t, err)
	require.Contains(t, eu.Formats, "UBL")

	vida, err := reg.Get("EU-ViDA")
	require.NoError(t, err)
	require.Equal(t, "ViDA / EN16931 (2028+)", vida.Name)
}

func TestRegistryCanAddJurisdictionWithoutCoreAPIChange(t *testing.T) {
	reg := NewRegistry()
	reg.Register(Profile{CountryCode: "DE", Name: "XRechnung", Formats: []string{"CII", "UBL"}})

	got, err := reg.Get("DE")

	require.NoError(t, err)
	require.Equal(t, "XRechnung", got.Name)
}
