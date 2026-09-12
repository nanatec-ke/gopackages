package definite

import (
	"encoding/json"
	"strings"
)

// Response is the { status, message, data, errorCode } envelope returned by the
// read endpoints and the client, intermediary and payment endpoints.
//
// newProposal and generateCertificate do NOT use it — see NewProposalResponse
// and GenerateCertificateResponse.
//
// Raw holds the complete response body, so fields this SDK does not model can
// be recovered with json.Unmarshal(resp.Raw, &yourStruct).
type Response[T any] struct {
	Status    FlexString      `json:"status,omitempty"`    // "success" or "error"
	Message   FlexString      `json:"message,omitempty"`   // Human-readable outcome
	Data      T               `json:"data,omitempty"`      // Endpoint-specific payload
	ErrorCode FlexString      `json:"errorCode,omitempty"` // Machine-readable code on failure
	Raw       json.RawMessage `json:"-"`                   // Complete response body
}

// Succeeded reports whether the envelope signals success. A response with no
// status at all counts as a success, so a lean payload is never mistaken for a
// failure.
func (r *Response[T]) Succeeded() bool {
	s := strings.ToLower(strings.TrimSpace(string(r.Status)))
	return s == "" || s == ResponseStatusSuccess
}

/* -------------------------------------------------------------------------- */
/*  Clients                                                                   */
/* -------------------------------------------------------------------------- */

// CreateClientRequest is the payload for POST /api/v1/createClient.
type CreateClientRequest struct {
	FirstName  string `json:"firstName"`
	LastName   string `json:"lastName"`
	OtherNames string `json:"otherNames,omitempty"`
	IDNumber   string `json:"idNumber"` // National ID number
	PIN        string `json:"pin"`      // KRA PIN, e.g. A0101401976H
	Email      string `json:"email"`
	Phone      string `json:"phone"`
}

// ClientRecord is a client as returned by checkClient and createClient.
type ClientRecord struct {
	// ID is the internal client record ID. This — not the national ID number —
	// is what NewProposalRequest.Client takes.
	ID         FlexInt    `json:"id,omitempty"`
	FirstName  FlexString `json:"firstName,omitempty"`
	LastName   FlexString `json:"lastName,omitempty"`
	OtherNames FlexString `json:"otherNames,omitempty"`
	IDNumber   FlexString `json:"idNumber,omitempty"`
	PIN        FlexString `json:"pin,omitempty"`
	Email      FlexString `json:"email,omitempty"`
	Phone      FlexString `json:"phone,omitempty"`
}

// CheckClientResponse is returned by Client.CheckClient.
type CheckClientResponse = Response[*ClientRecord]

// CreateClientResponse is returned by Client.CreateClient.
type CreateClientResponse = Response[*ClientRecord]

/* -------------------------------------------------------------------------- */
/*  Reference data                                                            */
/* -------------------------------------------------------------------------- */

// CoverOption belongs to a Product.
type CoverOption struct {
	ID   FlexInt    `json:"id,omitempty"` // Passed as coverOption / CoverOption
	Name FlexString `json:"name,omitempty"`
}

// Product is an underwriting product with its cover options.
type Product struct {
	ID           FlexInt       `json:"id,omitempty"` // Passed as product / Product
	Name         FlexString    `json:"name,omitempty"`
	CoverOptions []CoverOption `json:"coverOptions,omitempty"`
}

// ProductsResponse is returned by Client.GetProducts.
type ProductsResponse = Response[[]Product]

// VehicleModel belongs to a VehicleMake.
type VehicleModel struct {
	ID   FlexInt    `json:"id,omitempty"` // Passed as ModelCode
	Name FlexString `json:"name,omitempty"`
}

// VehicleMake is a manufacturer with its models nested.
type VehicleMake struct {
	ID     FlexInt        `json:"id,omitempty"` // Passed as MakeCode
	Name   FlexString     `json:"name,omitempty"`
	Models []VehicleModel `json:"models,omitempty"`
}

// MakesResponse is returned by Client.GetMakes.
type MakesResponse = Response[[]VehicleMake]

/* -------------------------------------------------------------------------- */
/*  Premium calculation                                                       */
/* -------------------------------------------------------------------------- */

// CalculatePremiumRequest holds the inputs for GET /api/v1/calculatePremium.
//
// The API reads these values from both the query string and the JSON body, so
// the client sends them in both places, matching the documented request.
type CalculatePremiumRequest struct {
	SumInsured  float64 `json:"sumInsured"`  // Declared value; 0 for third-party cover
	Capacity    int     `json:"capacity"`    // Seating capacity
	Tonnage     float64 `json:"tonnage"`     // Carrying capacity in tonnes; 0 for private vehicles
	CoverPeriod string  `json:"coverPeriod"` // One of the PaymentInterval constants
	Product     int64   `json:"product"`     // Product ID from GetProducts
	CoverOption int64   `json:"coverOption"` // Cover option ID from GetProducts

	// TransactionType is sent as a query parameter only; 0 omits it.
	// See TransactionTypeNewBusiness.
	TransactionType int `json:"-"`
}

