package fetcher

import (
	"context"
	"fmt"
	"math"
	"net"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Constants for error handler configuration
const (
	defaultMaxRetries    = 3
	defaultBackoffFactor = 2.0
	jitterModulus        = 1000
	jitterMultiplier     = 2
)

// RetryableError represents an error that can be retried
type RetryableError struct {
	Err     error
	Attempt int
}

func (e *RetryableError) Error() string {
	return fmt.Sprintf("retryable error (attempt %d): %v", e.Attempt, e.Err)
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

// DefaultErrorHandler implements the ErrorHandler interface with exponential backoff
type DefaultErrorHandler struct {
	maxRetries    int
	baseDelay     time.Duration
	maxDelay      time.Duration
	backoffFactor float64
	jitterEnabled bool
	logger        *logrus.Entry
}

// NewErrorHandler creates a new error handler with default settings
func NewErrorHandler() *DefaultErrorHandler {
	return &DefaultErrorHandler{
		maxRetries:    defaultMaxRetries,
		baseDelay:     time.Second,
		maxDelay:      time.Minute,
		backoffFactor: defaultBackoffFactor,
		jitterEnabled: true,
		logger: logrus.WithFields(logrus.Fields{
			"component": "error_handler",
		}),
	}
}

// HandleError processes an error and determines if it should be retried
func (h *DefaultErrorHandler) HandleError(operation string, err error) error {
	if err == nil {
		return nil
	}

	h.logger.WithFields(logrus.Fields{
		"operation": operation,
		"error":     err.Error(),
	}).Debug("Handling error")

	// Log different error types with appropriate levels
	switch {
	case isTemporaryError(err):
		h.logger.WithError(err).Warn("Temporary error occurred")
	case isNetworkError(err):
		h.logger.WithError(err).Error("Network error occurred")
	default:
		h.logger.WithError(err).Error("Unhandled error occurred")
	}

	return err
}

// ShouldRetry determines if an error should be retried
func (h *DefaultErrorHandler) ShouldRetry(err error) bool {
	if err == nil {
		return false
	}

	// Check for retryable error types
	if isTemporaryError(err) || isNetworkError(err) || isTimeoutError(err) {
		return true
	}

	// Check for specific RPC errors that are retryable
	if isRetryableRPCError(err) {
		return true
	}

	return false
}

// GetRetryDelay calculates the delay before the next retry
func (h *DefaultErrorHandler) GetRetryDelay(attempt int) time.Duration {
	if attempt >= h.maxRetries {
		return 0
	}

	// Exponential backoff: baseDelay * (backoffFactor ^ attempt)
	delay := time.Duration(float64(h.baseDelay) * math.Pow(h.backoffFactor, float64(attempt)))

	// Cap at maximum delay
	if delay > h.maxDelay {
		delay = h.maxDelay
	}

	// Add jitter to prevent thundering herd
	if h.jitterEnabled {
		randomFactor := float64(time.Now().UnixNano()%jitterModulus) / float64(jitterModulus) // 0.0 to 1.0
		jitterFactor := jitterMultiplier*randomFactor - 1                                     // -1.0 to 1.0
		jitter := time.Duration(float64(delay) * 0.1 * jitterFactor)
		delay += jitter
	}

	return delay
}

// WithRetry executes a function with retry logic
func (h *DefaultErrorHandler) WithRetry(ctx context.Context, operation string, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt < h.maxRetries; attempt++ {
		// Execute the function
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = h.HandleError(operation, err)

		// Check if we should retry
		if !h.ShouldRetry(err) {
			h.logger.WithFields(logrus.Fields{
				"operation": operation,
				"attempt":   attempt + 1,
				"error":     err.Error(),
			}).Info("Error is not retryable, giving up")
			return err
		}

		// Check if we've reached max retries
		if attempt == h.maxRetries-1 {
			h.logger.WithFields(logrus.Fields{
				"operation":   operation,
				"max_retries": h.maxRetries,
				"final_error": err.Error(),
			}).Error("Max retries reached, giving up")
			return &RetryableError{Err: err, Attempt: attempt + 1}
		}

		// Calculate delay and wait
		delay := h.GetRetryDelay(attempt)
		h.logger.WithFields(logrus.Fields{
			"operation": operation,
			"attempt":   attempt + 1,
			"delay":     delay,
			"error":     err.Error(),
		}).Info("Retrying after delay")

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return lastErr
}

// Error classification functions

func isTemporaryError(err error) bool {
	if err == nil {
		return false
	}

	// Check if error implements Temporary interface
	type temporary interface {
		Temporary() bool
	}

	if temp, ok := err.(temporary); ok {
		return temp.Temporary()
	}

	return false
}

func isNetworkError(err error) bool {
	if err == nil {
		return false
	}

	// Check for common network errors
	if _, ok := err.(*net.OpError); ok {
		return true
	}

	if _, ok := err.(*net.DNSError); ok {
		return true
	}

	// Check error message for network-related keywords
	errMsg := strings.ToLower(err.Error())
	networkKeywords := []string{
		"connection refused",
		"connection timeout",
		"network unreachable",
		"host unreachable",
		"no route to host",
		"connection reset",
		"broken pipe",
	}

	for _, keyword := range networkKeywords {
		if strings.Contains(errMsg, keyword) {
			return true
		}
	}

	return false
}

func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	// Check if error implements Timeout interface
	type timeout interface {
		Timeout() bool
	}

	if timeout, ok := err.(timeout); ok {
		return timeout.Timeout()
	}

	// Check for context timeout
	if err == context.DeadlineExceeded {
		return true
	}

	// Check error message for timeout keywords
	errMsg := strings.ToLower(err.Error())
	timeoutKeywords := []string{
		"timeout",
		"deadline exceeded",
		"request timeout",
		"operation timed out",
	}

	for _, keyword := range timeoutKeywords {
		if strings.Contains(errMsg, keyword) {
			return true
		}
	}

	return false
}

func isRetryableRPCError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())

	// Bitcoin RPC specific retryable errors
	retryableRPCErrors := []string{
		"rpc temporarily unavailable",
		"loading block index",
		"verifying blocks",
		"rescanning",
		"server busy",
		"try again",
		"-28", // RPC in warmup
		"-1",  // RPC misc error (sometimes temporary)
	}

	for _, rpcError := range retryableRPCErrors {
		if strings.Contains(errMsg, rpcError) {
			return true
		}
	}

	return false
}

