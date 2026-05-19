package facturx

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/yawo/onefacture/internal/core/invoice"
)

func TestPackagePDFA3Success(t *testing.T) {
	inv := &invoice.Invoice{
		Number:  "INV-001",
		Profile: invoice.ProfileEN16931,
	}
	xml := []byte("<?xml version=\"1.0\"?><CrossIndustryInvoice></CrossIndustryInvoice>")

	pdf, err := PackagePDFA3(inv, xml)
	require.NoError(t, err)
	require.NotNil(t, pdf)
	require.NotEmpty(t, pdf)

	// Verify it contains the placeholder marker
	str := string(pdf)
	require.Contains(t, str, "%PDF-1.7")
	require.Contains(t, str, "onefacture placeholder PDF/A-3")
	require.Contains(t, str, "INV-001")
	require.Contains(t, str, "%%EOF")
}

func TestPackagePDFA3WithNilInvoice(t *testing.T) {
	xml := []byte("<?xml version=\"1.0\"?><CrossIndustryInvoice></CrossIndustryInvoice>")
	_, err := PackagePDFA3(nil, xml)
	require.Error(t, err)
	require.Contains(t, err.Error(), "nil invoice")
}

func TestPackagePDFA3IncludesXMLSize(t *testing.T) {
	inv := &invoice.Invoice{
		Number:  "INV-002",
		Profile: invoice.ProfileBasic,
	}
	xml := []byte("test xml content here")

	pdf, err := PackagePDFA3(inv, xml)
	require.NoError(t, err)

	str := string(pdf)
	require.Contains(t, str, "cii_xml_size=21") // "test xml content here" is 21 bytes
}

func TestPackagePDFA3IncludesProfile(t *testing.T) {
	profiles := []invoice.Profile{
		invoice.ProfileMinimum,
		invoice.ProfileBasic,
		invoice.ProfileEN16931,
		invoice.ProfileExtended,
	}

	for _, profile := range profiles {
		inv := &invoice.Invoice{
			Number:  "INV-TEST",
			Profile: profile,
		}
		xml := []byte("<test/>")

		pdf, err := PackagePDFA3(inv, xml)
		require.NoError(t, err)

		str := string(pdf)
		require.Contains(t, str, "profile="+string(profile))
	}
}

func TestPackagePDFA3IncludesTimestamp(t *testing.T) {
	inv := &invoice.Invoice{
		Number:  "INV-003",
		Profile: invoice.ProfileEN16931,
	}
	xml := []byte("<test/>")

	before := time.Now().UTC()
	pdf, err := PackagePDFA3(inv, xml)
	require.NoError(t, err)

	str := string(pdf)
	require.Contains(t, str, "generated_at=")

	// Verify the timestamp is approximately current
	require.Contains(t, str, before.Format("2006-01-02T"))
}

func TestPackagePDFA3EmptyXML(t *testing.T) {
	inv := &invoice.Invoice{
		Number:  "INV-004",
		Profile: invoice.ProfileEN16931,
	}
	xml := []byte("")

	pdf, err := PackagePDFA3(inv, xml)
	require.NoError(t, err)
	require.NotEmpty(t, pdf)

	str := string(pdf)
	require.Contains(t, str, "cii_xml_size=0")
}

func TestPackagePDFA3LargeXML(t *testing.T) {
	inv := &invoice.Invoice{
		Number:  "INV-005",
		Profile: invoice.ProfileEN16931,
	}

	// Create a large XML content
	largeXML := bytes.Repeat([]byte("x"), 10000)

	pdf, err := PackagePDFA3(inv, largeXML)
	require.NoError(t, err)
	require.NotEmpty(t, pdf)

	str := string(pdf)
	require.Contains(t, str, "cii_xml_size=10000")
}

func TestPackagePDFA3MultipleInvocations(t *testing.T) {
	// Test that multiple calls produce similar structure
	inv1 := &invoice.Invoice{Number: "INV-A", Profile: invoice.ProfileEN16931}
	inv2 := &invoice.Invoice{Number: "INV-B", Profile: invoice.ProfileEN16931}
	xml := []byte("<test/>")

	pdf1, err1 := PackagePDFA3(inv1, xml)
	pdf2, err2 := PackagePDFA3(inv2, xml)

	require.NoError(t, err1)
	require.NoError(t, err2)

	// Both should start with %PDF-1.7
	require.True(t, bytes.HasPrefix(pdf1, []byte("%PDF")))
	require.True(t, bytes.HasPrefix(pdf2, []byte("%PDF")))

	// Both should end with %%EOF
	require.True(t, bytes.HasSuffix(pdf1, []byte("%%EOF\n")))
	require.True(t, bytes.HasSuffix(pdf2, []byte("%%EOF\n")))

	// But they should differ (different invoice numbers)
	require.NotEqual(t, pdf1, pdf2)
}
