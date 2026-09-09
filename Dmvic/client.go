package dmvic

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

// Client defines the interface for DMVIC operations.
// It provides methods for authentication, certificate management, insurance validation,
// and other DMVIC API operations.
type Client interface {
	// Login authenticates with the DMVIC API and obtains an access token.
	// Returns an error if authentication fails.
	Login() error

	// GetCertificate retrieves certificate information by certificate number.
	// Returns the certificate response or an error if the operation fails.
	GetCertificate(certificateNumber string) (*CertificateResponse, error)

	// CancelCertificate cancels an existing certificate with the specified reason.
	// reasonID represents the cancellation reason code.
	CancelCertificate(certificateNumber string, reasonID int) (*CancellationResponse, error)

	// ValidateInsurance validates insurance information against DMVIC records.
	ValidateInsurance(req *InsuranceValidationRequest) (*InsuranceValidationResponse, error)

	// ValidateDoubleInsurance checks for duplicate insurance coverage.
	ValidateDoubleInsurance(req *DoubleInsuranceRequest) (*DoubleInsuranceResponse, error)

	// IssueTypeACertificate issues a Type A insurance certificate.
	IssueTypeACertificate(req *TypeAIssuanceRequest) (*InsuranceResponse, error)

	// IssueTypeBCertificate issues a Type B insurance certificate.
	IssueTypeBCertificate(req *TypeBIssuanceRequest) (*InsuranceResponse, error)

	// IssueTypeCCertificate issues a Type C insurance certificate.
	IssueTypeCCertificate(req *TypeCIssuanceRequest) (*InsuranceResponse, error)

	// IssueTypeDCertificate issues a Type D insurance certificate.
	IssueTypeDCertificate(req *TypeDIssuanceRequest) (*InsuranceResponse, error)

	// ConfirmCertificateIssuance confirms the issuance of a certificate.
	ConfirmCertificateIssuance(req *ConfirmationRequest) (*InsuranceResponse, error)

	// GetMemberCompanyStock retrieves stock information for a member company.
	GetMemberCompanyStock(memberCompanyID int) (*StockResponse, error)

	// GetMemberIntermediaryStock retrieves stock information for a member
	// intermediary.
	GetMemberIntermediaryStock(memberIntermediaryID int) (*StockResponse, error)

	// GetIntermediaryStock retrieves certificate stock for one or more
	// intermediaries by their IRA numbers (DMVIC 4.8.2 IntermediaryStock).
	GetIntermediaryStock(iraNumbers []string) (*StockResponse, error)

	// GetToken returns the current authentication token.
	GetToken() string

	// IsTokenValid checks if the current token is valid and not expired.
	IsTokenValid() bool

	// secureRequest creates a secure HTTP request with proper TLS configuration.
	secureRequest(method, url string, jsonPayload []byte) (*http.Client, *http.Request, error)

	// normalRequest creates a standard HTTP request without special security configurations.
	normalRequest(method, url string, jsonPayload []byte) (*http.Client, *http.Request, error)
}

// client implements the Client interface for DMVIC API operations.
// It maintains configuration, HTTP client, authentication tokens, and endpoint information.
type client struct {
	config     *Config                   // Configuration settings for the client
	httpClient *http.Client              // HTTP client for making requests
	endpoint   string                    // Base endpoint URL for DMVIC API
	tknStorage *TTLCache[string, string] // Token storage with TTL functionality
}

// NewClient creates a new DMVIC client instance with the provided configuration.
// It validates the configuration and sets up the HTTP client with appropriate TLS settings.
// Returns a Client interface implementation or an error if configuration is invalid.
func NewClient(config *Config) (Client, error) {

	if err := config.Validate(); err != nil {
		return nil, &ClientError{
			Type:      InternalError,
			Code:      ErrInvalidConfig,
			Message:   err.Error(),
			Operation: "NewClient",
		}
	}
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.InsecureSkipVerify,
		},
	}
	httpClient := &http.Client{
		Timeout:   config.Timeout,
		Transport: transport,
	}
	tknStorage := NewTTL[string, string](config.TokenTTL) // 24 hours TTL
	return &client{
		config:     config,
		httpClient: httpClient,
		endpoint:   config.GetEndpoint(),
		tknStorage: tknStorage,
	}, nil
}

// debugLog outputs debug information if debug mode is enabled in the configuration.
// It prefixes all log messages with "[DMVIC DEBUG]" for easy identification.
func (c *client) debugLog(format string, args ...interface{}) {
	if c.config.Debug {
		log.Printf("[DMVIC DEBUG] "+format, args...)
	}
}

