package circuitbreaker

import (
	"sync"
	"sync/atomic"
	"time"

	errbuilder "github.com/ZanzyTHEbar/errbuilder-go"
	"github.com/ZanzyTHEbar/thothnetwork/pkg/logger"
)

// State represents the state of the circuit breaker
type State int32

const (
	// StateClosed means the circuit breaker is closed and requests are allowed
	StateClosed State = iota
	// StateOpen means the circuit breaker is open and requests are not allowed
	StateOpen
	// StateHalfOpen means the circuit breaker is half-open and a limited number of requests are allowed
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	// name is the name of the circuit breaker
	name string

	// state is the current state of the circuit breaker
	state int32

	// failureThreshold is the number of failures that will trip the circuit
	failureThreshold int64

	// failureCount is the current number of consecutive failures
	failureCount int64

	// resetTimeout is the time to wait before transitioning from open to half-open
	resetTimeout time.Duration

	// halfOpenMaxRequests is the maximum number of requests allowed in half-open state
	halfOpenMaxRequests int64

	// halfOpenRequestCount is the current number of requests in half-open state
	halfOpenRequestCount int64

	// lastStateChange is the time of the last state change
	lastStateChange time.Time

	// mutex for protecting state changes
	mu sync.RWMutex

	// logger for logging circuit breaker events
	logger logger.Logger
}

// Config holds configuration for the circuit breaker
type Config struct {
	// Name is the name of the circuit breaker
	Name string

	// FailureThreshold is the number of failures that will trip the circuit
	FailureThreshold int64

	// ResetTimeout is the time to wait before transitioning from open to half-open
	ResetTimeout time.Duration

	// HalfOpenMaxRequests is the maximum number of requests allowed in half-open state
	HalfOpenMaxRequests int64

	// Logger is the logger for circuit breaker events
	Logger logger.Logger
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config Config) *CircuitBreaker {
	if config.FailureThreshold <= 0 {
		config.FailureThreshold = 5
	}
	if config.ResetTimeout <= 0 {
		config.ResetTimeout = 30 * time.Second
	}
	if config.HalfOpenMaxRequests <= 0 {
		config.HalfOpenMaxRequests = 1
	}

	return &CircuitBreaker{
		name:                config.Name,
		state:               int32(StateClosed),
		failureThreshold:    config.FailureThreshold,
		resetTimeout:        config.ResetTimeout,
		halfOpenMaxRequests: config.HalfOpenMaxRequests,
		lastStateChange:     time.Now(),
		logger:              config.Logger.With("component", "circuit_breaker", "name", config.Name),
	}
}

// Execute executes the given function with circuit breaker protection
func (cb *CircuitBreaker) Execute(fn func() error) error {
	// Check if the circuit is open
	if !cb.AllowRequest() {

		var errs errbuilder.ErrorMap
		errs.Set("circuit_breaker", cb.name)
		errs.Set("state", State(cb.state).String())
		errs.Set("failure_threshold", cb.failureThreshold)
		errs.Set("reset_timeout", cb.resetTimeout.String())
		errs.Set("half_open_max_requests", cb.halfOpenMaxRequests)

		details := errbuilder.NewErrDetails(errs)

		return errbuilder.NewErrBuilder().
			WithMsg("Circuit breaker is open").
			WithCode(errbuilder.CodeUnavailable).
			WithDetails(details)
	}

	// Execute the function
	err := fn()

	// Record the result
	if err != nil {
		cb.RecordFailure()
		return err
	}

	cb.RecordSuccess()
	return nil
}

// AllowRequest checks if a request is allowed based on the current state
func (cb *CircuitBreaker) AllowRequest() bool {
	state := State(atomic.LoadInt32(&cb.state))

	switch state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if reset timeout has elapsed
		cb.mu.RLock()
		elapsed := time.Since(cb.lastStateChange)
		cb.mu.RUnlock()

		if elapsed > cb.resetTimeout {
			// Transition to half-open state
			cb.transitionToHalfOpen()
			return true
		}
		return false
	case StateHalfOpen:
		// Allow a limited number of requests in half-open state
		return atomic.AddInt64(&cb.halfOpenRequestCount, 1) <= cb.halfOpenMaxRequests
	default:
		return false
	}
}

// RecordSuccess records a successful request
func (cb *CircuitBreaker) RecordSuccess() {
	state := State(atomic.LoadInt32(&cb.state))

	switch state {
	case StateHalfOpen:
		// Reset failure count and transition to closed state
		atomic.StoreInt64(&cb.failureCount, 0)
		cb.transitionToClosed()
	case StateClosed:
		// Reset failure count
		atomic.StoreInt64(&cb.failureCount, 0)
	}
}

// RecordFailure records a failed request
func (cb *CircuitBreaker) RecordFailure() {
	state := State(atomic.LoadInt32(&cb.state))

	switch state {
	case StateClosed:
		// Increment failure count
		newCount := atomic.AddInt64(&cb.failureCount, 1)

		// Check if failure threshold is reached
		if newCount >= cb.failureThreshold {
			cb.transitionToOpen()
		}
	case StateHalfOpen:
		// Transition back to open state
		cb.transitionToOpen()
	}
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() State {
	return State(atomic.LoadInt32(&cb.state))
}

// transitionToClosed transitions the circuit breaker to closed state
func (cb *CircuitBreaker) transitionToClosed() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if State(cb.state) != StateClosed {
		cb.logger.Info("Circuit breaker transitioning to closed state")
		atomic.StoreInt32(&cb.state, int32(StateClosed))
		atomic.StoreInt64(&cb.failureCount, 0)
		atomic.StoreInt64(&cb.halfOpenRequestCount, 0)
		cb.lastStateChange = time.Now()
	}
}

// transitionToOpen transitions the circuit breaker to open state
func (cb *CircuitBreaker) transitionToOpen() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if State(cb.state) != StateOpen {
		cb.logger.Info("Circuit breaker transitioning to open state", "failures", cb.failureCount)
		atomic.StoreInt32(&cb.state, int32(StateOpen))
		cb.lastStateChange = time.Now()
	}
}

// transitionToHalfOpen transitions the circuit breaker to half-open state
func (cb *CircuitBreaker) transitionToHalfOpen() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if State(cb.state) != StateHalfOpen {
		cb.logger.Info("Circuit breaker transitioning to half-open state")
		atomic.StoreInt32(&cb.state, int32(StateHalfOpen))
		atomic.StoreInt64(&cb.halfOpenRequestCount, 0)
		cb.lastStateChange = time.Now()
	}
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.transitionToClosed()
}

// GetFailureCount returns the current failure count
func (cb *CircuitBreaker) GetFailureCount() int64 {
	return atomic.LoadInt64(&cb.failureCount)
}

// GetLastStateChange returns the time of the last state change
func (cb *CircuitBreaker) GetLastStateChange() time.Time {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.lastStateChange
}
