package definite

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// API lists the operations *Client provides, so callers can substitute a fake
// in tests.
type API interface {
	CheckClient(ctx context.Context, idNumber string) (*CheckClientResponse, error)
	CreateClient(ctx context.Context, req *CreateClientRequest) (*CreateClientResponse, error)
	GetProducts(ctx context.Context, forceRefresh bool) (*ProductsResponse, error)
	GetMakes(ctx context.Context, forceRefresh bool) (*MakesResponse, error)
	CalculatePremium(ctx context.Context, req *CalculatePremiumRequest) (*CalculatePremiumResponse, error)
	CheckIntermediary(ctx context.Context, identification string, details *CheckIntermediaryDetails) (*CheckIntermediaryResponse, error)
	RegisterIntermediary(ctx context.Context, req *RegisterIntermediaryRequest) (*RegisterIntermediaryResponse, error)
	CheckPolicy(ctx context.Context, registration string) (*CheckPolicyResponse, error)
	CreateProposal(ctx context.Context, req *NewProposalRequest) (*NewProposalResponse, error)
	InitiatePayment(ctx context.Context, req *InitiatePaymentRequest) (*InitiatePaymentResponse, error)
	GenerateCertificate(ctx context.Context, req *GenerateCertificateRequest) (*GenerateCertificateResponse, error)
	ClearCache()
}

var _ API = (*Client)(nil)

// CheckClient looks up an existing client by national ID number. Run it before
// CreateClient — the documented flow only creates a client when this finds none.
func (c *Client) CheckClient(ctx context.Context, idNumber string) (*CheckClientResponse, error) {
	const op = "CheckClient"
	id := strings.TrimSpace(idNumber)
	if id == "" {
		return nil, newInputError(op, errors.New("ID number is required"))
	}
	body, err := c.do(ctx, request{op: op, method: http.MethodGet, path: endpointCheckClient,
		query: url.Values{"IDNumber": {id}}})
	if err != nil {
		return nil, err
	}
	return decodeEnvelope[*ClientRecord](op, ErrCheckClient, "client lookup failed", body)
}

// CreateClient onboards a new client. The returned record's ID is what
// NewProposalRequest.Client takes.
func (c *Client) CreateClient(ctx context.Context, req *CreateClientRequest) (*CreateClientResponse, error) {
	const op = "CreateClient"
	if err := ValidateCreateClientRequest(req); err != nil {
		return nil, newInputError(op, err)
	}
	body, err := c.do(ctx, request{op: op, method: http.MethodPost, path: endpointCreateClient, body: req})
	if err != nil {
		return nil, err
	}
	return decodeEnvelope[*ClientRecord](op, ErrCreateClient, "client creation failed", body)
}

// GetProducts lists underwriting products and their cover options. The result
// is cached for Config.CacheTTL; forceRefresh bypasses the cache.
func (c *Client) GetProducts(ctx context.Context, forceRefresh bool) (*ProductsResponse, error) {
	return referenceData[[]Product](ctx, c, "GetProducts", cacheKeyProducts, endpointProducts,
		ErrProducts, "product retrieval failed", forceRefresh)
}

// GetMakes lists vehicle makes with their models nested. The result is cached
// for Config.CacheTTL; forceRefresh bypasses the cache.
func (c *Client) GetMakes(ctx context.Context, forceRefresh bool) (*MakesResponse, error) {
	return referenceData[[]VehicleMake](ctx, c, "GetMakes", cacheKeyMakes, endpointMakes,
		ErrMakes, "makes retrieval failed", forceRefresh)
}

// CalculatePremium quotes a premium. The documented request carries the same
// values in the query string and the JSON body, so both are sent.
func (c *Client) CalculatePremium(ctx context.Context, req *CalculatePremiumRequest) (*CalculatePremiumResponse, error) {
	const op = "CalculatePremium"
	if err := ValidateCalculatePremiumRequest(req); err != nil {
		return nil, newInputError(op, err)
	}
	q := url.Values{}
	q.Set("sumInsured", formatNumber(req.SumInsured))
	q.Set("capacity", strconv.Itoa(req.Capacity))
	q.Set("tonnage", formatNumber(req.Tonnage))
	q.Set("coverPeriod", req.CoverPeriod)
	q.Set("product", strconv.FormatInt(req.Product, 10))
	q.Set("coverOption", strconv.FormatInt(req.CoverOption, 10))
	if req.TransactionType != 0 {
		q.Set("transactionType", strconv.Itoa(req.TransactionType))
	}
	body, err := c.do(ctx, request{op: op, method: http.MethodGet, path: endpointCalculatePremium, query: q, body: req})
	if err != nil {
		return nil, err
	}
	return decodeEnvelope[*PremiumBreakdown](op, ErrCalculatePremium, "premium calculation failed", body)
}