// ensureValidToken checks if a valid token exists in storage and refreshes it if needed.
// This method ensures that API calls always have a valid authentication token.
func (c *client) ensureValidToken() error {
	/*	if c.token == "" || time.Now().After(c.expires.Add(-2*time.Minute)) {
			c.debugLog("Token expired or missing, refreshing...")
			return c.Login()
		}
		return nil
	*/

	_, found := c.tknStorage.Get("dmvictoken")
	if !found {
		c.debugLog("Token not found or empty, refreshing...")
		err := c.Login()
		if err != nil {
			return err
		}
	} else {
		//c.token = value
		c.debugLog("Using cached token")
	}

	return nil
}

// parseDMVICError converts DMVIC API error messages to standardized error codes.
// It maps common error messages to predefined error constants for better error handling.
func (c *client) parseDMVICError(errorMsg string) string {
	switch {
	case errorMsg == "Input json format is Incorrect":
		return DMVICErrInvalidJSON
	case errorMsg == "Unknown Error":
		return DMVICErrUnknownError
	case errorMsg == "Mandatory field is missing":
		return DMVICErrMandatoryField
	case errorMsg == "Input not valid":
		return DMVICErrInvalidInput
	case errorMsg == "Double Insurance":
		return DMVICErrDoubleInsurance
	case errorMsg == "No sufficient Inventory":
		return DMVICErrInsufficientStock
	case errorMsg == "Data Validation Error":
		return DMVICErrDataValidation
	default:
		if len(errorMsg) >= 5 && errorMsg[:2] == "ER" {
			return errorMsg[:5]
		}
		return ""
	}
}

// makeAPICall is a generic method for making authenticated API calls to DMVIC.
// It handles token validation, request marshaling, response handling, and error parsing.
// Parameters:
//   - method: HTTP method (GET, POST, etc.)
//   - endpoint: API endpoint path
//   - request: Request payload to be JSON marshaled
//   - response: Response struct to unmarshal the result into
//   - errorCode: Base error code for this operation
func (c *client) makeAPICall(method, endpoint string, request interface{}, response interface{}, errorCode int) error {
	var body []byte
	var err error
	if request != nil {
		body, err = json.Marshal(request)
		if err != nil {
			return newInternalError("makeAPICall", errorCode+2, err)
		}
		c.debugLog("Request body: %s", string(body))
	}
	url := c.endpoint + endpoint
	c.debugLog("Making %s request to: %s", method, url)

	attempts := 0
	for attempts < 2 {
		client, req, err := c.secureRequest(method, url, body)
		if err != nil {
			return newInternalError("makeAPICall", ErrCreateRequest, err)
		}

		resp, err := client.Do(req)
		if err != nil {
			return newExternalError("makeAPICall", errorCode+3, err.Error())
		}
		respBody, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return newInternalError("makeAPICall", ErrReadResponse, readErr)
		}
		c.debugLog("Response status: %d, body: %s", resp.StatusCode, string(respBody))

		if resp.StatusCode != http.StatusOK {
			clientErr := newExternalError("makeAPICall", errorCode+1, fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody)))
			clientErr.HTTPStatus = resp.StatusCode
			return clientErr
		}

		if err := json.Unmarshal(respBody, response); err != nil {
			return newInternalError("makeAPICall", ErrUnmarshalResponse, err)
		}

		// Detect DMVIC error from typed response (many response types implement GetError)
		var dmvicErrCode, dmvicErrText string
		if apiResp, ok := response.(interface{ GetError() string }); ok {
			dmvicErrText = apiResp.GetError()
			dmvicErrCode = c.parseDMVICError(dmvicErrText)
			// If GetError returned an error code string (like "ER001"), parseDMVICError may return it or empty.
			// If empty and the returned text looks like an ER code, use directly.
			if dmvicErrCode == "" && len(dmvicErrText) >= 5 && strings.HasPrefix(dmvicErrText, "ER") {
				dmvicErrCode = dmvicErrText[:5]
			}
		}

		// Fallback: inspect raw response body for Error array/object
		if dmvicErrCode == "" {
			var respMap map[string]interface{}
			if json.Unmarshal(respBody, &respMap) == nil {
				if e, exists := respMap["Error"]; exists {
					switch v := e.(type) {
					case []interface{}:
						if len(v) > 0 {
							if emap, ok := v[0].(map[string]interface{}); ok {
								if code, ok2 := emap["errorCode"].(string); ok2 {
									dmvicErrCode = code
								}
								if text, ok2 := emap["errorText"].(string); ok2 && dmvicErrText == "" {
									dmvicErrText = text
								}
							}
						}
					case map[string]interface{}:
						if code, ok2 := v["errorCode"].(string); ok2 {
							dmvicErrCode = code
						}
						if text, ok2 := v["errorText"].(string); ok2 && dmvicErrText == "" {
							dmvicErrText = text
						}
					}
				}
			}
		}

		// If token expired/invalid detected, refresh and retry once
		if dmvicErrCode == "ER001" || strings.Contains(strings.ToLower(dmvicErrText), "token is expired") || strings.Contains(strings.ToLower(dmvicErrText), "token is invalid") {
			if attempts == 0 {
				c.debugLog("DMVIC token error detected (%s / %s). Refreshing token and retrying...", dmvicErrCode, dmvicErrText)
				if err := c.Login(); err != nil {
					return err
				}
				attempts++
				continue
			}
		}

		// If there's a DMVIC error, return a DMVICError
		// For now let's skip this
		if (dmvicErrText != "" || dmvicErrCode != "") && false {
			codeToReturn := dmvicErrCode
			if codeToReturn == "" {
				codeToReturn = c.parseDMVICError(dmvicErrText)
			}
			return newDMVICError("makeAPICall", errorCode+4, codeToReturn, dmvicErrText)
		}

		// success path
		return nil
	}

	return newExternalError("makeAPICall", errorCode+5, "max retry attempts reached")
}

