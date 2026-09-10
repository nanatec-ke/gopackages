package dmvic

import "fmt"

// Package dmvic provides error types, error codes, and error helpers for DMVIC client operations.

// ErrorType categorizes different kinds of errors that can occur during DMVIC operations.
type ErrorType string

const (
	// InternalError represents client-side errors such as configuration issues or marshaling problems
	InternalError ErrorType = "InternalError"
	// ExternalError represents server-side or network errors from the DMVIC API
	ExternalError ErrorType = "ExternalError"
)

// Predefined error codes for specific error conditions.
// Error codes are organized by category for easy identification and handling.
const (
	// Configuration errors (1000-1099)
	ErrInvalidConfig     = 1001 // Invalid client configuration
	ErrMarshalRequest    = 1002 // Failed to marshal request to JSON
	ErrCreateRequest     = 1003 // Failed to create HTTP request
	ErrHTTPRequest       = 1004 // HTTP request execution failed
	ErrReadResponse      = 1005 // Failed to read HTTP response body
	ErrParseTime         = 1006 // Failed to parse time/date string
	ErrUnmarshalResponse = 1007 // Failed to unmarshal JSON response

	// Authentication errors (2000-2099)
	ErrLoginFailed        = 2001 // Login operation failed
	ErrTokenExpired       = 2002 // Authentication token has expired
	ErrUnauthorized       = 2003 // Unauthorized access attempt
	ErrInvalidCredentials = 2004 // Invalid username or password
	ErrTokenRefresh       = 2005 // Token refresh operation failed

	// API operation errors (3000-8999)
	ErrGetCertificate          = 3000 // Certificate retrieval operation failed
	ErrValidateInsurance       = 4000 // Insurance validation operation failed
	ErrCancelCertificate       = 5000 // Certificate cancellation operation failed
	ErrMemberCompanyStock      = 6000 // Member company stock retrieval failed
	ErrIntermediaryStock       = 6500 //Intermediary company stock retrieval failed
	ErrIssuanceTypeA           = 7000 // Type A certificate issuance failed
	ErrIssuanceTypeB           = 7100 // Type B certificate issuance failed
	ErrIssuanceTypeC           = 7200 // Type C certificate issuance failed
	ErrIssuanceTypeD           = 7300 // Type D certificate issuance failed
	ErrConfirmIssuance         = 7400 // Certificate issuance confirmation failed
	ErrValidateDoubleInsurance = 8000 // Double insurance validation failed
)

// API-specific error codes from DMVIC responses.
// These codes are returned by the DMVIC API to indicate specific error conditions.
const (
	DMVICErrInvalidJSON       = "ER001" // Input json format is Incorrect
	DMVICErrUnknownError      = "ER002" // Unknown Error
	DMVICErrMandatoryField    = "ER003" // Mandatory field is missing
	DMVICErrInvalidInput      = "ER004" // Input not valid
	DMVICErrDoubleInsurance   = "ER005" // Double Insurance
	DMVICErrInsufficientStock = "ER006" // No sufficient Inventory
	DMVICErrDataValidation    = "ER007" // Data Validation Error
)

// ClientError represents an error that occurred during DMVIC operations.
// It provides detailed information about the error including type, code, message, and context.
type ClientError struct {
	Type       ErrorType `json:"type"`                  // Type of error (Internal or External)
	Code       int       `json:"code"`                  // Numeric error code
	Message    string    `json:"message"`               // Human-readable error message
	Operation  string    `json:"operation,omitempty"`   // Operation that caused the error
	DMVICCode  string    `json:"dmvic_code,omitempty"`  // DMVIC-specific error code
	HTTPStatus int       `json:"http_status,omitempty"` // HTTP status code if applicable
}

// Error returns a formatted string representation of the ClientError.
// It implements the error interface and provides context-aware error messages.
func (e *ClientError) Error() string {
	if e.Operation != "" {
		if e.DMVICCode != "" {
			return fmt.Sprintf("dmvic %s error %d (%s): %s", e.Operation, e.Code, e.DMVICCode, e.Message)
		}
		return fmt.Sprintf("dmvic %s error %d: %s", e.Operation, e.Code, e.Message)
	}
	return fmt.Sprintf("dmvic error %d: %s", e.Code, e.Message)
}

// IsInsufficientInventory checks if the error is due to insufficient inventory/stock.
// Returns true if the DMVIC error code indicates insufficient stock.
func (e *ClientError) IsInsufficientInventory() bool {
	return e.DMVICCode == DMVICErrInsufficientStock
}

// IsDoubleInsurance checks if the error is due to double insurance detection.
// Returns true if the DMVIC error code indicates double insurance.
func (e *ClientError) IsDoubleInsurance() bool {
	return e.DMVICCode == DMVICErrDoubleInsurance
}

// IsDataValidationError checks if the error is due to data validation issues.
// Returns true if the DMVIC error code indicates a data validation error.
func (e *ClientError) IsDataValidationError() bool {
	return e.DMVICCode == DMVICErrDataValidation
}

// Helper functions for creating different types of errors

// newInternalError creates a new ClientError for internal/client-side errors.
// These are typically configuration issues, marshaling problems, or other client-side failures.
// Parameters:
//   - op: The operation that caused the error
//   - code: The error code
//   - err: The underlying error
func newInternalError(op string, code int, err error) *ClientError {
	return &ClientError{
		Type:      InternalError,
		Code:      code,
		Message:   err.Error(),
		Operation: op,
	}
}

// newExternalError creates a new ClientError for external/server-side errors.
// These are typically network errors, HTTP errors, or server-side failures.
// Parameters:
//   - op: The operation that caused the error
//   - code: The error code
//   - message: The error message
func newExternalError(op string, code int, message string) *ClientError {
	return &ClientError{
		Type:      ExternalError,
		Code:      code,
		Message:   message,
		Operation: op,
	}
}

// newDMVICError creates a new ClientError for DMVIC API-specific errors.
// These errors include the DMVIC error code returned by the API.
// Parameters:
//   - op: The operation that caused the error
//   - code: The client error code
//   - dmvicCode: The DMVIC-specific error code
//   - message: The error message from DMVIC
func newDMVICError(op string, code int, dmvicCode, message string) *ClientError {
	return &ClientError{
		Type:      ExternalError,
		Code:      code,
		Message:   message,
		Operation: op,
		DMVICCode: dmvicCode,
	}
}