// CheckIntermediary looks up an agent or broker by identification number.
// details, when non-nil, is sent as the request body. The returned record's ID
// is what NewProposalRequest.Agent takes.
func (c *Client) CheckIntermediary(ctx context.Context, identification string, details *CheckIntermediaryDetails) (*CheckIntermediaryResponse, error) {
	const op = "CheckIntermediary"
	id := strings.TrimSpace(identification)
	if id == "" {
		return nil, newInputError(op, errors.New("identification is required"))
	}
	r := request{op: op, method: http.MethodGet, path: endpointCheckIntermediary,
		query: url.Values{"identification": {id}}}
	if details != nil { // a typed nil must not become a "null" body
		r.body = details
	}
	body, err := c.do(ctx, r)
	if err != nil {
		return nil, err
	}
	return decodeEnvelope[*IntermediaryRecord](op, ErrCheckIntermediary, "intermediary lookup failed", body)
}

// RegisterIntermediary onboards a new intermediary.
func (c *Client) RegisterIntermediary(ctx context.Context, req *RegisterIntermediaryRequest) (*RegisterIntermediaryResponse, error) {
	const op = "RegisterIntermediary"
	if err := ValidateRegisterIntermediaryRequest(req); err != nil {
		return nil, newInputError(op, err)
	}
	body, err := c.do(ctx, request{op: op, method: http.MethodPost, path: endpointRegisterIntermediary, body: req})
	if err != nil {
		return nil, err
	}
	return decodeEnvelope[*IntermediaryRecord](op, ErrRegisterIntermediary, "intermediary registration failed", body)
}

// CheckPolicy looks up a policy by vehicle registration number, e.g. KBS264M.
func (c *Client) CheckPolicy(ctx context.Context, registration string) (*CheckPolicyResponse, error) {
	const op = "CheckPolicy"
	reg := strings.TrimSpace(registration)
	if reg == "" {
		return nil, newInputError(op, errors.New("registration number is required"))
	}
	body, err := c.do(ctx, request{op: op, method: http.MethodGet, path: endpointCheckPolicy,
		query: url.Values{"registration": {reg}}})
	if err != nil {
		return nil, err
	}
	return decodeEnvelope[*PolicyRecord](op, ErrCheckPolicy, "policy lookup failed", body)
}

// CreateProposal submits a new proposal.
//
// Check ProposalDetails on the result: its OID, InsuredItem and Note feed the
// payment and certificate calls. A rejected proposal returns an error tagged
// ErrNewProposal.
func (c *Client) CreateProposal(ctx context.Context, req *NewProposalRequest) (*NewProposalResponse, error) {
	const op = "CreateProposal"
	if err := ValidateNewProposalRequest(req); err != nil {
		return nil, newInputError(op, err)
	}
	body, err := c.do(ctx, request{op: op, method: http.MethodPost, path: endpointNewProposal, body: req})
	if err != nil {
		return nil, err
	}
	out := &NewProposalResponse{}
	if err := decodeBoolEnvelope(op, ErrNewProposal, "proposal creation failed", body, out); err != nil {
		return nil, err
	}
	out.Raw = body
	return out, nil
}

// InitiatePayment posts a payment against a proposal. An empty APIUser
// defaults to Config.APIUser, then Credentials.Username. req is not modified.
func (c *Client) InitiatePayment(ctx context.Context, req *InitiatePaymentRequest) (*InitiatePaymentResponse, error) {
	const op = "InitiatePayment"
	if err := ValidateInitiatePaymentRequest(req); err != nil {
		return nil, newInputError(op, err)
	}
	payload := *req
	if payload.APIUser == "" {
		payload.APIUser = c.cfg.APIUser
	}
	if payload.APIUser == "" {
		payload.APIUser = c.cfg.Credentials.Username
	}
	body, err := c.do(ctx, request{op: op, method: http.MethodPost, path: endpointInitiatePayment, body: &payload})
	if err != nil {
		return nil, err
	}
	return decodeEnvelope[*PaymentRecord](op, ErrInitiatePayment, "payment initiation failed", body)
}

// GenerateCertificate generates the DMVIC certificate for a paid proposal.
//
// Definite registers the certificate with DMVIC itself — CertNo is a DMVIC
// certificate number — so a policy issued this way must not also be issued
// directly through DMVIC, or it is certificated twice. CertURL expires:
// download the PDF rather than storing the link.
func (c *Client) GenerateCertificate(ctx context.Context, req *GenerateCertificateRequest) (*GenerateCertificateResponse, error) {
	const op = "GenerateCertificate"
	if err := ValidateGenerateCertificateRequest(req); err != nil {
		return nil, newInputError(op, err)
	}
	payload := *req
	if payload.APIUser == "" {
		payload.APIUser = c.cfg.APIUser
	}
	if payload.APIUser == "" {
		payload.APIUser = c.cfg.Credentials.Username
	}
	body, err := c.do(ctx, request{op: op, method: http.MethodPost, path: endpointGenerateCertificate, body: &payload})
	if err != nil {
		return nil, err
	}
	out := &GenerateCertificateResponse{}
	if err := decodeBoolEnvelope(op, ErrGenerateCertificate, "certificate generation failed", body, out); err != nil {
		return nil, err
	}
	out.Raw = body
	return out, nil
}

// formatNumber renders a float without an exponent or trailing zeros: 1500000, 2.5.
func formatNumber(x float64) string { return strconv.FormatFloat(x, 'f', -1, 64) }
