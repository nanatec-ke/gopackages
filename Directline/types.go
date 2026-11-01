package directline

import "time"

// TokenResponse models the token generation response from Directline
type TokenResponse struct {
	Success     bool   `json:"success"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	Message     string `json:"message"`
}

// CreateTransactionRequest minimal sample
type CreateTransactionRequest struct {
	PolicyRef string `json:"policyRef,omitempty"`
	Premium   int    `json:"premium,omitempty"`
}

type CreateTransactionResponse struct {
	Success bool   `json:"success"`
	ID      string `json:"id,omitempty"`
	Message string `json:"message,omitempty"`
}

// CalculatePremium request/response
type CalculatePremiumRequest struct {
	VehicleType string `json:"vehicleType"`
	SumInsured  int    `json:"sumInsured"`
}

type CalculatePremiumResponse struct {
	Success bool    `json:"success"`
	Premium float64 `json:"premium"`
}

// BodyType represents a vehicle body type
type BodyType struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// TokenCacheEntry is used to store token and expiry
type TokenCacheEntry struct {
	Token     string
	ExpiresAt time.Time
}

// TransactionRequest matches Directline Create Transaction API
type TransactionRequest struct {
	TransactionCode      interface{} `json:"transactionCode"` // decimal/number
	CoverStartDate       string      `json:"coverStartDate"`
	Period               int         `json:"period"`
	PlateNumber          string      `json:"plateNumber"`
	ChasisNo             string      `json:"chasisNo"`
	Make                 string      `json:"make"`
	Model                string      `json:"model"`
	YrManft              string      `json:"yrManft"`
	CubicCapacity        string      `json:"cubicCapacity"`
	CarryCapacity        int         `json:"carryCapacity"`
	Tonnage              int         `json:"tonnage"`
	AssetValue           float64     `json:"assetValue"`
	BodyTypeID           int         `json:"bodyTypeID"`
	CustomerFullName     string      `json:"customerFullName"`
	IDNumber             string      `json:"idNumber"`
	Email                string      `json:"email"`
	MSISDN               string      `json:"msisdn"`
	PIN                  string      `json:"pin"`
	PostalAddress        string      `json:"postalAddress"`
	Town                 string      `json:"town"`
	ClientType           string      `json:"clientType"`
	RequestType          string      `json:"requestType"`
	Amount               float64     `json:"amount"`
	PaymentMode          string      `json:"paymentMode"`
	AgentCode            int         `json:"agentCode"`
	APICode              string      `json:"apI_Code"`
	PVT                  string      `json:"pvt"`
	EPR                  string      `json:"epr"`
	BimaPlus             int         `json:"bimaplus"`
	CoverType            string      `json:"coverType,omitempty"`
	CoverTypeId          int         `json:"coverTypeId,omitempty"`
	VehicleType          string      `json:"vehicleType,omitempty"`
	VehicleTypeId        int         `json:"vehicleTypeId,omitempty"`
	BodyType             string      `json:"bodyType,omitempty"`
	ResolvedBodyTypeName string      `json:"resolvedBodyTypeName,omitempty"`
}

// TransactionResponse models a successful transaction response
type TransactionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		CoverEndDate  string  `json:"coverEndDate"`
		PlateNumber   string  `json:"plateNumber"`
		EngineNo      string  `json:"engineNo"`
		CubicCapacity string  `json:"cubicCapacity"`
		ChasisNo      string  `json:"chasisNo"`
		Amount        float64 `json:"amount"`
	} `json:"data"`
}

// ValidationErrorResponse models a 400 validation response
type ValidationErrorResponse struct {
	Success            bool     `json:"success"`
	Errors             []string `json:"errors"`
	Message            string   `json:"message"`
	AvailableBodyTypes []struct {
		BodyTypeID   int    `json:"bodyTypeId"`
		BodyTypeName string `json:"bodyTypeName"`
	} `json:"availableBodyTypes"`
}

// --- Premium calculation types ---
// PremiumRequest models the /api/premium/calculate request parameters
type PremiumRequest struct {
	VehicleTypeID int     `json:"vehicleTypeID"`
	CarryCapacity int     `json:"carryCapacity"`
	Tonnage       int     `json:"tonnage"`
	CoverTypeID   int     `json:"coverTypeID"`
	Period        int     `json:"period"`
	AssetValue    float64 `json:"assetValue,omitempty"`
	BodyTypeID    int     `json:"bodyTypeID"`
	PVT           string  `json:"pvt,omitempty"`
	EPR           string  `json:"epr,omitempty"`
	AgentCode     int     `json:"agentCode"`
	RequestType   string  `json:"requestType"`
}

// PremiumResponse models a successful premium calculation response
type PremiumResponse struct {
	Success      bool    `json:"success"`
	BasePremium  float64 `json:"basePremium"`
	TotalPremium float64 `json:"totalPremium"`
	NewPolicyFee float64 `json:"newPolicyFee"`
	FinalAmount  float64 `json:"finalAmount"`
	BodyTypeName string  `json:"bodyTypeName"`
	BreakDown    struct {
		BasePremium  float64 `json:"basePremium"`
		Levies       float64 `json:"levies"`
		NewPolicyFee float64 `json:"newPolicyFee"`
		Total        float64 `json:"total"`
	} `json:"breakDown"`
}

// PremiumErrorResponse models a 400 error for premium calculation
type PremiumErrorResponse struct {
	Success            bool     `json:"success"`
	Message            string   `json:"message"`
	Errors             []string `json:"errors"`
	AvailableBodyTypes []struct {
		BodyTypeID   int    `json:"bodyTypeId"`
		BodyTypeName string `json:"bodyTypeName"`
	} `json:"availableBodyTypes"`
}

// BodyTypesResponse models the /api/premium/bodytypes/{vehicleTypeId} response
type BodyTypesResponse struct {
	Success       bool `json:"success"`
	VehicleTypeId int  `json:"vehicleTypeId"`
	BodyTypes     []struct {
		BodyTypeID   int    `json:"bodyTypeId"`
		BodyTypeName string `json:"bodyTypeName"`
	} `json:"bodyTypes"`
}

// --- Agent wallet types ---

// AgentTopUpRequest models the /api/pay/stk request, which sends an M-PESA STK
// push to fund an agent's wallet.
type AgentTopUpRequest struct {
	// Number is the handset prompted, in 2547XXXXXXXX form. It need not be the
	// agent's own line — any subscriber can fund the wallet named by Account.
	Number string `json:"number"`
	// Account is the agent code to credit. It must match the agent code in the
	// bearer token, or Directline answers 400.
	Account string `json:"account"`
	// Amount is the top-up in whole KES.
	Amount int `json:"amount"`
}

// AgentTopUpResponse models a successful /api/pay/stk response.
//
// A response means the prompt was accepted for delivery, not that the wallet
// has been funded: the credit lands only once the subscriber approves the
// prompt on the handset. Confirm with GetAgentBalance rather than treating a
// returned response as settled.
type AgentTopUpResponse struct {
	// Message is the outcome description, e.g. "Submission successful".
	Message string `json:"message"`
	// ID is the M-PESA checkout request ID, e.g. "ws_CO_12345678". It is absent
	// when the API answers 200 without one, which is not a failure.
	ID string `json:"id"`
}

// AgentBalanceRequest models the /api/agent/balance request.
type AgentBalanceRequest struct {
	AgentCode int `json:"agentCode"`
}

// AgentBalanceResponse models the raw /api/agent/balance response.
//
// Balance is a pointer so an omitted balance stays distinguishable from a
// genuine zero one.
type AgentBalanceResponse struct {
	AgentCode int      `json:"agentCode"`
	Balance   *float64 `json:"balance"`
	Message   string   `json:"message"`
}

// AgentBalance is an agent's wallet balance, normalised from
// AgentBalanceResponse so Balance is always present.
type AgentBalance struct {
	// AgentCode the balance belongs to.
	AgentCode int
	// Balance available, in KES.
	Balance float64
}
