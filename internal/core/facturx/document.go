package facturx

import (
	"encoding/xml"

	"github.com/yawo/onefacture/internal/core/invoice"
)

// CrossIndustryInvoice is the root element of CII D22B. We define a
// pragmatic subset sufficient for EN16931 conformance; the sidecar performs
// full XSD validation.
type CrossIndustryInvoice struct {
	XMLName xml.Name `xml:"rsm:CrossIndustryInvoice"`
	RSM     string   `xml:"xmlns:rsm,attr"`
	RAM     string   `xml:"xmlns:ram,attr"`
	QDT     string   `xml:"xmlns:qdt,attr"`
	UDT     string   `xml:"xmlns:udt,attr"`

	ExchangedDocumentContext ExchangedDocumentContext `xml:"rsm:ExchangedDocumentContext"`
	ExchangedDocument        ExchangedDocument        `xml:"rsm:ExchangedDocument"`
	SupplyChainTradeTx       SupplyChainTradeTx       `xml:"rsm:SupplyChainTradeTransaction"`
}

type ExchangedDocumentContext struct {
	GuidelineSpecifiedDocumentContextParameter struct {
		ID string `xml:"ram:ID"`
	} `xml:"ram:GuidelineSpecifiedDocumentContextParameter"`
}

type ExchangedDocument struct {
	ID            string         `xml:"ram:ID"`
	TypeCode      string         `xml:"ram:TypeCode"`
	IssueDateTime DateTimeString `xml:"ram:IssueDateTime"`
}

type DateTimeString struct {
	Date struct {
		Format string `xml:"format,attr"`
		Value  string `xml:",chardata"`
	} `xml:"udt:DateTimeString"`
}

type SupplyChainTradeTx struct {
	Lines         []LineItem    `xml:"ram:IncludedSupplyChainTradeLineItem"`
	ApplicableHeaderTradeAgreement   HeaderTradeAgreement   `xml:"ram:ApplicableHeaderTradeAgreement"`
	ApplicableHeaderTradeDelivery    HeaderTradeDelivery    `xml:"ram:ApplicableHeaderTradeDelivery"`
	ApplicableHeaderTradeSettlement  HeaderTradeSettlement  `xml:"ram:ApplicableHeaderTradeSettlement"`
}

type LineItem struct {
	AssociatedDocumentLineDocument struct {
		LineID string `xml:"ram:LineID"`
	} `xml:"ram:AssociatedDocumentLineDocument"`
	SpecifiedTradeProduct struct {
		Name string `xml:"ram:Name"`
	} `xml:"ram:SpecifiedTradeProduct"`
	SpecifiedLineTradeAgreement struct {
		NetPriceProductTradePrice struct {
			ChargeAmount string `xml:"ram:ChargeAmount"`
		} `xml:"ram:NetPriceProductTradePrice"`
	} `xml:"ram:SpecifiedLineTradeAgreement"`
	SpecifiedLineTradeDelivery struct {
		BilledQuantity QuantityValue `xml:"ram:BilledQuantity"`
	} `xml:"ram:SpecifiedLineTradeDelivery"`
	SpecifiedLineTradeSettlement struct {
		ApplicableTradeTax struct {
			TypeCode             string `xml:"ram:TypeCode"`
			CategoryCode         string `xml:"ram:CategoryCode"`
			RateApplicablePercent string `xml:"ram:RateApplicablePercent"`
		} `xml:"ram:ApplicableTradeTax"`
		SpecifiedTradeSettlementLineMonetarySummation struct {
			LineTotalAmount string `xml:"ram:LineTotalAmount"`
		} `xml:"ram:SpecifiedTradeSettlementLineMonetarySummation"`
	} `xml:"ram:SpecifiedLineTradeSettlement"`
}

type QuantityValue struct {
	UnitCode string `xml:"unitCode,attr"`
	Value    string `xml:",chardata"`
}

type HeaderTradeAgreement struct {
	BuyerReference string      `xml:"ram:BuyerReference,omitempty"`
	SellerTradeParty TradeParty `xml:"ram:SellerTradeParty"`
	BuyerTradeParty  TradeParty `xml:"ram:BuyerTradeParty"`
}

type TradeParty struct {
	Name                   string                 `xml:"ram:Name"`
	SpecifiedLegalOrganization *LegalOrganization `xml:"ram:SpecifiedLegalOrganization,omitempty"`
	PostalTradeAddress     PostalTradeAddress     `xml:"ram:PostalTradeAddress"`
	SpecifiedTaxRegistration *TaxRegistration     `xml:"ram:SpecifiedTaxRegistration,omitempty"`
}

type LegalOrganization struct {
	ID struct {
		SchemeID string `xml:"schemeID,attr"`
		Value    string `xml:",chardata"`
	} `xml:"ram:ID"`
}

type PostalTradeAddress struct {
	PostcodeCode string `xml:"ram:PostcodeCode"`
	LineOne      string `xml:"ram:LineOne"`
	LineTwo      string `xml:"ram:LineTwo,omitempty"`
	CityName     string `xml:"ram:CityName"`
	CountryID    string `xml:"ram:CountryID"`
}

type TaxRegistration struct {
	ID struct {
		SchemeID string `xml:"schemeID,attr"`
		Value    string `xml:",chardata"`
	} `xml:"ram:ID"`
}

type HeaderTradeDelivery struct{}

type HeaderTradeSettlement struct {
	InvoiceCurrencyCode string `xml:"ram:InvoiceCurrencyCode"`
	ApplicableTradeTax  []ApplicableTradeTax `xml:"ram:ApplicableTradeTax"`
	SpecifiedTradeSettlementHeaderMonetarySummation TradeSummation `xml:"ram:SpecifiedTradeSettlementHeaderMonetarySummation"`
}

