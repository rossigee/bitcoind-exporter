package fetcher

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewErrorHandler(t *testing.T) {
	handler := NewErrorHandler()

	assert.NotNil(t, handler)
	assert.Equal(t, 3, handler.maxRetries)
	assert.Equal(t, time.Second, handler.baseDelay)
	assert.Equal(t, time.Minute, handler.maxDelay)
	assert.Equal(t, 2.0, handler.backoffFactor)
	assert.True(t, handler.jitterEnabled)
}

func TestErrorHandler_HandleError(t *testing.T) {
	handler := NewErrorHandler()

	t.Run("nil error", func(t *testing.T) {
		err := handler.HandleError("test", nil)
		assert.NoError(t, err)
	})

	t.Run("regular error", func(t *testing.T) {
		originalErr := errors.New("test error")
		err := handler.HandleError("test", originalErr)
		assert.Equal(t, originalErr, err)
	})
}

func TestErrorHandler_ShouldRetry(t *testing.T) {
	handler := NewErrorHandler()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "network error",
			err:      &net.OpError{Op: "dial"},
			expected: true,
		},
		{
			name:     "dns error",
			err:      &net.DNSError{},
			expected: true,
		},
		{
			name:     "timeout error",
			err:      context.DeadlineExceeded,
			expected: true,
		},
		{
			name:     "connection refused",
			err:      errors.New("connection refused"),
			expected: true,
		},
		{
			name:     "rpc warmup error",
			err:      errors.New("RPC error: -28"),
			expected: true,
		},
		{
			name:     "regular error",
			err:      errors.New("regular error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.ShouldRetry(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestErrorHandler_GetRetryDelay(t *testing.T) {
	handler := NewErrorHandler()

	tests := []struct {
		name     string
		attempt  int
		expected time.Duration
	}{
		{
			name:     "first attempt",
			attempt:  0,
			expected: time.Second,
		},
		{
			name:     "second attempt",
			attempt:  1,
			expected: 2 * time.Second,
		},
		{
			name:     "third attempt",
			attempt:  2,
			expected: 4 * time.Second,
		},
		{
			name:     "beyond max retries",
			attempt:  3,
			expected: 0,
		},
	}

	// Disable jitter for predictable testing
	handler.jitterEnabled = false

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.GetRetryDelay(tt.attempt)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestErrorHandler_GetRetryDelay_MaxDelay(t *testing.T) {
	handler := NewErrorHandler()
	handler.maxDelay = 5 * time.Second
	handler.jitterEnabled = false

	// Test that delay is capped at maxDelay
	delay := handler.GetRetryDelay(10)       // Very high attempt
	assert.Equal(t, time.Duration(0), delay) // Should be 0 because attempt > maxRetries

	// Test with attempt within range but would exceed maxDelay
	handler.maxRetries = 10
	delay = handler.GetRetryDelay(3) // 2^3 * 1s = 8s, should be capped at 5s
	assert.Equal(t, 5*time.Second, delay)
}

func TestErrorHandler_WithRetry_Success(t *testing.T) {
	handler := NewErrorHandler()
	ctx := context.Background()

	callCount := 0
	fn := func() error {
		callCount++
		return nil
	}

	err := handler.WithRetry(ctx, "test", fn)

	assert.NoError(t, err)
	assert.Equal(t, 1, callCount)
}

func TestErrorHandler_WithRetry_SuccessAfterRetries(t *testing.T) {
	handler := NewErrorHandler()
	handler.baseDelay = time.Millisecond // Speed up test
	ctx := context.Background()

	callCount := 0
	fn := func() error {
		callCount++
		if callCount < 3 {
			return errors.New("connection refused") // Retryable error
		}
		return nil
	}

	err := handler.WithRetry(ctx, "test", fn)

	assert.NoError(t, err)
	assert.Equal(t, 3, callCount)
}

func TestErrorHandler_WithRetry_NonRetryableError(t *testing.T) {
	handler := NewErrorHandler()
	ctx := context.Background()

	callCount := 0
	originalErr := errors.New("non-retryable error")
	fn := func() error {
		callCount++
		return originalErr
	}

	err := handler.WithRetry(ctx, "test", fn)

	assert.Equal(t, originalErr, err)
	assert.Equal(t, 1, callCount)
}

func TestErrorHandler_WithRetry_MaxRetriesExceeded(t *testing.T) {
	handler := NewErrorHandler()
	handler.baseDelay = time.Millisecond // Speed up test
	ctx := context.Background()

	callCount := 0
	fn := func() error {
		callCount++
		return errors.New("connection refused") // Always retryable error
	}

	err := handler.WithRetry(ctx, "test", fn)

	assert.Error(t, err)
	assert.IsType(t, &RetryableError{}, err)
	assert.Equal(t, 3, callCount) // maxRetries
}

func TestErrorHandler_WithRetry_ContextCanceled(t *testing.T) {
	handler := NewErrorHandler()
	handler.baseDelay = time.Second // Long delay to test cancellation

	ctx, cancel := context.WithCancel(context.Background())

	callCount := 0
	fn := func() error {
		callCount++
		if callCount == 1 {
			// Cancel context after first call to test cancellation during retry delay
			go func() {
				time.Sleep(10 * time.Millisecond)
				cancel()
			}()
		}
		return errors.New("connection refused") // Retryable error
	}

	err := handler.WithRetry(ctx, "test", fn)

	assert.Equal(t, context.Canceled, err)
	assert.Equal(t, 1, callCount)
}

// Test error classification functions

func TestIsNetworkError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "net.OpError",
			err:      &net.OpError{Op: "dial"},
			expected: true,
		},
		{
			name:     "net.DNSError",
			err:      &net.DNSError{},
			expected: true,
		},
		{
			name:     "connection refused message",
			err:      errors.New("connection refused"),
			expected: true,
		},
		{
			name:     "connection timeout message",
			err:      errors.New("connection timeout"),
			expected: true,
		},
		{
			name:     "regular error",
			err:      errors.New("regular error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNetworkError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsTimeoutError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "context deadline exceeded",
			err:      context.DeadlineExceeded,
			expected: true,
		},
		{
			name:     "timeout message",
			err:      errors.New("operation timed out"),
			expected: true,
		},
		{
			name:     "deadline message",
			err:      errors.New("deadline exceeded"),
			expected: true,
		},
		{
			name:     "regular error",
			err:      errors.New("regular error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTimeoutError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsRetryableRPCError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "RPC warmup error",
			err:      errors.New("RPC error: -28"),
			expected: true,
		},
		{
			name:     "loading block index",
			err:      errors.New("loading block index"),
			expected: true,
		},
		{
			name:     "server busy",
			err:      errors.New("server busy"),
			expected: true,
		},
		{
			name:     "regular RPC error",
			err:      errors.New("RPC error: invalid parameter"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryableRPCError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Circuit Breaker Tests

func TestNewCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(5, time.Minute)

	assert.NotNil(t, cb)
	assert.Equal(t, 5, cb.maxFailures)
	assert.Equal(t, time.Minute, cb.resetTimeout)
	assert.Equal(t, CircuitClosed, cb.state)
	assert.Equal(t, 0, cb.failures)
}

func TestCircuitBreaker_Call_Success(t *testing.T) {
	cb := NewCircuitBreaker(3, time.Minute)

	callCount := 0
	fn := func() error {
		callCount++
		return nil
	}

	err := cb.Call(fn)

	assert.NoError(t, err)
	assert.Equal(t, 1, callCount)
	assert.Equal(t, CircuitClosed, cb.GetState())
	assert.Equal(t, 0, cb.GetFailures())
}

func TestCircuitBreaker_Call_FailuresOpenCircuit(t *testing.T) {
	cb := NewCircuitBreaker(2, time.Minute)

	// First failure
	err := cb.Call(func() error { return errors.New("error1") })
	assert.Error(t, err)
	assert.Equal(t, CircuitClosed, cb.GetState())
	assert.Equal(t, 1, cb.GetFailures())

	// Second failure - should open circuit
	err = cb.Call(func() error { return errors.New("error2") })
	assert.Error(t, err)
	assert.Equal(t, CircuitOpen, cb.GetState())
	assert.Equal(t, 2, cb.GetFailures())

	// Third call - should be rejected
	err = cb.Call(func() error { return nil })
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker is open")
}

func TestCircuitBreaker_HalfOpen_Success(t *testing.T) {
	cb := NewCircuitBreaker(1, time.Millisecond*10)

	// Trigger circuit to open
	err := cb.Call(func() error { return errors.New("error") })
	assert.Error(t, err)
	assert.Equal(t, CircuitOpen, cb.GetState())

	// Wait for reset timeout
	time.Sleep(time.Millisecond * 15)

	// Next call should be allowed (half-open) and succeed
	err = cb.Call(func() error { return nil })
	assert.NoError(t, err)
	assert.Equal(t, CircuitClosed, cb.GetState())
	assert.Equal(t, 0, cb.GetFailures())
}

func TestCircuitBreaker_HalfOpen_Failure(t *testing.T) {
	cb := NewCircuitBreaker(1, time.Millisecond*10)

	// Trigger circuit to open
	err := cb.Call(func() error { return errors.New("error") })
	assert.Error(t, err)
	assert.Equal(t, CircuitOpen, cb.GetState())

	// Wait for reset timeout
	time.Sleep(time.Millisecond * 15)

	// Next call should be allowed (half-open) but fail
	err = cb.Call(func() error { return errors.New("error") })
	assert.Error(t, err)
	assert.Equal(t, CircuitOpen, cb.GetState())
}

// Benchmark tests

func BenchmarkErrorHandler_WithRetry_Success(b *testing.B) {
	handler := NewErrorHandler()
	ctx := context.Background()

	fn := func() error { return nil }

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = handler.WithRetry(ctx, "test", fn)
	}
}

func BenchmarkCircuitBreaker_Call_Success(b *testing.B) {
	cb := NewCircuitBreaker(5, time.Minute)
	fn := func() error { return nil }

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cb.Call(fn)
	}
}
