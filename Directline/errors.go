package directline

import "fmt"

// ErrorKind categorizes errors produced by the Directline client
type ErrorKind string

const (
	Internal ErrorKind = "Internal"
	External ErrorKind = "External"
	Auth     ErrorKind = "Auth"
)

const (
	ErrInvalidConfig     = 1001
	ErrMarshalRequest    = 1002
	ErrCreateRequest     = 1003
	ErrHTTPRequest       = 1004
	ErrReadResponse      = 1005
	ErrUnmarshalResponse = 1006
	// ErrInvalidInput marks a request rejected by local validation, before
	// anything was sent. Matches the TypeScript SDK's INVALID_INPUT.
	ErrInvalidInput = 1007
	ErrLoginFailed  = 2001
)

// APIError wraps response errors from Directline
type APIError struct {
	Kind      ErrorKind
	Code      int
	Message   string
	Operation string
	HTTPCode  int
	Debug     bool
}

func (c *APIError) debugLog(format string, args ...any) {
	if c.Debug {
		fmt.Printf("[Directline DEBUG] "+format+"\n", args...)
	}
}

func (e *APIError) Error() string {
	if e.Operation != "" {
		return fmt.Sprintf("directline %s error %d: %s", e.Operation, e.Code, e.Message)
	}
	e.debugLog("directline error %d: %s", e.Code, e.Message)
	return fmt.Sprintf("directline error %d: %s", e.Code, e.Message)
}

func newInternalError(op string, code int, err error, debug bool) *APIError {
	return &APIError{Kind: Internal, Code: code, Message: err.Error(), Operation: op, Debug: debug}
}

func newExternalError(op string, code int, message string, httpCode int, debug bool) *APIError {
	return &APIError{Kind: External, Code: code, Message: message, Operation: op, HTTPCode: httpCode, Debug: debug}
}

func newAuthError(op string, code int, message string, debug bool) *APIError {
	return &APIError{Kind: Auth, Code: code, Message: message, Operation: op, Debug: debug}
}
