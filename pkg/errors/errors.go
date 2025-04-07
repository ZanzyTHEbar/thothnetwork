package errors

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	errbuilder "github.com/ZanzyTHEbar/errbuilder-go"
)

// ErrorCategory categorizes errors for better handling
type ErrorCategory string

const (
	// CategoryValidation represents validation errors
	CategoryValidation ErrorCategory = "validation"
	// CategoryDatabase represents database errors
	CategoryDatabase ErrorCategory = "database"
	// CategoryNetwork represents network errors
	CategoryNetwork ErrorCategory = "network"
	// CategorySecurity represents security errors
	CategorySecurity ErrorCategory = "security"
	// CategoryInternal represents internal errors
	CategoryInternal ErrorCategory = "internal"
	// CategoryExternal represents errors from external services
	CategoryExternal ErrorCategory = "external"
	// CategoryTimeout represents timeout errors
	CategoryTimeout ErrorCategory = "timeout"
	// CategoryNotFound represents not found errors
	CategoryNotFound ErrorCategory = "not_found"
	// CategoryAlreadyExists represents already exists errors
	CategoryAlreadyExists ErrorCategory = "already_exists"
	// CategoryPermission represents permission errors
	CategoryPermission ErrorCategory = "permission"
	// CategoryResource represents resource errors (e.g., out of memory)
	CategoryResource ErrorCategory = "resource"
)

// ErrorSeverity represents the severity of an error
type ErrorSeverity string

const (
	// SeverityDebug represents debug-level errors
	SeverityDebug ErrorSeverity = "debug"
	// SeverityInfo represents informational errors
	SeverityInfo ErrorSeverity = "info"
	// SeverityWarning represents warning-level errors
	SeverityWarning ErrorSeverity = "warning"
	// SeverityError represents error-level errors
	SeverityError ErrorSeverity = "error"
	// SeverityCritical represents critical errors
	SeverityCritical ErrorSeverity = "critical"
	// SeverityFatal represents fatal errors
	SeverityFatal ErrorSeverity = "fatal"
)

// ErrorRetryStrategy represents the retry strategy for an error
type ErrorRetryStrategy string

const (
	// RetryNone means no retry
	RetryNone ErrorRetryStrategy = "none"
	// RetryImmediate means retry immediately
	RetryImmediate ErrorRetryStrategy = "immediate"
	// RetryBackoff means retry with exponential backoff
	RetryBackoff ErrorRetryStrategy = "backoff"
	// RetryFixed means retry with fixed delay
	RetryFixed ErrorRetryStrategy = "fixed"
)

// StackFrame represents a stack frame
type StackFrame struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Function string `json:"function"`
}

// Error represents a structured error
type Error struct {
	// Message is the error message
	Message string `json:"message"`

	// Code is the error code
	Code errbuilder.ErrCode `json:"code"`

	// Category is the error category
	Category ErrorCategory `json:"category"`

	// Severity is the error severity
	Severity ErrorSeverity `json:"severity"`

	// RetryStrategy is the retry strategy
	RetryStrategy ErrorRetryStrategy `json:"retry_strategy"`

	// RetryDelay is the delay between retries
	RetryDelay time.Duration `json:"retry_delay"`

	// MaxRetries is the maximum number of retries
	MaxRetries int `json:"max_retries"`

	// Metadata is additional information about the error
	Metadata map[string]any `json:"metadata"`

	// Cause is the underlying error
	Cause error `json:"cause"`

	// Stack is the stack trace
	Stack []StackFrame `json:"stack"`

	// Timestamp is the time the error occurred
	Timestamp time.Time `json:"timestamp"`
}

// Error returns the error message
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s", e.Message, e.Cause.Error())
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *Error) Unwrap() error {
	return e.Cause
}

// WithCause adds a cause to the error
func (e *Error) WithCause(cause error) *Error {
	e.Cause = cause
	return e
}

