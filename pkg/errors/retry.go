package errors

import (
	"context"
	"math"
	"math/rand"
	"time"
)

// RetryConfig holds configuration for retry operations
type RetryConfig struct {
	// MaxRetries is the maximum number of retries
	MaxRetries int
	
	// InitialDelay is the initial delay between retries
	InitialDelay time.Duration
	
	// MaxDelay is the maximum delay between retries
	MaxDelay time.Duration
	
	// Factor is the factor by which the delay increases
	Factor float64
	
	// Jitter is the jitter factor for randomizing delays
	Jitter float64
	
	// RetryableErrors is a function that determines if an error is retryable
	RetryableErrors func(error) bool
}

// DefaultRetryConfig returns the default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:      3,
		InitialDelay:    100 * time.Millisecond,
		MaxDelay:        10 * time.Second,
		Factor:          2.0,
		Jitter:          0.2,
		RetryableErrors: IsTemporary,
	}
}

// Retry executes the given function with retries
func Retry(ctx context.Context, fn func() error, config RetryConfig) error {
	var err error
	
	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		// Execute the function
		err = fn()
		
		// If there's no error or the error is not retryable, return
		if err == nil || (config.RetryableErrors != nil && !config.RetryableErrors(err)) {
			return err
		}
		
		// If this was the last attempt, return the error
		if attempt == config.MaxRetries {
			return err
		}
		
		// Calculate the delay
		delay := calculateDelay(attempt, config)
		
		// Create a timer
		timer := time.NewTimer(delay)
		
		// Wait for the timer or context cancellation
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
			// Continue to the next attempt
		}
	}
	
	return err
}

// RetryWithResult executes the given function with retries and returns a result
func RetryWithResult[T any](ctx context.Context, fn func() (T, error), config RetryConfig) (T, error) {
	var result T
	var err error
	
	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		// Execute the function
		result, err = fn()
		
		// If there's no error or the error is not retryable, return
		if err == nil || (config.RetryableErrors != nil && !config.RetryableErrors(err)) {
			return result, err
		}
		
		// If this was the last attempt, return the error
		if attempt == config.MaxRetries {
			return result, err
		}
		
		// Calculate the delay
		delay := calculateDelay(attempt, config)
		
		// Create a timer
		timer := time.NewTimer(delay)
		
		// Wait for the timer or context cancellation
		select {
		case <-ctx.Done():
			timer.Stop()
			var zero T
			return zero, ctx.Err()
		case <-timer.C:
			// Continue to the next attempt
		}
	}
	
	return result, err
}

// calculateDelay calculates the delay for a retry attempt
func calculateDelay(attempt int, config RetryConfig) time.Duration {
	// Calculate the base delay using exponential backoff
	delay := float64(config.InitialDelay) * math.Pow(config.Factor, float64(attempt))
	
	// Apply jitter
	if config.Jitter > 0 {
		delay = delay * (1 - config.Jitter + config.Jitter*rand.Float64())
	}
	
	// Cap the delay at the maximum
	if delay > float64(config.MaxDelay) {
		delay = float64(config.MaxDelay)
	}
	
	return time.Duration(delay)
}

// RetryableFunc is a function that can be retried
type RetryableFunc func() error

// WithRetry wraps a function with retry logic
func WithRetry(fn RetryableFunc, config RetryConfig) RetryableFunc {
	return func() error {
		return Retry(context.Background(), fn, config)
	}
}

// RetryableFuncWithContext is a function that can be retried with a context
type RetryableFuncWithContext func(context.Context) error

// WithRetryContext wraps a function with retry logic and context
func WithRetryContext(fn RetryableFuncWithContext, config RetryConfig) RetryableFuncWithContext {
	return func(ctx context.Context) error {
		return Retry(ctx, func() error {
			return fn(ctx)
		}, config)
	}
}

// RetryableFuncWithResult is a function that can be retried and returns a result
type RetryableFuncWithResult[T any] func() (T, error)

// WithRetryResult wraps a function with retry logic and result
func WithRetryResult[T any](fn RetryableFuncWithResult[T], config RetryConfig) RetryableFuncWithResult[T] {
	return func() (T, error) {
		return RetryWithResult(context.Background(), fn, config)
	}
}

// RetryableFuncWithContextAndResult is a function that can be retried with a context and returns a result
type RetryableFuncWithContextAndResult[T any] func(context.Context) (T, error)

// WithRetryContextAndResult wraps a function with retry logic, context, and result
func WithRetryContextAndResult[T any](fn RetryableFuncWithContextAndResult[T], config RetryConfig) RetryableFuncWithContextAndResult[T] {
	return func(ctx context.Context) (T, error) {
		return RetryWithResult(ctx, func() (T, error) {
			return fn(ctx)
		}, config)
	}
}
