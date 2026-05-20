// Package invoice defines the unified Invoice domain model based on
// EN 16931 / Factur-X 1.08. Field selection covers the EN16931 profile;
// MINIMUM/BASIC narrow the set, EXTENDED widens it.
package invoice

import (
	"time"
)

// Profile is the Factur-X conformance profile.
type Profile string

const (
	ProfileMinimum  Profile = "MINIMUM"
	ProfileBasic    Profile = "BASIC"
	ProfileEN16931  Profile = "EN16931"
	ProfileExtended Profile = "EXTENDED"
)

func (p Profile) Valid() bool {
	switch p {
	case ProfileMinimum, ProfileBasic, ProfileEN16931, ProfileExtended:
		return true
	}
	return false
}

// Status represents the lifecycle state of an invoice in onefacture's
// internal state machine. PA-specific codes are mapped here.
type Status string

const (
	StatusDraft     Status = "DRAFT"
	StatusValidated Status = "VALIDATED"
	StatusSubmitted Status = "SUBMITTED"
	StatusReceived  Status = "RECEIVED"
	StatusAccepted  Status = "ACCEPTED"
	StatusRejected  Status = "REJECTED"
	StatusPaid      Status = "PAID"
	StatusCancelled Status = "CANCELLED"
)

// TypeCode is the CII BT-3 invoice type code.
// 380 = Commercial invoice, 381 = Credit note, 384 = Corrective invoice.
type TypeCode string

const (
	TypeCommercialInvoice TypeCode = "380"
	TypeCreditNote        TypeCode = "381"
	TypeCorrective        TypeCode = "384"
)

// Invoice is the unified invoice resource.
type Invoice struct {
	ID             string     `json:"id"`
	OrganizationID string     `json:"organization_id"`
	Status         Status     `json:"status"`
	Profile        Profile    `json:"profile"               validate:"required,oneof=MINIMUM BASIC EN16931 EXTENDED"`
	TypeCode       TypeCode   `json:"type_code"             validate:"required,oneof=380 381 384"`
	Number         string     `json:"number"                validate:"required,max=64"`
	IssueDate      time.Time  `json:"issue_date"            validate:"required"`
	DueDate        *time.Time `json:"due_date,omitempty"`
	Currency       string     `json:"currency"              validate:"required,len=3"`

	Seller Party  `json:"seller"                validate:"required"`
	Buyer  Party  `json:"buyer"                 validate:"required"`
	Lines  []Line `json:"lines"                 validate:"required,min=1,dive"`
	Totals Totals `json:"totals"                validate:"required"`

	Notes          []Note  `json:"notes,omitempty"           validate:"dive"`
	PaymentTerms   string  `json:"payment_terms,omitempty"`
	PaymentMeans   []Means `json:"payment_means,omitempty"   validate:"dive"`
	BuyerReference string  `json:"buyer_reference,omitempty"`
	OrderRef       string  `json:"order_reference,omitempty"`

	// PA routing
	PAID  string `json:"pa_id,omitempty"`
	PARef string `json:"pa_ref,omitempty"`

	// Stored raw artifacts (not serialised in JSON responses, only via dedicated endpoints).
	RawXML []byte `json:"-"`
	RawPDF []byte `json:"-"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	LastRejection *Rejection `json:"last_rejection,omitempty"`
}

// Rejection contains normalized PA rejection details.
type Rejection struct {
	Code           string     `json:"code,omitempty"`
	Message        string     `json:"message,omitempty"`
	OccurredAt     time.Time  `json:"occurred_at"`
	RetryCount     int        `json:"retry_count,omitempty"`
	LastRetryAt    *time.Time `json:"last_retry_at,omitempty"`
	NextRetryAt    *time.Time `json:"next_retry_at,omitempty"`
	ResolutionHint string     `json:"resolution_hint,omitempty"`
}

// Party is a seller or buyer.
type Party struct {
	Name      string   `json:"name"                validate:"required,max=200"`
	LegalName string   `json:"legal_name,omitempty"`
	SIREN     string   `json:"siren,omitempty"     validate:"omitempty,len=9,numeric"`
	SIRET     string   `json:"siret,omitempty"     validate:"omitempty,len=14,numeric"`
	VATNumber string   `json:"vat_number,omitempty"`
	Address   Address  `json:"address"             validate:"required"`
	Contact   *Contact `json:"contact,omitempty"`
}

type Address struct {
	Line1       string `json:"line1"                validate:"required,max=200"`
	Line2       string `json:"line2,omitempty"`
	PostalCode  string `json:"postal_code"          validate:"required"`
	City        string `json:"city"                 validate:"required"`
	CountryCode string `json:"country_code"         validate:"required,len=2"`
}

type Contact struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty" validate:"omitempty,email"`
	Phone string `json:"phone,omitempty"`
}

// Line is an invoice line item.
type Line struct {
	ID          string  `json:"id,omitempty"`
	Description string  `json:"description"        validate:"required,max=512"`
	Quantity    float64 `json:"quantity"           validate:"required,gt=0"`
	UnitCode    string  `json:"unit_code"          validate:"required"` // UN/ECE Rec 20, e.g. "C62" = piece
	UnitPrice   float64 `json:"unit_price"         validate:"gte=0"`
	NetAmount   float64 `json:"net_amount"         validate:"gte=0"`
	TaxRate     float64 `json:"tax_rate"           validate:"gte=0,lte=100"`
	TaxCategory string  `json:"tax_category"       validate:"required,oneof=S Z E AE K G L M O"`
	TaxAmount   float64 `json:"tax_amount"         validate:"gte=0"`
}

// Totals aggregates monetary values per EN 16931.
type Totals struct {
	LineNetAmount      float64       `json:"line_net_amount"          validate:"gte=0"`
	TaxExclusiveAmount float64       `json:"tax_exclusive_amount"     validate:"gte=0"`
	TaxAmount          float64       `json:"tax_amount"               validate:"gte=0"`
	TaxInclusiveAmount float64       `json:"tax_inclusive_amount"     validate:"gte=0"`
	PaidAmount         float64       `json:"paid_amount"              validate:"gte=0"`
	PayableAmount      float64       `json:"payable_amount"           validate:"gte=0"`
	TaxBreakdown       []TaxSubtotal `json:"tax_breakdown,omitempty" validate:"dive"`
}

type TaxSubtotal struct {
	Category    string  `json:"category"      validate:"required"`
	Rate        float64 `json:"rate"          validate:"gte=0,lte=100"`
	TaxableBase float64 `json:"taxable_base"  validate:"gte=0"`
	TaxAmount   float64 `json:"tax_amount"    validate:"gte=0"`
}

type Note struct {
	Subject string `json:"subject,omitempty"`
	Content string `json:"content"           validate:"required"`
}

// Means is a payment means (BT-81 code).
// 30 = SEPA Credit Transfer, 49 = Direct Debit, 48 = Card.
type Means struct {
	Code    string `json:"code"             validate:"required"`
	IBAN    string `json:"iban,omitempty"`
	BIC     string `json:"bic,omitempty"`
	Account string `json:"account,omitempty"`
}