// === API Methods Implementation ===
// helper to calculate the number of days to expiry from a date string
// Returns the duration until expiry
func (c *client) getDurationToExpiry(dateStr string) (time.Duration, error) {
	expiryDate, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return 0, fmt.Errorf("error parsing date: %v", err)
	}
	currentDate := time.Now()
	duration := expiryDate.Sub(currentDate)
	if duration <= 0 {
		return 0, fmt.Errorf("token already expired")
	}
	return duration, nil
}

// Login authenticates with the DMVIC API and obtains an access token
func (c *client) Login() error {
	c.debugLog("Attempting login...")
	jsonData, err := json.Marshal(c.config.Credentials)
	if err != nil {
		return newInternalError("Login", ErrMarshalRequest, err)
	}
	loginURL := c.endpoint + "/V1/Account/Login"
	req, err := http.NewRequestWithContext(c.config.Context, http.MethodPost, loginURL, bytes.NewReader(jsonData))
	if err != nil {
		return newInternalError("Login", ErrCreateRequest, err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return newExternalError("Login", ErrHTTPRequest, err.Error())
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return newInternalError("Login", ErrReadResponse, err)
	}
	c.debugLog("Login response status: %d, body: %s", resp.StatusCode, string(body))
	if resp.StatusCode != http.StatusOK {
		return newExternalError("Login", ErrLoginFailed, fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)))
	}
	var loginResp LoginResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return newInternalError("Login", ErrUnmarshalResponse, err)
	}
	if loginResp.Code < 0 {
		var errorMsg string
		switch loginResp.Code {
		case -2:
			errorMsg = "Password is not set. Please activate your account"
		case -3:
			errorMsg = "Username or password is incorrect"
		case -4:
			errorMsg = "Your account is locked by admin"
		case -5:
			errorMsg = "Your account is blocked"
		case -6:
			errorMsg = "Username doesn't exist. Please enter correct username"
		case -7:
			errorMsg = "Your entity is suspended"
		case -8:
			errorMsg = "Your entity is deactivated"
		default:
			errorMsg = fmt.Sprintf("Login failed with code: %d", loginResp.Code)
		}
		return newExternalError("Login", ErrInvalidCredentials, errorMsg)
	}
	//expires, err := time.Parse(time.RFC3339, loginResp.Expires)
	if err != nil {
		return newInternalError("Login", ErrParseTime, err)
	}
	duration, err := c.getDurationToExpiry(loginResp.Expires)
	if err != nil {
		return newInternalError("Login", ErrParseTime, fmt.Errorf("error calculating days to expiry: %w", err))
	}
	c.tknStorage.Set("dmvictoken", loginResp.Token, duration)
	//c.token = loginResp.Token
	//c.expires = expires
	c.debugLog("Login successful, token expires in : %v ", duration)
	return nil
}

// GetToken returns the current authentication token
func (c *client) GetToken() string {
	tkn, found := c.tknStorage.Get("dmvictoken")
	if !found {
		c.debugLog("Error getting token from storage: ")
		return ""
	}
	return tkn
}