// WithMetadata adds metadata to the error
func (e *Error) WithMetadata(key string, value any) *Error {
	if e.Metadata == nil {
		e.Metadata = make(map[string]any)
	}
	e.Metadata[key] = value
	return e
}

// WithCategory sets the error category
func (e *Error) WithCategory(category ErrorCategory) *Error {
	e.Category = category
	return e
}

// WithSeverity sets the error severity
func (e *Error) WithSeverity(severity ErrorSeverity) *Error {
	e.Severity = severity
	return e
}

// WithRetry sets the retry strategy
func (e *Error) WithRetry(strategy ErrorRetryStrategy, delay time.Duration, maxRetries int) *Error {
	e.RetryStrategy = strategy
	e.RetryDelay = delay
	e.MaxRetries = maxRetries
	return e
}

// WithStack adds a stack trace to the error
func (e *Error) WithStack() *Error {
	e.Stack = captureStack(2) // Skip this function and the caller
	return e
}

// New creates a new error
func New(message string) *Error {
	return &Error{
		Message:       message,
		Code:          errbuilder.CodeInternal,
		Category:      CategoryInternal,
		Severity:      SeverityError,
		RetryStrategy: RetryNone,
		Timestamp:     time.Now(),
		Stack:         captureStack(2), // Skip this function and the caller
	}
}

// Wrap wraps an error with a message
func Wrap(err error, message string) *Error {
	if err == nil {
		return nil
	}

	// If the error is already a *Error, just update the message
	if e, ok := err.(*Error); ok {
		return &Error{
			Message:       message + ": " + e.Message,
			Code:          e.Code,
			Category:      e.Category,
			Severity:      e.Severity,
			RetryStrategy: e.RetryStrategy,
			RetryDelay:    e.RetryDelay,
			MaxRetries:    e.MaxRetries,
			Metadata:      e.Metadata,
			Cause:         e.Cause,
			Stack:         captureStack(2), // Skip this function and the caller
			Timestamp:     time.Now(),
		}
	}

	return &Error{
		Message:       message,
		Code:          errbuilder.CodeInternal,
		Category:      CategoryInternal,
		Severity:      SeverityError,
		RetryStrategy: RetryNone,
		Cause:         err,
		Stack:         captureStack(2), // Skip this function and the caller
		Timestamp:     time.Now(),
	}
}

// FromErrBuilder converts an errbuilder.ErrBuilder to an Error
func FromErrBuilder(err error) *Error {
	if err == nil {
		return nil
	}

	// If the error is already a *Error, just return it
	if e, ok := err.(*Error); ok {
		return e
	}

	// Try to convert from errbuilder.ErrBuilder
	if e, ok := err.(*errbuilder.ErrBuilder); ok {
		category := CategoryInternal
		severity := SeverityError

		// Map errbuilder codes to categories and severities
		switch e.Code {
		case errbuilder.CodeInvalidArgument:
			category = CategoryValidation
			severity = SeverityWarning
		case errbuilder.CodeNotFound:
			category = CategoryNotFound
			severity = SeverityWarning
		case errbuilder.CodeAlreadyExists:
			category = CategoryAlreadyExists
			severity = SeverityWarning
		case errbuilder.CodePermissionDenied:
			category = CategoryPermission
			severity = SeverityError
		case errbuilder.CodeResourceExhausted:
			category = CategoryResource
			severity = SeverityCritical
		case errbuilder.CodeFailedPrecondition:
			category = CategoryValidation
			severity = SeverityWarning
		case errbuilder.CodeAborted:
			category = CategoryInternal
			severity = SeverityError
		case errbuilder.CodeOutOfRange:
			category = CategoryValidation
			severity = SeverityWarning
		case errbuilder.CodeUnimplemented:
			category = CategoryInternal
			severity = SeverityError
		case errbuilder.CodeInternal:
			category = CategoryInternal
			severity = SeverityError
		case errbuilder.CodeUnavailable:
			category = CategoryExternal
			severity = SeverityCritical
		case errbuilder.CodeDataLoss:
			category = CategoryDatabase
			severity = SeverityCritical
		case errbuilder.CodeUnauthenticated:
			category = CategorySecurity
			severity = SeverityError
		}

		result := &Error{
			Message:       e.Msg,
			Code:          e.Code,
			Category:      category,
			Severity:      severity,
			RetryStrategy: RetryNone,
			Metadata:      make(map[string]any),
			Cause:         e.Cause,
			Stack:         captureStack(2), // Skip this function and the caller
			Timestamp:     time.Now(),
		}

		// Copy metadata if available
		if e.Details.Errors != nil {
			for k, v := range e.Details.Errors {
				result.Metadata[k] = v
			}
		}

		return result
	}

	// Just wrap the error
	return Wrap(err, "")
}

