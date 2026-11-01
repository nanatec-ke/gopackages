package directline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// msisdnRe matches the 12-digit 2547XXXXXXXX form Directline documents for an
// STK push. A push sent to a malformed number is accepted and then silently
// never delivered, so the shape is checked rather than only non-emptiness.
var msisdnRe = regexp.MustCompile(`^254\d{9}$`)

func (c *Client) CalculatePremium(ctx context.Context, req *CalculatePremiumRequest) (*CalculatePremiumResponse, error) {
	var out CalculatePremiumResponse
	_, err := c.doAuthJSON(ctx, "POST", "/calculate-premium", mustMarshal(req), &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetBodyTypes(ctx context.Context) ([]BodyType, error) {
	var out []BodyType
	_, err := c.doAuthJSON(ctx, "GET", "/body-types", nil, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) CreateTransaction(ctx context.Context, endpoint string, req *TransactionRequest) (*TransactionResponse, *ValidationErrorResponse, error) {
	var out TransactionResponse
	path := fmt.Sprintf("/api/bima/%s", endpoint)
	status, err := c.doAuthJSON(ctx, http.MethodPost, path, mustMarshal(req), &out)
	if err != nil {
		// If it's an APIError representing a validation (HTTP 400), try to unmarshal validation details
		if apiErr, ok := err.(*APIError); ok && apiErr.HTTPCode == http.StatusBadRequest {
			var vr ValidationErrorResponse
			// try to unmarshal the last body from doAuthJSON? We don't return it — instead caller should inspect apiErr.Message
			_ = json.Unmarshal([]byte(apiErr.Message), &vr)
			return nil, &vr, apiErr
		}
		return nil, nil, err
	}
	if status == http.StatusOK || status == http.StatusCreated {
		return &out, nil, nil
	}
	return nil, nil, newExternalError("CreateTransaction", ErrHTTPRequest, "unexpected status", status, c.IsDebug())
}

// CalculatePremiumDetailed posts to /api/premium/calculate and returns PremiumResponse on 200 or PremiumErrorResponse on 400
func (c *Client) CalculatePremiumDetailed(ctx context.Context, req *PremiumRequest) (*PremiumResponse, *PremiumErrorResponse, error) {
	var out PremiumResponse
	status, err := c.doAuthJSON(ctx, http.MethodPost, "/api/premium/calculate", mustMarshal(req), &out)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok && apiErr.HTTPCode == http.StatusBadRequest {
			var perr PremiumErrorResponse
			_ = json.Unmarshal([]byte(apiErr.Message), &perr)
			return nil, &perr, apiErr
		}
		return nil, nil, err
	}
	if status == http.StatusOK || status == http.StatusCreated {
		return &out, nil, nil
	}
	return nil, nil, newExternalError("CalculatePremiumDetailed", ErrHTTPRequest, "unexpected status", status, c.IsDebug())
}

// GetBodyTypesFor retrieves available body types for a vehicleTypeId and caches results for 1 hour
func (c *Client) GetBodyTypesFor(ctx context.Context, vehicleTypeId int) ([]BodyType, error) {
	// check cache
	if bts, ok := c.bodyTypes.Get(vehicleTypeId); ok {
		return bts, nil
	}
	// not cached: request
	path := fmt.Sprintf("/api/premium/bodytypes/%d", vehicleTypeId)
	var resp BodyTypesResponse
	status, err := c.doAuthJSON(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}
	if status == http.StatusNotFound {
		return nil, newExternalError("GetBodyTypesFor", ErrHTTPRequest, fmt.Sprintf("no body types for vehicleTypeId %d", vehicleTypeId), status, c.IsDebug())
	}
	if status != http.StatusOK {
		return nil, newExternalError("GetBodyTypesFor", ErrHTTPRequest, "unexpected status", status, c.IsDebug())
	}
	// map response to []BodyType
	out := make([]BodyType, 0, len(resp.BodyTypes))
	for _, b := range resp.BodyTypes {
		out = append(out, BodyType{ID: b.BodyTypeID, Name: b.BodyTypeName})
	}
	// cache for 1 hour
	c.bodyTypes.Set(vehicleTypeId, out, 1*time.Hour)
	return out, nil
}

// Convenience wrappers for common endpoints
func (c *Client) CreateBusComprehensive(ctx context.Context, req *TransactionRequest) (*TransactionResponse, *ValidationErrorResponse, error) {
	return c.CreateTransaction(ctx, EndpointBusComp, req)
}

func (c *Client) CreateBusTPO(ctx context.Context, req *TransactionRequest) (*TransactionResponse, *ValidationErrorResponse, error) {
	return c.CreateTransaction(ctx, EndpointBusTPO, req)
}

func (c *Client) CreatePrivateComprehensive(ctx context.Context, req *TransactionRequest) (*TransactionResponse, *ValidationErrorResponse, error) {
	return c.CreateTransaction(ctx, EndpointPrivateComp, req)
}

func (c *Client) CreatePrivateTPO(ctx context.Context, req *TransactionRequest) (*TransactionResponse, *ValidationErrorResponse, error) {
	return c.CreateTransaction(ctx, EndpointPrivateTPO, req)
}

// New convenience wrappers added for other vehicle/product types
func (c *Client) CreateMatatuComprehensive(ctx context.Context, req *TransactionRequest) (*TransactionResponse, *ValidationErrorResponse, error) {
	return c.CreateTransaction(ctx, EndpointMatatuComp, req)
}

func (c *Client) CreateMatatuTPO(ctx context.Context, req *TransactionRequest) (*TransactionResponse, *ValidationErrorResponse, error) {
	return c.CreateTransaction(ctx, EndpointMatatuTPO, req)
}

func (c *Client) CreateTaxiComprehensive(ctx context.Context, req *TransactionRequest) (*TransactionResponse, *ValidationErrorResponse, error) {
	return c.CreateTransaction(ctx, EndpointTaxiComp, req)
}

func (c *Client) CreateTaxiTPO(ctx context.Context, req *TransactionRequest) (*TransactionResponse, *ValidationErrorResponse, error) {
	return c.CreateTransaction(ctx, EndpointTaxiTPO, req)
}

func (c *Client) CreatePrivateHireComprehensive(ctx context.Context, req *TransactionRequest) (*TransactionResponse, *ValidationErrorResponse, error) {
	return c.CreateTransaction(ctx, EndpointPrivateHireComp, req)
}

func (c *Client) CreatePrivateHireTPO(ctx context.Context, req *TransactionRequest) (*TransactionResponse, *ValidationErrorResponse, error) {
	return c.CreateTransaction(ctx, EndpointPrivateHireTPO, req)
}

func (c *Client) CreateOwnGoodsComprehensive(ctx context.Context, req *TransactionRequest) (*TransactionResponse, *ValidationErrorResponse, error) {
	return c.CreateTransaction(ctx, EndpointOwnGoodsComp, req)
}

func (c *Client) CreateOwnGoodsTPO(ctx context.Context, req *TransactionRequest) (*TransactionResponse, *ValidationErrorResponse, error) {
	return c.CreateTransaction(ctx, EndpointOwnGoodsTPO, req)
}

func (c *Client) CreateInstitutionalComprehensive(ctx context.Context, req *TransactionRequest) (*TransactionResponse, *ValidationErrorResponse, error) {
	return c.CreateTransaction(ctx, EndpointInstitutionalComp, req)
}

func (c *Client) CreateInstitutionalTPO(ctx context.Context, req *TransactionRequest) (*TransactionResponse, *ValidationErrorResponse, error) {
	return c.CreateTransaction(ctx, EndpointInstitutionalTPO, req)
}

func (c *Client) CreatePrimeMoverComprehensive(ctx context.Context, req *TransactionRequest) (*TransactionResponse, *ValidationErrorResponse, error) {
	return c.CreateTransaction(ctx, EndpointPrimeMoverComp, req)
}

func (c *Client) CreatePrimeMoverTPO(ctx context.Context, req *TransactionRequest) (*TransactionResponse, *ValidationErrorResponse, error) {
	return c.CreateTransaction(ctx, EndpointPrimeMoverTPO, req)
}

func (c *Client) CreateTrailerComprehensive(ctx context.Context, req *TransactionRequest) (*TransactionResponse, *ValidationErrorResponse, error) {
	return c.CreateTransaction(ctx, EndpointTrailerComp, req)
}

func (c *Client) CreateTrailerTPO(ctx context.Context, req *TransactionRequest) (*TransactionResponse, *ValidationErrorResponse, error) {
	return c.CreateTransaction(ctx, EndpointTrailerTPO, req)
}

func (c *Client) CreateTankersComprehensive(ctx context.Context, req *TransactionRequest) (*TransactionResponse, *ValidationErrorResponse, error) {
	return c.CreateTransaction(ctx, EndpointTankersComp, req)
}

func (c *Client) CreateTankersTPO(ctx context.Context, req *TransactionRequest) (*TransactionResponse, *ValidationErrorResponse, error) {
	return c.CreateTransaction(ctx, EndpointTankersTPO, req)
}

func (c *Client) CreateBikeComprehensive(ctx context.Context, req *TransactionRequest) (*TransactionResponse, *ValidationErrorResponse, error) {
	return c.CreateTransaction(ctx, EndpointBikeComp, req)
}

func (c *Client) CreateBikeTPO(ctx context.Context, req *TransactionRequest) (*TransactionResponse, *ValidationErrorResponse, error) {
	return c.CreateTransaction(ctx, EndpointBikeTPO, req)
}

func (c *Client) CreateBodaComprehensive(ctx context.Context, req *TransactionRequest) (*TransactionResponse, *ValidationErrorResponse, error) {
	return c.CreateTransaction(ctx, EndpointBodaComp, req)
}

func (c *Client) CreateBodaTPO(ctx context.Context, req *TransactionRequest) (*TransactionResponse, *ValidationErrorResponse, error) {
	return c.CreateTransaction(ctx, EndpointBodaTPO, req)
}

func (c *Client) CreateGeneralCartageComprehensive(ctx context.Context, req *TransactionRequest) (*TransactionResponse, *ValidationErrorResponse, error) {
	return c.CreateTransaction(ctx, EndpointGeneralCartageComp, req)
}

func (c *Client) CreateGeneralCartageTPO(ctx context.Context, req *TransactionRequest) (*TransactionResponse, *ValidationErrorResponse, error) {
	return c.CreateTransaction(ctx, EndpointGeneralCartageTPO, req)
}

func (c *Client) CreateTractorComprehensive(ctx context.Context, req *TransactionRequest) (*TransactionResponse, *ValidationErrorResponse, error) {
	return c.CreateTransaction(ctx, EndpointTractorComp, req)
}

func (c *Client) CreateTractorTPO(ctx context.Context, req *TransactionRequest) (*TransactionResponse, *ValidationErrorResponse, error) {
	return c.CreateTransaction(ctx, EndpointTractorTPO, req)
}

func (c *Client) CreateFlammableTankersComprehensive(ctx context.Context, req *TransactionRequest) (*TransactionResponse, *ValidationErrorResponse, error) {
	return c.CreateTransaction(ctx, EndpointFlammableTankersComp, req)
}

func (c *Client) CreateFlammableTankersTPO(ctx context.Context, req *TransactionRequest) (*TransactionResponse, *ValidationErrorResponse, error) {
	return c.CreateTransaction(ctx, EndpointFlammableTankersTPO, req)
}

// --- Agent wallet ---

// validateAgentTopUpRequest rejects a top-up payload the API cannot accept,
// before anything is sent.
func (c *Client) validateAgentTopUpRequest(op string, req *AgentTopUpRequest) error {
	if req == nil {
		return newInternalError(op, ErrInvalidInput, errors.New("top up request is required"), c.IsDebug())
	}
	if !msisdnRe.MatchString(req.Number) {
		return newInternalError(op, ErrInvalidInput, errors.New("number must be a 12-digit phone number in 2547XXXXXXXX form"), c.IsDebug())
	}
	if strings.TrimSpace(req.Account) == "" {
		return newInternalError(op, ErrInvalidInput, errors.New("account is required"), c.IsDebug())
	}
	if req.Amount <= 0 {
		return newInternalError(op, ErrInvalidInput, errors.New("amount must be a whole number of KES greater than zero"), c.IsDebug())
	}
	return nil
}

// TopUpAgent sends an M-PESA STK push to fund an agent's wallet, via
// POST /api/pay/stk.
//
// The returned response means the prompt was accepted for delivery, not that
// the wallet has been funded — confirm with GetAgentBalance. A 200 carrying no
// checkout ID is reported as a success for the same reason: resubmitting would
// prompt the subscriber twice.
//
// req.Account must match the agent code in the bearer token; a mismatch answers
// 400, and an agent code Directline does not know answers 401.
func (c *Client) TopUpAgent(ctx context.Context, req *AgentTopUpRequest) (*AgentTopUpResponse, error) {
	const op = "TopUpAgent"
	if err := c.validateAgentTopUpRequest(op, req); err != nil {
		return nil, err
	}
	var out AgentTopUpResponse
	if _, err := c.doAuthJSON(ctx, http.MethodPost, "/api/pay/stk", mustMarshal(req), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetAgentBalance reads an agent's current wallet balance, via
// POST /api/agent/balance.
//
// The result is never cached: a balance moves with every transaction, so each
// call is a fresh read. agentCode must match the agent code in the bearer
// token, or Directline answers 400; one it does not know answers 401.
func (c *Client) GetAgentBalance(ctx context.Context, agentCode int) (*AgentBalance, error) {
	const op = "GetAgentBalance"
	if agentCode <= 0 {
		return nil, newInternalError(op, ErrInvalidInput, errors.New("agentCode must be a positive agent code"), c.IsDebug())
	}

	var out AgentBalanceResponse
	payload := mustMarshal(&AgentBalanceRequest{AgentCode: agentCode})
	if _, err := c.doAuthJSON(ctx, http.MethodPost, "/api/agent/balance", payload, &out); err != nil {
		return nil, err
	}

	// A response carrying no balance is an error rather than a zero balance, so
	// a rejected or empty answer cannot pass for an empty wallet.
	if out.Balance == nil {
		msg := out.Message
		if msg == "" {
			msg = fmt.Sprintf("no balance returned for agent code %d", agentCode)
		}
		return nil, newInternalError(op, ErrUnmarshalResponse, errors.New(msg), c.IsDebug())
	}

	code := out.AgentCode
	if code == 0 {
		code = agentCode
	}
	return &AgentBalance{AgentCode: code, Balance: *out.Balance}, nil
}

func mustMarshal(v interface{}) []byte {
	b, _ := jsonMarshal(v)
	return b
}

func jsonMarshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}
