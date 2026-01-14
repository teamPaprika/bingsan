package updates

import "fmt"

// UpdateError represents a failure during update processing.
type UpdateError struct {
	Action  string // The action that failed
	Message string // Human-readable error message
	Cause   error  // Underlying error, if any
}

// Error implements the error interface.
func (e *UpdateError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Action, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Action, e.Message)
}

// Unwrap returns the underlying error for errors.Is/As support.
func (e *UpdateError) Unwrap() error {
	return e.Cause
}

// NewUpdateError creates a new UpdateError.
func NewUpdateError(action, message string) *UpdateError {
	return &UpdateError{
		Action:  action,
		Message: message,
	}
}

// WrapUpdateError creates a new UpdateError with an underlying cause.
func WrapUpdateError(action, message string, cause error) *UpdateError {
	return &UpdateError{
		Action:  action,
		Message: message,
		Cause:   cause,
	}
}
