package definite

import (
	"errors"
	"fmt"
	"math"
	"strings"
)

// The validators run automatically inside each Client method, before any
// request is sent, and are exported so callers can validate input up front —
// at a form boundary, say. Inside the client their errors are wrapped in a
// *ClientError with Code ErrInvalidInput.

// ValidateCreateClientRequest validates a createClient payload.
func ValidateCreateClientRequest(req *CreateClientRequest) error {
	if req == nil {
		return errors.New("create client request is required")
	}
	if blank(req.FirstName) {
		return errors.New("firstName is required")
	}
	if blank(req.LastName) {
		return errors.New("lastName is required")
	}
	if blank(req.IDNumber) {
		return errors.New("idNumber is required")
	}
	if blank(req.PIN) {
		return errors.New("pin (KRA PIN) is required")
	}
	if blank(req.Email) {
		return errors.New("email is required")
	}
	if !IsValidEmail(req.Email) {
		return fmt.Errorf("invalid email address: %s", req.Email)
	}
	if blank(req.Phone) {
		return errors.New("phone is required")
	}
	return nil
}

// ValidateCalculatePremiumRequest validates calculatePremium inputs.
func ValidateCalculatePremiumRequest(req *CalculatePremiumRequest) error {
	if req == nil {
		return errors.New("premium calculation request is required")
	}
	if req.Product <= 0 {
		return errors.New("product must be a positive number")
	}
	if req.CoverOption <= 0 {
		return errors.New("coverOption must be a positive number")
	}
	if blank(req.CoverPeriod) {
		return errors.New("coverPeriod is required")
	}
	if !nonNegative(req.SumInsured) {
		return errors.New("sumInsured must be zero or greater")
	}
	if req.Capacity < 0 {
		return errors.New("capacity must be zero or greater")
	}
	if !nonNegative(req.Tonnage) {
		return errors.New("tonnage must be zero or greater")
	}
	return nil
}

// ValidateRegisterIntermediaryRequest validates a registerIntermediary payload.
func ValidateRegisterIntermediaryRequest(req *RegisterIntermediaryRequest) error {
	if req == nil {
		return errors.New("register intermediary request is required")
	}
	if blank(req.Name) {
		return errors.New("name is required")
	}
	if blank(req.Email) {
		return errors.New("email is required")
	}
	if !IsValidEmail(req.Email) {
		return fmt.Errorf("invalid email address: %s", req.Email)
	}
	if blank(req.Phone) {
		return errors.New("phone is required")
	}
	if blank(req.Identification) {
		return errors.New("identification is required")
	}
	if blank(req.KRAPin) {
		return errors.New("krapin is required")
	}
	return nil
}

// ValidateNewProposalRequest validates a newProposal payload.
func ValidateNewProposalRequest(req *NewProposalRequest) error {
	if req == nil {
		return errors.New("proposal request is required")
	}
	if req.Client <= 0 {
		return errors.New("Client must be a positive client record ID")
	}
	if req.Agent == "" || strings.TrimSpace(req.Agent) == "" {
		return errors.New("agent must be a positive intermediary record ID")
	}
	if req.CommencementDate.IsZero() {
		return errors.New("CommencementDate is required")
	}
	if blank(req.PaymentInterval) {
		return errors.New("PaymentInterval is required")
	}
	if req.Product <= 0 {
		return errors.New("Product must be a positive number")
	}
	if req.CoverOption <= 0 {
		return errors.New("CoverOption must be a positive number")
	}
	if blank(req.Registration) {
		return errors.New("Registration is required")
	}
	if req.MakeCode <= 0 {
		return errors.New("MakeCode must be a positive number")
	}
	if req.ModelCode <= 0 {
		return errors.New("ModelCode must be a positive number")
	}
	if req.YOM < 1900 {
		return errors.New("YOM must be a valid year of manufacture")
	}
	if blank(req.ChassisNumber) {
		return errors.New("ChassisNumber is required")
	}
	if blank(req.ClientID) {
		return errors.New("ClientID (national ID number) is required")
	}
	return nil
}

// ValidateInitiatePaymentRequest validates an initiatePayment payload.
func ValidateInitiatePaymentRequest(req *InitiatePaymentRequest) error {
	if req == nil {
		return errors.New("payment request is required")
	}
	if req.Proposal <= 0 {
		return errors.New("Proposal must be a positive proposal record ID")
	}
	if req.InsuredItem <= 0 {
		return errors.New("InsuredItem must be a positive insured item record ID")
	}
	
	if blank(req.PhoneNumber) {
		return errors.New("PhoneNumber is required")
	}
	if req.PaymentMethod < 0 {
		return errors.New("PaymentMethod must be zero or greater")
	}
	if !nonNegative(req.AmountPaid) || req.AmountPaid == 0 {
		return errors.New("AmountPaid must be greater than zero")
	}
	return nil
}

// ValidateGenerateCertificateRequest validates a generateCertificate payload.
func ValidateGenerateCertificateRequest(req *GenerateCertificateRequest) error {
	if req == nil {
		return errors.New("generate certificate request is required")
	}
	if req.Proposal <= 0 {
		return errors.New("proposal must be a positive proposal record ID")
	}
	if req.NoteOID <= 0 {
		return errors.New("NoteOID must be a positive debit note OID")
	}
	return nil
}

func blank(s string) bool { return strings.TrimSpace(s) == "" }

// nonNegative rejects negatives, NaN and ±Inf — the last two would also fail JSON encoding.
func nonNegative(x float64) bool { return !math.IsNaN(x) && !math.IsInf(x, 0) && x >= 0 }