type ApplicableTradeTax struct {
	CalculatedAmount     string `xml:"ram:CalculatedAmount"`
	TypeCode             string `xml:"ram:TypeCode"`
	BasisAmount          string `xml:"ram:BasisAmount"`
	CategoryCode         string `xml:"ram:CategoryCode"`
	RateApplicablePercent string `xml:"ram:RateApplicablePercent"`
}

type TradeSummation struct {
	LineTotalAmount      string `xml:"ram:LineTotalAmount"`
	TaxBasisTotalAmount  string `xml:"ram:TaxBasisTotalAmount"`
	TaxTotalAmount       Amount `xml:"ram:TaxTotalAmount"`
	GrandTotalAmount     string `xml:"ram:GrandTotalAmount"`
	DuePayableAmount     string `xml:"ram:DuePayableAmount"`
}

type Amount struct {
	Currency string `xml:"currencyID,attr"`
	Value    string `xml:",chardata"`
}

func buildDocument(inv *invoice.Invoice) *CrossIndustryInvoice {
	doc := &CrossIndustryInvoice{
		RSM: "urn:un:unece:uncefact:data:standard:CrossIndustryInvoice:100",
		RAM: "urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100",
		QDT: "urn:un:unece:uncefact:data:standard:QualifiedDataType:100",
		UDT: "urn:un:unece:uncefact:data:standard:UnqualifiedDataType:100",
	}
	doc.ExchangedDocumentContext.GuidelineSpecifiedDocumentContextParameter.ID = GuidelineSpecifiedDocumentContextParameterID(inv.Profile)

	doc.ExchangedDocument.ID = inv.Number
	doc.ExchangedDocument.TypeCode = string(inv.TypeCode)
	doc.ExchangedDocument.IssueDateTime.Date.Format = "102"
	doc.ExchangedDocument.IssueDateTime.Date.Value = formatDate(inv.IssueDate)

	for i, l := range inv.Lines {
		li := LineItem{}
		li.AssociatedDocumentLineDocument.LineID = itoa(i + 1)
		li.SpecifiedTradeProduct.Name = l.Description
		li.SpecifiedLineTradeAgreement.NetPriceProductTradePrice.ChargeAmount = ftoa(l.UnitPrice)
		li.SpecifiedLineTradeDelivery.BilledQuantity = QuantityValue{UnitCode: l.UnitCode, Value: ftoa(l.Quantity)}
		li.SpecifiedLineTradeSettlement.ApplicableTradeTax.TypeCode = "VAT"
		li.SpecifiedLineTradeSettlement.ApplicableTradeTax.CategoryCode = l.TaxCategory
		li.SpecifiedLineTradeSettlement.ApplicableTradeTax.RateApplicablePercent = ftoa(l.TaxRate)
		li.SpecifiedLineTradeSettlement.SpecifiedTradeSettlementLineMonetarySummation.LineTotalAmount = ftoa(l.NetAmount)
		doc.SupplyChainTradeTx.Lines = append(doc.SupplyChainTradeTx.Lines, li)
	}

	doc.SupplyChainTradeTx.ApplicableHeaderTradeAgreement.SellerTradeParty = toTradeParty(inv.Seller)
	doc.SupplyChainTradeTx.ApplicableHeaderTradeAgreement.BuyerTradeParty = toTradeParty(inv.Buyer)
	doc.SupplyChainTradeTx.ApplicableHeaderTradeAgreement.BuyerReference = inv.BuyerReference

	st := &doc.SupplyChainTradeTx.ApplicableHeaderTradeSettlement
	st.InvoiceCurrencyCode = inv.Currency
	for _, tb := range inv.Totals.TaxBreakdown {
		st.ApplicableTradeTax = append(st.ApplicableTradeTax, ApplicableTradeTax{
			CalculatedAmount:      ftoa(tb.TaxAmount),
			TypeCode:              "VAT",
			BasisAmount:           ftoa(tb.TaxableBase),
			CategoryCode:          tb.Category,
			RateApplicablePercent: ftoa(tb.Rate),
		})
	}
	st.SpecifiedTradeSettlementHeaderMonetarySummation = TradeSummation{
		LineTotalAmount:     ftoa(inv.Totals.LineNetAmount),
		TaxBasisTotalAmount: ftoa(inv.Totals.TaxExclusiveAmount),
		TaxTotalAmount:      Amount{Currency: inv.Currency, Value: ftoa(inv.Totals.TaxAmount)},
		GrandTotalAmount:    ftoa(inv.Totals.TaxInclusiveAmount),
		DuePayableAmount:    ftoa(inv.Totals.PayableAmount),
	}
	return doc
}

func toTradeParty(p invoice.Party) TradeParty {
	out := TradeParty{
		Name: p.Name,
		PostalTradeAddress: PostalTradeAddress{
			PostcodeCode: p.Address.PostalCode,
			LineOne:      p.Address.Line1,
			LineTwo:      p.Address.Line2,
			CityName:     p.Address.City,
			CountryID:    p.Address.CountryCode,
		},
	}
	if p.SIREN != "" || p.SIRET != "" {
		lo := &LegalOrganization{}
		if p.SIRET != "" {
			lo.ID.SchemeID = "0009"
			lo.ID.Value = p.SIRET
		} else {
			lo.ID.SchemeID = "0002"
			lo.ID.Value = p.SIREN
		}
		out.SpecifiedLegalOrganization = lo
	}
	if p.VATNumber != "" {
		tr := &TaxRegistration{}
		tr.ID.SchemeID = "VA"
		tr.ID.Value = p.VATNumber
		out.SpecifiedTaxRegistration = tr
	}
	return out
}