// IsTokenValid checks if the current token is valid and not expired
func (c *client) IsTokenValid() bool {
	_, found := c.tknStorage.Get("dmvictoken")
	return found
}

// Add GetError methods to response types for better error handling
func (r *CertificateResponse) GetError() string {
	if len(r.Error) > 0 {
		if r.Error[0].ErrorText != "" {
			return r.Error[0].ErrorText
		}
		if r.Error[0].ErrorCode != "" {
			return r.Error[0].ErrorCode
		}
	}
	return ""
}
func (r *InsuranceValidationResponse) GetError() string {
	if len(r.Error) > 0 {
		if r.Error[0].ErrorText != "" {
			return r.Error[0].ErrorText
		}
		if r.Error[0].ErrorCode != "" {
			return r.Error[0].ErrorCode
		}
	}
	return ""
}
func (r *CancellationResponse) GetError() string {
	if len(r.Error) > 0 {
		if r.Error[0].ErrorText != "" {
			return r.Error[0].ErrorText
		}
		if r.Error[0].ErrorCode != "" {
			return r.Error[0].ErrorCode
		}
	}
	return ""
}
func (r *DoubleInsuranceResponse) GetError() string {
	if len(r.Error) > 0 {
		if r.Error[0].ErrorText != "" {
			return r.Error[0].ErrorText
		}
		if r.Error[0].ErrorCode != "" {
			return r.Error[0].ErrorCode
		}
	}
	return ""
}
func (r *InsuranceResponse) GetError() string {
	if len(r.Error) > 0 {
		if r.Error[0].ErrorText != "" {
			return r.Error[0].ErrorText
		}
		if r.Error[0].ErrorCode != "" {
			return r.Error[0].ErrorCode
		}
	}
	return ""
}
func (r *StockResponse) GetError() string {
	if len(r.Error) > 0 {
		if r.Error[0].ErrorText != "" {
			return r.Error[0].ErrorText
		}
		if r.Error[0].ErrorCode != "" {
			return r.Error[0].ErrorCode
		}
	}
	return ""
}

func (c *client) GetCertificate(certificateNumber string) (*CertificateResponse, error) {
	req := &CertificateRequest{CertificateNumber: certificateNumber}
	var resp CertificateResponse
	err := c.makeAPICall(http.MethodPost, "/V4/Integration/GetCertificate", req, &resp, ErrGetCertificate)
	if err != nil {
		return nil, err
	}
	if !resp.Success && len(resp.Error) > 0 {
		dmvicCode := c.parseDMVICError(resp.Error[0].ErrorCode)
		return &resp, newDMVICError("GetCertificate", ErrGetCertificate, dmvicCode, resp.Error[0].ErrorText)
	}
	return &resp, nil
}

func (c *client) ValidateInsurance(req *InsuranceValidationRequest) (*InsuranceValidationResponse, error) {
	var resp InsuranceValidationResponse
	err := c.makeAPICall(http.MethodPost, "/V4/Integration/ValidateInsurance", req, &resp, ErrValidateInsurance)
	if err != nil {
		return nil, err
	}
	if !resp.Success && len(resp.Error) > 0 {
		dmvicCode := c.parseDMVICError(resp.Error[0].ErrorCode)
		return &resp, newDMVICError("ValidateInsurance", ErrValidateInsurance, dmvicCode, resp.Error[0].ErrorText)
	}
	return &resp, nil
}

func (c *client) CancelCertificate(certificateNumber string, reasonID int) (*CancellationResponse, error) {
	req := &CancellationRequest{
		CertificateNumber: certificateNumber,
		CancelReasonID:    reasonID,
	}
	var resp CancellationResponse
	err := c.makeAPICall(http.MethodPost, "/V4/Integration/CancelCertificate", req, &resp, ErrCancelCertificate)
	if err != nil {
		return nil, err
	}
	if !resp.Success && len(resp.Error) > 0 {
		dmvicCode := c.parseDMVICError(resp.Error[0].ErrorCode)
		return &resp, newDMVICError("CancelCertificate", ErrCancelCertificate, dmvicCode, resp.Error[0].ErrorText)
	}
	return &resp, nil
}