// PremiumBreakdown is the quote returned by calculatePremium.
type PremiumBreakdown struct {
	Premium      FlexFloat `json:"premium,omitempty"`      // Total payable premium
	BasicPremium FlexFloat `json:"basicPremium,omitempty"` // Before levies and charges
	TrainingLevy FlexFloat `json:"trainingLevy,omitempty"`
	PCF          FlexFloat `json:"pcf,omitempty"` // Policyholders Compensation Fund
	StampDuty    FlexFloat `json:"stampDuty,omitempty"`
	Total        FlexFloat `json:"total,omitempty"` // Total including all levies
}

// CalculatePremiumResponse is returned by Client.CalculatePremium.
type CalculatePremiumResponse = Response[*PremiumBreakdown]

/* -------------------------------------------------------------------------- */
/*  Intermediaries                                                            */
/* -------------------------------------------------------------------------- */

// CheckIntermediaryDetails are optional contact details sent in the body of a
// checkIntermediary lookup; the identification travels in the query string.
type CheckIntermediaryDetails struct {
	FirstName  string `json:"firstName,omitempty"`
	LastName   string `json:"lastName,omitempty"`
	OtherNames string `json:"otherNames,omitempty"`
	IDNumber   string `json:"idNumber,omitempty"`
	PIN        string `json:"pin,omitempty"`
	Email      string `json:"email,omitempty"`
	Phone      string `json:"phone,omitempty"`
}

// RegisterIntermediaryRequest is the payload for POST /api/v1/registerIntermediary.
type RegisterIntermediaryRequest struct {
	Name           string `json:"name"` // Registered name of the agency or brokerage
	Email          string `json:"email"`
	Phone          string `json:"phone"`
	Identification string `json:"identification"` // IRA registration number
	KRAPin         string `json:"krapin"`
}

// IntermediaryRecord is an agent or broker.
type IntermediaryRecord struct {
	// ID is the internal intermediary record ID — what NewProposalRequest.Agent takes.
	ID             FlexInt    `json:"id,omitempty"`
	Name           FlexString `json:"name,omitempty"`
	Email          FlexString `json:"email,omitempty"`
	Phone          FlexString `json:"phone,omitempty"`
	Identification FlexString `json:"identification,omitempty"`
	KRAPin         FlexString `json:"krapin,omitempty"`
}

// CheckIntermediaryResponse is returned by Client.CheckIntermediary.
type CheckIntermediaryResponse = Response[*IntermediaryRecord]

// RegisterIntermediaryResponse is returned by Client.RegisterIntermediary.
type RegisterIntermediaryResponse = Response[*IntermediaryRecord]

/* -------------------------------------------------------------------------- */
/*  Policies                                                                  */
/* -------------------------------------------------------------------------- */

// PolicyRecord is returned by checkPolicy.
type PolicyRecord struct {
	ID               FlexInt    `json:"id,omitempty"`
	PolicyNumber     FlexString `json:"policyNumber,omitempty"`
	Registration     FlexString `json:"registration,omitempty"`
	CommencementDate FlexString `json:"commencementDate,omitempty"`
	ExpiryDate       FlexString `json:"expiryDate,omitempty"`
	Status           FlexString `json:"status,omitempty"`
}

// CheckPolicyResponse is returned by Client.CheckPolicy.
type CheckPolicyResponse = Response[*PolicyRecord]

/* -------------------------------------------------------------------------- */
/*  Proposals                                                                 */
/* -------------------------------------------------------------------------- */

// NewProposalRequest is the payload for POST /api/v1/newProposal. JSON field
// names match the API's own casing, including the lower-case "agent".
//
// Every field is always sent: the API expects SumInsured and Tonnage to be
// present even when they are 0.
type NewProposalRequest struct {
	Client           int64     `json:"Client"` // Client record ID — NOT the national ID number
	Agent            int64     `json:"agent"`  // Intermediary record ID
	CommencementDate Timestamp `json:"CommencementDate"`
	PaymentInterval  string    `json:"PaymentInterval"` // One of the PaymentInterval constants
	Product          int64     `json:"Product"`
	CoverOption      int64     `json:"CoverOption"`
	Registration     string    `json:"Registration"`
	MakeCode         int64     `json:"MakeCode"`
	ModelCode        int64     `json:"ModelCode"`
	Capacity         int       `json:"Capacity"` // Seating capacity
	YOM              int       `json:"YOM"`      // Year of manufacture
	SumInsured       float64   `json:"SumInsured"`
	ChassisNumber    string    `json:"ChassisNumber"`
	EngineNumber     string    `json:"EngineNumber"`
	EngineRating     int       `json:"EngineRating"` // Engine capacity in cc
	ClientID         string    `json:"ClientID"`     // Client's national ID number
	Tonnage          float64   `json:"Tonnage"`
}

