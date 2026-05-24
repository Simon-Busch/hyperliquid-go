package types

import "fmt"

// ValidationError is returned by trade pre-flight checks when a placement
// spec fails. Callers can branch on Code via errors.As.
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
