package hyperliquid

import "fmt"

// APIError is returned for server-side error responses from /info and
// /exchange.
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
	Data    any    `json:"data,omitempty"`
}

// Error renders the API error as a string.
func (e APIError) Error() string {
	return fmt.Sprintf("API error %d: %s", e.Code, e.Message)
}

// ValidationError is returned by validate() when a placement spec fails a
// pre-flight check. Callers can branch on Code via errors.As.
type ValidationError struct {
	Field   string
	Code    string
	Message string
	Got     any
	Want    any
}

// Error renders the validation failure as a string. If a Message is set, it
// is returned verbatim; otherwise the Code is used.
func (e *ValidationError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("validation error: field=%s code=%s", e.Field, e.Code)
}
