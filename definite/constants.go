package definite

import "time"

// Premium payment intervals accepted by the newProposal endpoint (PaymentInterval)
// and the premium calculator (coverPeriod). The API uses the same vocabulary for both.
const (
	PaymentIntervalAnnually        = "Annually"      // Single up-front annual premium
	PaymentIntervalMonthly         = "Monthly"       // Twelve monthly instalments
	PaymentIntervalQuarterly       = "Quarterly"     // Four quarterly instalments
	PaymentIntervalTwoInstallments = "2installments" // Two instalments over the period
)

// Payment method codes used by the initiatePayment endpoint.
//
// The published API documentation only exercises 0 (M-Pesa). Additional codes may
// exist — pass the numeric value directly if your integration uses one not listed here.
const (
	PaymentMethodMpesa = 0
)

// Transaction type codes used by the calculatePremium endpoint.
// Only 1 (new business) appears in the published documentation.
const (
	TransactionTypeNewBusiness = 1
)

// Status strings returned in the "status" field of the read endpoints' envelope.
const (
	ResponseStatusSuccess = "success"
	ResponseStatusError   = "error"
)

// API endpoint paths, relative to the base URL. All require HTTP Basic authentication.
const (
	endpointCheckClient          = "/checkClient"          // GET
	endpointCreateClient         = "/createClient"         // POST
	endpointProducts             = "/products"             // GET
	endpointMakes                = "/makes"                // GET
	endpointCalculatePremium     = "/calculatePremium"     // GET (query + body)
	endpointCheckIntermediary    = "/checkIntermediary"    // GET (query + optional body)
	endpointRegisterIntermediary = "/registerIntermediary" // POST
	endpointCheckPolicy          = "/checkPolicy"          // GET
	endpointNewProposal          = "/newProposal"          // POST
	endpointInitiatePayment      = "/initiatePayment"
	endpointCheckWalletBalance   = "/checkWalletBalance"  // GET
	endpointTopUpWallet          = "/topUpWallet"         // POST
	endpointGenerateCertificate  = "/generateCertificate" // POST
	endpointInitiateExtension    = "/initiateExtension"   // POST
	endpointPayForExtension      = "/PayForExtension"     // POST — capitalised, unlike the others
)

// Base URLs for each environment.
//
// The published documentation covers UAT only. The production host is the
// conventional counterpart and is unconfirmed — get the real host from Definite
// and set Config.BaseURL if it differs.
const (
	BaseURLUAT        = "https://uat.definiteassurance.com"
	BaseURLProduction = "https://digitalapi.definiteassurance.com"
)

// Defaults applied by NewClient when the corresponding Config field is zero.
const (
	DefaultTimeout  = 30 * time.Second
	DefaultCacheTTL = time.Hour
)

// DateLayout is the timestamp format the API requires: YYYY-MM-DD HH:MM:SS.mmm.
// See FormatDate and ParseDate.
const DateLayout = "2006-01-02 15:04:05.000"

// userAgent identifies this SDK on every request.
const userAgent = "definite-go"

// Cache keys for reference data.
const (
	cacheKeyProducts = "products"
	cacheKeyMakes    = "makes"
)
