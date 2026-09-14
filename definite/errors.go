package definite

import (
	"errors"
	"fmt"
)

// ErrorType categorises where an error originated.
type ErrorType string

const (
	// InternalError covers client-side failures: configuration, input, transport, parsing.
	InternalError ErrorType = "InternalError"
	// ExternalError covers failures reported by the Definite API itself.
	ExternalError ErrorType = "ExternalError"
)

// Error codes, grouped by category. They match the TypeScript SDK's ERROR_CODES,
// with one addition — ErrInvalidInput — noted below.
const (
	// Configuration and client-side errors (1000-1099)
	ErrInvalidConfig     = 1001 // Invalid client configuration
	ErrMarshalRequest    = 1002 // Failed to marshal request to JSON
	ErrCreateRequest     = 1003 // Failed to create the HTTP request
	ErrHTTPRequest       = 1004 // HTTP request execution failed
	ErrReadResponse      = 1005 // Failed to read the response body
	ErrParseTime         = 1006 // Failed to parse or format a timestamp
	ErrUnmarshalResponse = 1007 // Response JSON did not match the expected shape
	// ErrInvalidInput marks a request rejected by local validation before it was
	// sent. The TypeScript SDK reports these under ErrInvalidConfig (1001); this
	// SDK gives them their own code so callers can tell the two apart.
	ErrInvalidInput = 1008

	// Authentication errors (2000-2099)
	ErrUnauthorized       = 2001
	ErrInvalidCredentials = 2002
	ErrForbidden          = 2003

	// API operation errors (3000-8999) — one per endpoint
	ErrCheckClient          = 3000
	ErrCreateClient         = 3100
	ErrProducts             = 4000
	ErrMakes                = 4100
	ErrCalculatePremium     = 5000
	ErrCheckIntermediary    = 6000
	ErrRegisterIntermediary = 6100
	ErrCheckPolicy          = 7000
	ErrNewProposal          = 8000
	ErrInitiatePayment      = 8100
	ErrGenerateCertificate  = 8200
	ErrCheckWalletBalance   = 8300
	ErrTopUpWallet          = 8400

	// Transport and parsing errors (9000-9099)
	ErrEmptyResponse   = 9000
	ErrNonJSONResponse = 9001
	ErrAPI             = 9002
	ErrNetwork         = 9003
	ErrTimeout         = 9004
	ErrNotFound        = 9007
	ErrUnknown         = 9008
)

// ClientError is the error type returned by every Client method.
//
// Code follows the table above, except for HTTP-level failures: a 401 or 403
// carries Code 401, and any other non-2xx response carries its HTTP status as
// the Code — matching the TypeScript SDK.
type ClientError struct {
	Type         ErrorType `json:"type"`                    // Internal or External
	Code         int       `json:"code"`                    // See the error code constants
	Message      string    `json:"message"`                 // Human-readable description
	Operation    string    `json:"operation,omitempty"`     // Client method that failed, e.g. "CreateProposal"
	DefiniteCode string    `json:"definite_code,omitempty"` // errorCode from the API envelope, when present
	HTTPStatus   int       `json:"http_status,omitempty"`   // HTTP status, when a response was received
	Response     []byte    `json:"-"`                       // Raw response body, when one was received
	Err          error     `json:"-"`                       // Underlying cause, if any
}

// Error implements the error interface.
func (e *ClientError) Error() string {
	switch {
	case e.Operation != "" && e.DefiniteCode != "":
		return fmt.Sprintf("definite %s error %d (%s): %s", e.Operation, e.Code, e.DefiniteCode, e.Message)
	case e.Operation != "":
		return fmt.Sprintf("definite %s error %d: %s", e.Operation, e.Code, e.Message)
	default:
		return fmt.Sprintf("definite error %d: %s", e.Code, e.Message)
	}
}

// Unwrap exposes the underlying cause to errors.Is and errors.As.
func (e *ClientError) Unwrap() error { return e.Err }

// IsAuthentication reports whether the API rejected the Basic credentials.
func (e *ClientError) IsAuthentication() bool {
	return e.HTTPStatus == 401 || e.HTTPStatus == 403
}

// IsTimeout reports whether the request exceeded its deadline.
func (e *ClientError) IsTimeout() bool { return e.Code == ErrTimeout }

// IsNetwork reports whether the request failed at the transport level.
func (e *ClientError) IsNetwork() bool { return e.Code == ErrNetwork }

// IsInvalidInput reports whether local validation rejected the request before it was sent.
func (e *ClientError) IsInvalidInput() bool { return e.Code == ErrInvalidInput }

// AsClientError unwraps err to a *ClientError when it is one.
func AsClientError(err error) (*ClientError, bool) {
	var ce *ClientError
	if errors.As(err, &ce) {
		return ce, true
	}
	return nil, false
}

// IsAuthError reports whether err is an authentication failure (HTTP 401 or 403).
func IsAuthError(err error) bool {
	ce, ok := AsClientError(err)
	return ok && ce.IsAuthentication()
}

// IsTimeoutError reports whether err is a request timeout.
func IsTimeoutError(err error) bool {
	ce, ok := AsClientError(err)
	return ok && ce.IsTimeout()
}

// IsInvalidInputError reports whether err is a local validation failure.
func IsInvalidInputError(err error) bool {
	ce, ok := AsClientError(err)
	return ok && ce.IsInvalidInput()
}

// newInternalError wraps a client-side failure.
func newInternalError(op string, code int, err error) *ClientError {
	return &ClientError{Type: InternalError, Code: code, Message: err.Error(), Operation: op, Err: err}
}

// newInputError reports a request rejected by local validation.
func newInputError(op string, err error) *ClientError {
	return &ClientError{Type: InternalError, Code: ErrInvalidInput, Message: err.Error(), Operation: op, Err: err}
}

// newAPIError reports a failure the API signalled in a successful HTTP response.
func newAPIError(op string, code int, message, definiteCode string, body []byte) *ClientError {
	return &ClientError{
		Type:         ExternalError,
		Code:         code,
		Message:      message,
		Operation:    op,
		DefiniteCode: definiteCode,
		HTTPStatus:   200,
		Response:     body,
	}
}

// newHTTPError reports a non-2xx HTTP response.
func newHTTPError(op string, status int, body []byte) *ClientError {
	if status == 401 || status == 403 {
		return &ClientError{
			Type:       ExternalError,
			Code:       401,
			Message:    fmt.Sprintf("authentication failed (HTTP %d); check your Definite Basic auth credentials", status),
			Operation:  op,
			HTTPStatus: status,
			Response:   body,
		}
	}
	return &ClientError{
		Type:       ExternalError,
		Code:       status,
		Message:    fmt.Sprintf("HTTP error: status %d, response: %s", status, truncate(body, 512)),
		Operation:  op,
		HTTPStatus: status,
		Response:   body,
	}
}

// truncate shortens a body for inclusion in an error message.
func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "…"
}
