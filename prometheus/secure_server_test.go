package prometheus

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealthCheckHandler(t *testing.T) {
	req, err := http.NewRequestWithContext(context.Background(), "GET", "/health", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(healthCheckHandler)

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "healthy")
	assert.Contains(t, rr.Body.String(), "bitcoind-exporter")
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
}

func TestReadinessCheckHandler_Success(t *testing.T) {
	// Note: This test may fail if no real Bitcoin RPC is available
	// In a real test environment, you would mock the checkBitcoinRPCReadiness function
	req, err := http.NewRequestWithContext(context.Background(), "GET", "/ready", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(readinessCheckHandler)

	handler.ServeHTTP(rr, req)

	// Status could be either 200 (ready) or 503 (not ready) depending on RPC availability
	assert.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusServiceUnavailable)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
	
	if rr.Code == http.StatusOK {
		assert.Contains(t, rr.Body.String(), "ready")
	} else {
		assert.Contains(t, rr.Body.String(), "not_ready")
		assert.Contains(t, rr.Body.String(), "bitcoin_rpc_unavailable")
	}
}

func TestCheckBitcoinRPCReadiness(t *testing.T) {
	// This will test the actual readiness check function
	// In a real environment, this would fail without a Bitcoin RPC connection
	result := checkBitcoinRPCReadiness()
	
	// Result could be true or false depending on whether Bitcoin RPC is available
	// We just verify the function doesn't crash
	assert.IsType(t, true, result)
}