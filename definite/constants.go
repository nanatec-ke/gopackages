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
	endpointCheckClient          = "/api/v1/checkClient"          // GET
	endpointCreateClient         = "/api/v1/createClient"         // POST
	endpointProducts             = "/api/v1/products"             // GET
	endpointMakes                = "/api/v1/makes"                // GET
	endpointCalculatePremium     = "/api/v1/calculatePremium"     // GET (query + body)
	endpointCheckIntermediary    = "/api/v1/checkIntermediary"    // GET (query + optional body)
	endpointRegisterIntermediary = "/api/v1/registerIntermediary" // POST
	endpointCheckPolicy          = "/api/v1/checkPolicy"          // GET
	endpointNewProposal          = "/api/v1/newProposal"          // POST
	endpointInitiatePayment      = "/api/v1/initiatePayment"      // POST
	endpointGenerateCertificate  = "/api/v1/generateCertificate"  // POST
)

// Base URLs for each environment.
//
// The published documentation covers UAT only. The production host is the
// conventional counterpart and is unconfirmed — get the real host from Definite
// and set Config.BaseURL if it differs.
const (
	BaseURLUAT        = "https://uat.definiteassurance.com"
	BaseURLProduction = "https://api.definiteassurance.com"
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