// captureStack captures the current stack trace
func captureStack(skip int) []StackFrame {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(skip, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])

	stack := make([]StackFrame, 0, n)
	for {
		frame, more := frames.Next()

		// Skip runtime and standard library frames
		if strings.Contains(frame.File, "runtime/") {
			if more {
				continue
			}
			break
		}

		stack = append(stack, StackFrame{
			File:     frame.File,
			Line:     frame.Line,
			Function: frame.Function,
		})

		if !more {
			break
		}
	}

	return stack
}

// IsNotFound returns true if the error is a not found error
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}

	// Check if it's our Error type
	if e, ok := err.(*Error); ok {
		return e.Category == CategoryNotFound || e.Code == errbuilder.CodeNotFound
	}

	// Check if it's an errbuilder.ErrBuilder
	if e, ok := err.(*errbuilder.ErrBuilder); ok {
		return e.Code == errbuilder.CodeNotFound
	}

	// Check the error string as a last resort
	return strings.Contains(strings.ToLower(err.Error()), "not found")
}

// IsAlreadyExists returns true if the error is an already exists error
func IsAlreadyExists(err error) bool {
	if err == nil {
		return false
	}

	// Check if it's our Error type
	if e, ok := err.(*Error); ok {
		return e.Category == CategoryAlreadyExists || e.Code == errbuilder.CodeAlreadyExists
	}

	// Check if it's an errbuilder.ErrBuilder
	if e, ok := err.(*errbuilder.ErrBuilder); ok {
		return e.Code == errbuilder.CodeAlreadyExists
	}

	// Check the error string as a last resort
	return strings.Contains(strings.ToLower(err.Error()), "already exists")
}

// IsTimeout returns true if the error is a timeout error
func IsTimeout(err error) bool {
	if err == nil {
		return false
	}

	// Check if it's our Error type
	if e, ok := err.(*Error); ok {
		return e.Category == CategoryTimeout
	}

	// Check the error string as a last resort
	return strings.Contains(strings.ToLower(err.Error()), "timeout") ||
		strings.Contains(strings.ToLower(err.Error()), "timed out")
}

// IsTemporary returns true if the error is temporary and can be retried
func IsTemporary(err error) bool {
	if err == nil {
		return false
	}

	// Check if it's our Error type
	if e, ok := err.(*Error); ok {
		return e.RetryStrategy != RetryNone
	}

	// Check the error string as a last resort
	return strings.Contains(strings.ToLower(err.Error()), "temporary") ||
		strings.Contains(strings.ToLower(err.Error()), "retry")
}

// IsCritical returns true if the error is critical
func IsCritical(err error) bool {
	if err == nil {
		return false
	}

	// Check if it's our Error type
	if e, ok := err.(*Error); ok {
		return e.Severity == SeverityCritical || e.Severity == SeverityFatal
	}

	// Check the error string as a last resort
	return strings.Contains(strings.ToLower(err.Error()), "critical") ||
		strings.Contains(strings.ToLower(err.Error()), "fatal")
}