// ProposalDetails is the proposal record returned by newProposal. Every later
// call keys off it: OID identifies the proposal, InsuredItem is required to
// post a payment, and Note is the debit note used when generating the certificate.
type ProposalDetails struct {
	OID              FlexInt    `json:"OID"`                 // Proposal ID
	Narration        FlexString `json:"Narration,omitempty"` // In practice the registration
	DateDone         FlexString `json:"DateDone,omitempty"`  // Creation time, DateLayout
	CommencementDate FlexString `json:"CommencementDate,omitempty"`
	ExpiryDate       FlexString `json:"ExpiryDate,omitempty"` // Derived from the payment interval
	AmountToPay      FlexFloat  `json:"AmountToPay,omitempty"`
	InsuredItem      FlexInt    `json:"InsuredItem,omitempty"`    // Risk ID, for InitiatePayment
	Note             FlexInt    `json:"Note,omitempty"`           // Debit note OID
	DocumentNumber   FlexString `json:"DocumentNumber,omitempty"` // e.g. NW2026130551
}

// NewProposalResponse is returned by Client.CreateProposal.
//
// Unlike the read endpoints, newProposal reports its outcome with a boolean
// "success" and nests the record under "proposalDetails" — it does not use the
// Response envelope.
type NewProposalResponse struct {
	Success         *FlexBool        `json:"success,omitempty"`
	Message         FlexString       `json:"message,omitempty"`
	ProposalDetails *ProposalDetails `json:"proposalDetails,omitempty"`
	Raw             json.RawMessage  `json:"-"`
}

// Succeeded reports whether the proposal was created. An absent success field
// counts as a success; only an explicit false is a failure.
func (r *NewProposalResponse) Succeeded() bool { return r.Success == nil || bool(*r.Success) }

/* -------------------------------------------------------------------------- */
/*  Payments                                                                  */
/* -------------------------------------------------------------------------- */

// InitiatePaymentRequest is the payload for POST /api/v1/initiatePayment.
type InitiatePaymentRequest struct {
	InsuredItem      int64     `json:"InsuredItem"` // From ProposalDetails.InsuredItem
	CommencementDate Timestamp `json:"CommencementDate"`
	ExpiryDate       Timestamp `json:"ExpiryDate"`
	Proposal         int64     `json:"Proposal"`    // From ProposalDetails.OID
	PhoneNumber      string    `json:"PhoneNumber"` // 2547XXXXXXXX — see NormalizeKenyanPhone

	// APIUser defaults to Config.APIUser, then Credentials.Username.
	APIUser string `json:"APIUser,omitempty"`

	// PaymentMethod is always sent, because 0 (M-Pesa) is a valid value.
	PaymentMethod    int     `json:"PaymentMethod"`
	MpesaTransaction string  `json:"MpesaTransaction,omitempty"`
	AmountPaid       float64 `json:"AmountPaid"`
	Note             string  `json:"note,omitempty"` // Free text stored against the payment
}

// PaymentRecord is returned by initiatePayment.
//
// This shape is unverified against the live API. newProposal turned out not to
// match its documentation; if this one does not either, recover the fields
// from InitiatePaymentResponse.Raw.
type PaymentRecord struct {
	ID            FlexInt    `json:"id,omitempty"`
	ReceiptNumber FlexString `json:"receiptNumber,omitempty"`
	PolicyNumber  FlexString `json:"policyNumber,omitempty"`
	Status        FlexString `json:"status,omitempty"`
}

// InitiatePaymentResponse is returned by Client.InitiatePayment.
type InitiatePaymentResponse = Response[*PaymentRecord]

/* -------------------------------------------------------------------------- */
/*  Certificates                                                              */
/* -------------------------------------------------------------------------- */

// GenerateCertificateRequest is the payload for POST /api/v1/generateCertificate.
type GenerateCertificateRequest struct {
	Proposal int64 `json:"proposal"` // Proposal ID — ProposalDetails.OID
	NoteOID  int64 `json:"NoteOID"`  // Debit note OID; the API expects this exact casing

	// APIUser names the authorising user. The endpoint answers a bare
	// {proposal, NoteOID} body with "Authorization is required", so it is always
	// sent; an empty value defaults to Config.APIUser, then Credentials.Username,
	// exactly as on InitiatePayment.
	APIUser string `json:"APIUser,omitempty"`
}

// GenerateCertificateResponse is returned by Client.GenerateCertificate.
//
// Like newProposal, this endpoint reports its outcome with a boolean "success"
// and puts the certificate at the top level rather than using Response.
type GenerateCertificateResponse struct {
	Success *FlexBool  `json:"success,omitempty"`
	CertNo  FlexString `json:"certNo,omitempty"` // DMVIC certificate number, e.g. C27609783

	// CertURL is a pre-signed link to the certificate PDF. It expires — the se=
	// query parameter is its expiry — so download and store the PDF rather than
	// persisting the URL.
	CertURL FlexString      `json:"certURL,omitempty"`
	Message FlexString      `json:"message,omitempty"`
	Raw     json.RawMessage `json:"-"`
}

// Succeeded reports whether the certificate was generated. An absent success
// field counts as a success; only an explicit false is a failure.
func (r *GenerateCertificateResponse) Succeeded() bool {
	return r.Success == nil || bool(*r.Success)
}