// CircuitBreaker implements a circuit breaker pattern
type CircuitBreaker struct {
	maxFailures     int
	resetTimeout    time.Duration
	state           CircuitState
	failures        int
	lastFailureTime time.Time
	logger          *logrus.Entry
}

type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
		state:        CircuitClosed,
		logger: logrus.WithFields(logrus.Fields{
			"component": "circuit_breaker",
		}),
	}
}

// Call executes a function with circuit breaker protection
func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.updateState()

	switch cb.state {
	case CircuitOpen:
		cb.logger.Debug("Circuit breaker is open, rejecting call")
		return fmt.Errorf("circuit breaker is open")
	case CircuitHalfOpen:
		cb.logger.Debug("Circuit breaker is half-open, allowing one call")
	case CircuitClosed:
		cb.logger.Debug("Circuit breaker is closed, allowing call")
	}

	err := fn()

	if err != nil {
		cb.recordFailure()
		cb.logger.WithError(err).Debug("Circuit breaker recorded failure")
	} else {
		cb.recordSuccess()
		cb.logger.Debug("Circuit breaker recorded success")
	}

	return err
}

func (cb *CircuitBreaker) updateState() {
	switch cb.state {
	case CircuitClosed:
		// No action needed in closed state
	case CircuitOpen:
		if time.Since(cb.lastFailureTime) > cb.resetTimeout {
			cb.state = CircuitHalfOpen
			cb.logger.Info("Circuit breaker transitioning from open to half-open")
		}
	case CircuitHalfOpen:
		// No action needed in half-open state
	}
}

func (cb *CircuitBreaker) recordFailure() {
	cb.failures++
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case CircuitClosed:
		if cb.failures >= cb.maxFailures {
			cb.state = CircuitOpen
			cb.logger.WithField("failures", cb.failures).Warn("Circuit breaker opened due to failures")
		}
	case CircuitOpen:
		// Already open, no state change needed
	case CircuitHalfOpen:
		cb.state = CircuitOpen
		cb.logger.Info("Circuit breaker returned to open state after half-open failure")
	}
}

func (cb *CircuitBreaker) recordSuccess() {
	cb.failures = 0

	switch cb.state {
	case CircuitClosed:
		// Already closed, no state change needed
	case CircuitOpen:
		// Open state success should not happen, but reset failures
	case CircuitHalfOpen:
		cb.state = CircuitClosed
		cb.logger.Info("Circuit breaker closed after successful half-open call")
	}
}

// GetState returns the current circuit breaker state
func (cb *CircuitBreaker) GetState() CircuitState {
	return cb.state
}

// GetFailures returns the current failure count
func (cb *CircuitBreaker) GetFailures() int {
	return cb.failures
}
