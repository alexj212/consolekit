package consolekit

import (
	"fmt"
	"time"
)

// CLIError is a structured error type for CLI operations
type CLIError struct {
	Command   string    // The command that failed
	Message   string    // Human-readable error message
	Cause     error     // Underlying error
	Timestamp time.Time // When the error occurred
}

// Error implements the error interface
func (e *CLIError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Command, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Command, e.Message)
}

// Unwrap returns the underlying error for error unwrapping
func (e *CLIError) Unwrap() error {
	return e.Cause
}

// NewCLIError creates a new CLIError
func NewCLIError(command string, message string, cause error) *CLIError {
	return &CLIError{
		Command:   command,
		Message:   message,
		Cause:     cause,
		Timestamp: time.Now(),
	}
}

// WrapError wraps an error in a CLIError
func WrapError(command string, err error) error {
	if err == nil {
		return nil
	}
	return NewCLIError(command, "command failed", err)
}
