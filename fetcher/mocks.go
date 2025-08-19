package fetcher

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Mock configuration constants
const (
	mockRetryDelayMilliseconds = 100 // Mock retry delay in milliseconds
)

// MockBitcoinRPCClient is a mock implementation of BitcoinRPCClient for testing
type MockBitcoinRPCClient struct {
	mu        sync.RWMutex
	responses map[string]interface{}
	errors    map[string]error
	callCount map[string]int
}

// NewMockBitcoinRPCClient creates a new mock RPC client
func NewMockBitcoinRPCClient() *MockBitcoinRPCClient {
	return &MockBitcoinRPCClient{
		responses: make(map[string]interface{}),
		errors:    make(map[string]error),
		callCount: make(map[string]int),
	}
}

// Call implements the jsonrpc.RPCClient interface
func (m *MockBitcoinRPCClient) Call(method string, params ...interface{}) (interface{}, error) {
	m.mu.Lock()
	m.callCount[method]++
	m.mu.Unlock()

	m.mu.RLock()
	err, hasError := m.errors[method]
	response, hasResponse := m.responses[method]
	m.mu.RUnlock()

	if hasError {
		return nil, err
	}

	if hasResponse {
		return response, nil
	}

	return nil, errors.New("no mock response configured")
}

// CallFor implements BitcoinRPCClient interface
func (m *MockBitcoinRPCClient) CallFor(ctx context.Context, result interface{},
	method string, params ...interface{}) error {
	m.mu.Lock()
	m.callCount[method]++
	m.mu.Unlock()

	m.mu.RLock()
	err, hasError := m.errors[method]
	response, hasResponse := m.responses[method]
	m.mu.RUnlock()

	if hasError {
		return err
	}

	if hasResponse {
		return m.assignResponse(result, response)
	}

	return nil
}

// assignResponse handles type assertion and assignment for mock responses
func (m *MockBitcoinRPCClient) assignResponse(result interface{}, response interface{}) error {
	// Try to assign blockchain info types
	if m.assignBlockchainTypes(result, response) {
		return nil
	}

	// Try to assign network and fee types
	if m.assignNetworkTypes(result, response) {
		return nil
	}

	// Try to assign primitive types
	m.assignPrimitiveTypes(result, response)

	return nil
}

// assignBlockchainTypes handles blockchain-related type assignments
func (m *MockBitcoinRPCClient) assignBlockchainTypes(result interface{}, response interface{}) bool {
	switch v := result.(type) {
	case *BlockchainInfo:
		if resp, ok := response.(*BlockchainInfo); ok {
			*v = *resp
		}
		return true
	case **BlockchainInfo:
		if resp, ok := response.(*BlockchainInfo); ok {
			*v = resp
		}
		return true
	case *MempoolInfo:
		if resp, ok := response.(*MempoolInfo); ok {
			*v = *resp
		}
		return true
	case **MempoolInfo:
		if resp, ok := response.(*MempoolInfo); ok {
			*v = resp
		}
		return true
	case *MemoryInfo:
		if resp, ok := response.(*MemoryInfo); ok {
			*v = *resp
		}
		return true
	case **MemoryInfo:
		if resp, ok := response.(*MemoryInfo); ok {
			*v = resp
		}
		return true
	case *IndexInfo:
		if resp, ok := response.(*IndexInfo); ok {
			*v = *resp
		}
		return true
	case **IndexInfo:
		if resp, ok := response.(*IndexInfo); ok {
			*v = resp
		}
		return true
	}
	return false
}

// assignNetworkTypes handles network and fee-related type assignments
func (m *MockBitcoinRPCClient) assignNetworkTypes(result interface{}, response interface{}) bool {
	switch v := result.(type) {
	case *NetworkInfo:
		if resp, ok := response.(*NetworkInfo); ok {
			*v = *resp
		}
		return true
	case **NetworkInfo:
		if resp, ok := response.(*NetworkInfo); ok {
			*v = resp
		}
		return true
	case *SmartFee:
		if resp, ok := response.(*SmartFee); ok {
			*v = *resp
		}
		return true
	case **SmartFee:
		if resp, ok := response.(*SmartFee); ok {
			*v = resp
		}
		return true
	case *NetTotals:
		if resp, ok := response.(*NetTotals); ok {
			*v = *resp
		}
		return true
	case **NetTotals:
		if resp, ok := response.(*NetTotals); ok {
			*v = resp
		}
		return true
	}
	return false
}