func (c *client) ValidateDoubleInsurance(req *DoubleInsuranceRequest) (*DoubleInsuranceResponse, error) {
	var resp DoubleInsuranceResponse
	err := c.makeAPICall(http.MethodPost, "/V4/Integration/ValidateDoubleInsurance", req, &resp, ErrValidateDoubleInsurance)
	if err != nil {
		return nil, err
	}
	if !resp.Success && len(resp.Error) > 0 {
		dmvicCode := c.parseDMVICError(resp.Error[0].ErrorCode)
		return &resp, newDMVICError("ValidateDoubleInsurance", ErrValidateDoubleInsurance, dmvicCode, resp.Error[0].ErrorText)
	}
	return &resp, nil
}

func (c *client) IssueTypeACertificate(req *TypeAIssuanceRequest) (*InsuranceResponse, error) {

	var resp InsuranceResponse
	err := c.makeAPICall(http.MethodPost, "/V4/IntermediaryIntegration/IssuanceTypeACertificate", req, &resp, ErrIssuanceTypeA)
	if err != nil {
		return nil, err
	}
	if !resp.Success && len(resp.Error) > 0 {
		dmvicCode := c.parseDMVICError(resp.Error[0].ErrorCode)
		clientErr := newDMVICError("IssueTypeACertificate", ErrIssuanceTypeA, dmvicCode, resp.Error[0].ErrorText)
		return &resp, clientErr
	}
	return &resp, nil
}

func (c *client) IssueTypeBCertificate(req *TypeBIssuanceRequest) (*InsuranceResponse, error) {
	var resp InsuranceResponse
	err := c.makeAPICall(http.MethodPost, "/V4/IntermediaryIntegration/IssuanceTypeBCertificate", req, &resp, ErrIssuanceTypeB)
	if err != nil {
		return nil, err
	}
	if !resp.Success && len(resp.Error) > 0 {
		dmvicCode := c.parseDMVICError(resp.Error[0].ErrorCode)
		clientErr := newDMVICError("IssueTypeBCertificate", ErrIssuanceTypeB, dmvicCode, resp.Error[0].ErrorText)
		return &resp, clientErr
	}
	return &resp, nil
}

func (c *client) IssueTypeCCertificate(req *TypeCIssuanceRequest) (*InsuranceResponse, error) {
	var resp InsuranceResponse
	err := c.makeAPICall(http.MethodPost, "/V4/IntermediaryIntegration/IssuanceTypeCCertificate", req, &resp, ErrIssuanceTypeC)
	if err != nil {
		return nil, err
	}
	if !resp.Success && len(resp.Error) > 0 {
		dmvicCode := c.parseDMVICError(resp.Error[0].ErrorCode)
		clientErr := newDMVICError("IssueTypeCCertificate", ErrIssuanceTypeC, dmvicCode, resp.Error[0].ErrorText)
		return &resp, clientErr
	}
	return &resp, nil
}

func (c *client) IssueTypeDCertificate(req *TypeDIssuanceRequest) (*InsuranceResponse, error) {
	var resp InsuranceResponse
	err := c.makeAPICall(http.MethodPost, "/V4/IntermediaryIntegration/IssuanceTypeDCertificate", req, &resp, ErrIssuanceTypeD)
	if err != nil {
		return nil, err
	}
	if !resp.Success && len(resp.Error) > 0 {
		dmvicCode := c.parseDMVICError(resp.Error[0].ErrorCode)
		clientErr := newDMVICError("IssueTypeDCertificate", ErrIssuanceTypeD, dmvicCode, resp.Error[0].ErrorText)
		return &resp, clientErr
	}
	return &resp, nil
}

func (c *client) GetMemberCompanyStock(memberCompanyID int) (*StockResponse, error) {
	var resp StockResponse
	// DMVIC 4.8.1: POST /v6/Integration/MemberCompanyStock with the member
	// company id in the body (params are case-insensitive per the spec).
	reqBody := map[string]interface{}{"MemberCompanyId": memberCompanyID}
	err := c.makeAPICall(http.MethodPost, "/v6/Integration/MemberCompanyStock", reqBody, &resp, ErrMemberCompanyStock)
	if err != nil {
		return nil, err
	}
	if !resp.Success && len(resp.Error) > 0 {
		dmvicCode := c.parseDMVICError(resp.Error[0].ErrorCode)
		return &resp, newDMVICError("GetMemberCompanyStock", ErrMemberCompanyStock, dmvicCode, resp.Error[0].ErrorText)
	}

	return &resp, nil
}