// assignPrimitiveTypes handles primitive type assignments
func (m *MockBitcoinRPCClient) assignPrimitiveTypes(result interface{}, response interface{}) {
	switch v := result.(type) {
	case *float64:
		if resp, ok := response.(float64); ok {
			*v = resp
		}
	case **float64:
		if resp, ok := response.(float64); ok {
			*v = &resp
		}
	}
}

// SetResponse sets a mock response for a given method
func (m *MockBitcoinRPCClient) SetResponse(method string, response interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses[method] = response
}

// SetError sets a mock error for a given method
func (m *MockBitcoinRPCClient) SetError(method string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors[method] = err
}

// GetCallCount returns the number of times a method was called
func (m *MockBitcoinRPCClient) GetCallCount(method string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount[method]
}

// Reset clears all mock data
func (m *MockBitcoinRPCClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses = make(map[string]interface{})
	m.errors = make(map[string]error)
	m.callCount = make(map[string]int)
}

// MockMetricsCollector is a mock implementation of MetricsCollector for testing
type MockMetricsCollector struct {
	BlockchainUpdates int
	MempoolUpdates    int
	MemoryUpdates     int
	IndexUpdates      int
	NetworkUpdates    int
	FeeUpdates        int
	MiningUpdates     int
	ScrapeUpdates     int
	LastScrapeTime    time.Duration
}

// NewMockMetricsCollector creates a new mock metrics collector
func NewMockMetricsCollector() *MockMetricsCollector {
	return &MockMetricsCollector{}
}

func (m *MockMetricsCollector) UpdateBlockchainMetrics(info *BlockchainInfo) {
	m.BlockchainUpdates++
}

func (m *MockMetricsCollector) UpdateMempoolMetrics(info *MempoolInfo) {
	m.MempoolUpdates++
}

func (m *MockMetricsCollector) UpdateMemoryMetrics(info *MemoryInfo) {
	m.MemoryUpdates++
}

func (m *MockMetricsCollector) UpdateIndexMetrics(info *IndexInfo) {
	m.IndexUpdates++
}

func (m *MockMetricsCollector) UpdateNetworkMetrics(info *NetworkInfo, totals *NetTotals) {
	m.NetworkUpdates++
}

func (m *MockMetricsCollector) UpdateFeeMetrics(feeRate2, feeRate5, feeRate20 *SmartFee) {
	m.FeeUpdates++
}

func (m *MockMetricsCollector) UpdateMiningMetrics(hashRateLatest, hashRate1, hashRate120 float64) {
	m.MiningUpdates++
}

func (m *MockMetricsCollector) UpdateScrapeTime(duration time.Duration) {
	m.ScrapeUpdates++
	m.LastScrapeTime = duration
}

// Reset clears all counters
func (m *MockMetricsCollector) Reset() {
	m.BlockchainUpdates = 0
	m.MempoolUpdates = 0
	m.MemoryUpdates = 0
	m.IndexUpdates = 0
	m.NetworkUpdates = 0
	m.FeeUpdates = 0
	m.MiningUpdates = 0
	m.ScrapeUpdates = 0
	m.LastScrapeTime = 0
}

// MockErrorHandler is a mock implementation of ErrorHandler for testing
type MockErrorHandler struct {
	ShouldRetryResult bool
	RetryDelay        time.Duration
	HandledErrors     []error
}

// NewMockErrorHandler creates a new mock error handler
func NewMockErrorHandler() *MockErrorHandler {
	return &MockErrorHandler{
		ShouldRetryResult: true,
		RetryDelay:        time.Millisecond * mockRetryDelayMilliseconds,
		HandledErrors:     make([]error, 0),
	}
}

func (m *MockErrorHandler) HandleError(operation string, err error) error {
	m.HandledErrors = append(m.HandledErrors, err)
	return err
}

func (m *MockErrorHandler) ShouldRetry(err error) bool {
	return m.ShouldRetryResult
}

func (m *MockErrorHandler) GetRetryDelay(attempt int) time.Duration {
	return m.RetryDelay
}