func (c *client) GetMemberIntermediaryStock(memberIntermediaryID int) (*StockResponse, error) {
	var resp StockResponse
	endpoint := fmt.Sprintf("/V6/Integration/IntermediaryStock?MemberIntermediaryId=%d", memberIntermediaryID)
	err := c.makeAPICall(http.MethodGet, endpoint, nil, &resp, ErrIntermediaryStock)
	if err != nil {
		return nil, err
	}
	if !resp.Success && len(resp.Error) > 0 {
		dmvicCode := c.parseDMVICError(resp.Error[0].ErrorCode)
		return &resp, newDMVICError("GetMemberIntermediaryStock", ErrIntermediaryStock, dmvicCode, resp.Error[0].ErrorText)
	}
	return &resp, nil
}

// GetIntermediaryStock retrieves certificate stock for the given intermediary
// IRA numbers. DMVIC 4.8.2: POST /v4/Integration/IntermediaryStock with the IRA
// numbers in the body (params are case-insensitive per the spec). This endpoint
// is at v4 (unlike MemberCompanyStock at v6) — v6 returns "invalid API".
func (c *client) GetIntermediaryStock(iraNumbers []string) (*StockResponse, error) {
	var resp StockResponse
	reqBody := map[string]interface{}{"IntermediaryIRANumbers": iraNumbers}
	err := c.makeAPICall(http.MethodPost, "/v6/Integration/IntermediaryStock", reqBody, &resp, ErrIntermediaryStock)
	if err != nil {
		return nil, err
	}
	if !resp.Success && len(resp.Error) > 0 {
		dmvicCode := c.parseDMVICError(resp.Error[0].ErrorCode)
		return &resp, newDMVICError("GetIntermediaryStock", ErrIntermediaryStock, dmvicCode, resp.Error[0].ErrorText)
	}
	return &resp, nil
}

func (c *client) ConfirmCertificateIssuance(req *ConfirmationRequest) (*InsuranceResponse, error) {
	var resp InsuranceResponse
	err := c.makeAPICall(http.MethodPost, "/V4/IntermediaryIntegration/ConfirmCertificateIssuance", req, &resp, ErrConfirmIssuance)
	if err != nil {
		return nil, err
	}
	if !resp.Success && len(resp.Error) > 0 {
		dmvicCode := c.parseDMVICError(resp.Error[0].ErrorCode)
		return &resp, newDMVICError("ConfirmCertificateIssuance", ErrConfirmIssuance, dmvicCode, resp.Error[0].ErrorText)
	}
	return &resp, nil
}

// secureRequest creates a mutual TLS HTTP client and request for DMVIC
func (c *client) secureRequest(method, url string, jsonPayload []byte) (*http.Client, *http.Request, error) {
	// Load client cert

	value, found := c.tknStorage.Get("dmvictoken")
	if !found {
		c.debugLog("Token not found or empty, refreshing...")
		err := c.Login()
		if err != nil {
			return nil, nil, err
		}
		value, _ = c.tknStorage.Get("dmvictoken")
	} else {
		//c.token = value
		c.debugLog("Using cached token")
	}

	cert, err := tls.LoadX509KeyPair(c.config.AuthCertPath, c.config.AuthKeyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load cert/key: %w", err)
	}

	// Optionally load CA cert if the server uses a custom CA
	caCert, err := ioutil.ReadFile(c.config.AuthCaCertPath) // optional
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load CA cert: %w", err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Set up HTTPS client with mutual TLS
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		// RootCAs:      caCertPool, // optional, uncomment if needed
	}
	// Deprecated in Go 1.15+, but harmless for compatibility
	tlsConfig.BuildNameToCertificate()

	transport := &http.Transport{TLSClientConfig: tlsConfig}
	client := &http.Client{Transport: transport}

	// Build request
	req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", value))
	req.Header.Set("ClientID", c.config.ClientID)

	return client, req, nil
}

// secureRequest creates a mutual TLS HTTP client and request for DMVIC
func (c *client) normalRequest(method, url string, jsonPayload []byte) (*http.Client, *http.Request, error) {
	value, found := c.tknStorage.Get("dmvictoken")
	if !found {
		c.debugLog("Token not found or empty, refreshing...")
		err := c.Login()
		if err != nil {
			return nil, nil, err
		}
	} else {
		//c.token = value
		c.debugLog("Using cached token")
	}

	// Create a standard HTTP client
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: c.config.InsecureSkipVerify,
		},
	}
	client := &http.Client{
		Timeout:   c.config.Timeout,
		Transport: transport,
	}
	// Build request

	req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}
	c.debugLog(c.config.ClientID)

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", value))
	req.Header.Set("ClientID", c.config.ClientID)
	return client, req, nil
}
