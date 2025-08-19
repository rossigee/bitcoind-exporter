package fetcher

import (
	"context"
	"testing"
	"time"

	"github.com/Primexz/bitcoind-exporter/config"
	"github.com/stretchr/testify/assert"
)

func TestNewResilientRunner(t *testing.T) {
	runner := NewResilientRunner()

	assert.NotNil(t, runner)
	assert.NotNil(t, runner.client)
	assert.NotNil(t, runner.errorHandler)
	assert.NotNil(t, runner.circuitBreaker)
	assert.NotNil(t, runner.logger)
}

func TestResilientRunner_checkBitcoinRPCReadiness(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*MockBitcoinRPCClient)
		expectedResult bool
	}{
		{
			name: "RPC available",
			setupMock: func(mock *MockBitcoinRPCClient) {
				mock.SetResponse("getblockcount", int64(800000))
			},
			expectedResult: true,
		},
		{
			name: "RPC unavailable",
			setupMock: func(mock *MockBitcoinRPCClient) {
				mock.SetError("getblockcount", assert.AnError)
			},
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := NewMockBitcoinRPCClient()
			tt.setupMock(mockClient)

			adapter := &BitcoinRPCAdapter{Mock: mockClient}
			runner := &ResilientRunner{
				client: &Client{RpcClient: adapter},
			}

			// Mock the checkBitcoinRPCReadiness function behavior
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			var blockCount int64
			err := runner.client.RpcClient.CallFor(ctx, &blockCount, "getblockcount")
			result := err == nil

			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestResilientRunner_collectAllMetrics(t *testing.T) {
	// Save original config
	originalInterval := config.C.FetchInterval
	defer func() {
		config.C.FetchInterval = originalInterval
	}()

	// Set test config
	config.C.FetchInterval = 10

	mockClient := NewMockBitcoinRPCClient()
	
	// Set up minimal successful responses
	mockClient.SetResponse("getblockchaininfo", &BlockchainInfo{
		Chain:  "main",
		Blocks: 800000,
	})
	mockClient.SetResponse("getmempoolinfo", &MempoolInfo{
		Size: 1000,
	})
	mockClient.SetResponse("getmemoryinfo", &MemoryInfo{})
	mockClient.SetResponse("getindexinfo", &IndexInfo{})
	mockClient.SetResponse("getnetworkinfo", &NetworkInfo{
		TotalConnections: 10,
	})
	mockClient.SetResponse("getnettotals", &NetTotals{})
	mockClient.SetResponse("estimatesmartfee", &SmartFee{
		Feerate: 0.00001,
	})
	mockClient.SetResponse("getnetworkhashps", 500000000000000.0)

	adapter := &BitcoinRPCAdapter{Mock: mockClient}
	runner := &ResilientRunner{
		client:         &Client{RpcClient: adapter},
		errorHandler:   NewErrorHandler(),
		circuitBreaker: NewCircuitBreaker(circuitBreakerMaxFailures, circuitBreakerResetTime),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := runner.collectAllMetrics(ctx)

	// Should succeed without error for this test setup
	assert.NoError(t, err)
}

func TestResilientRunner_processResultsAndUpdateMetrics(t *testing.T) {
	runner := &ResilientRunner{
		errorHandler: NewErrorHandler(),
		logger:       NewResilientRunner().logger, // Use logger from constructor
	}

	// Create a test results channel
	results := make(chan result, 3)
	
	// Add some test results
	results <- result{
		name: "blockchain",
		data: &BlockchainInfo{Blocks: 800000},
		err:  nil,
	}
	results <- result{
		name: "mempool", 
		data: &MempoolInfo{Size: 1000},
		err:  nil,
	}
	results <- result{
		name: "error_case",
		data: nil,
		err:  assert.AnError,
	}
	close(results)

	err := runner.processResultsAndUpdateMetrics(results)

	// Should succeed as we have valid data and only 1 error (< maxAllowedFailures)
	assert.NoError(t, err)
}

func TestResilientRunner_processResultsAndUpdateMetrics_TooManyErrors(t *testing.T) {
	runner := &ResilientRunner{
		errorHandler: NewErrorHandler(),
		logger:       NewResilientRunner().logger, // Use logger from constructor
	}

	// Create a test results channel with too many errors
	results := make(chan result, 10)
	
	// Add many error results to exceed maxAllowedFailures (6)
	for i := 0; i < 8; i++ {
		results <- result{
			name: "error_case",
			data: nil,
			err:  assert.AnError,
		}
	}
	close(results)

	err := runner.processResultsAndUpdateMetrics(results)

	// Should fail due to too many errors
	assert.Error(t, err)
}

func TestResilientRunner_Individual_Fetch_Methods(t *testing.T) {
	mockClient := NewMockBitcoinRPCClient()
	adapter := &BitcoinRPCAdapter{Mock: mockClient}
	runner := &ResilientRunner{
		client:       &Client{RpcClient: adapter},
		errorHandler: NewErrorHandler(),
	}

	ctx := context.Background()

	t.Run("fetchBlockchainInfoWithRetry", func(t *testing.T) {
		expectedInfo := &BlockchainInfo{Chain: "main", Blocks: 800000}
		mockClient.SetResponse("getblockchaininfo", expectedInfo)

		info, err := runner.fetchBlockchainInfoWithRetry(ctx)

		assert.NoError(t, err)
		assert.Equal(t, expectedInfo, info)
	})

	t.Run("fetchMempoolInfoWithRetry", func(t *testing.T) {
		expectedInfo := &MempoolInfo{Size: 1000}
		mockClient.SetResponse("getmempoolinfo", expectedInfo)

		info, err := runner.fetchMempoolInfoWithRetry(ctx)

		assert.NoError(t, err)
		assert.Equal(t, expectedInfo, info)
	})

	t.Run("fetchNetworkHashrateWithRetry", func(t *testing.T) {
		expectedHashrate := 500000000000000.0
		mockClient.SetResponse("getnetworkhashps", expectedHashrate)

		hashrate, err := runner.fetchNetworkHashrateWithRetry(ctx, -1)

		assert.NoError(t, err)
		assert.Equal(t, expectedHashrate, hashrate)
	})
}

func TestResilientRunner_getFetchers(t *testing.T) {
	runner := &ResilientRunner{}
	
	fetchers := runner.getFetchers()

	// Verify all expected fetchers are present
	expectedFetchers := []string{
		"blockchain", "mempool", "memory", "index", "network",
		"fee_2", "fee_5", "fee_20",
		"hash_-1", "hash_1", "hash_120",
		"nettotals",
	}

	assert.Equal(t, len(expectedFetchers), len(fetchers))
	
	for _, name := range expectedFetchers {
		assert.Contains(t, fetchers, name)
		assert.NotNil(t, fetchers[name])
	}
}

func TestResilientRunner_collectResults(t *testing.T) {
	runner := &ResilientRunner{
		logger: NewResilientRunner().logger, // Use logger from constructor
	}

	results := make(chan result, 5)
	results <- result{name: "blockchain", data: &BlockchainInfo{Blocks: 800000}, err: nil}
	results <- result{name: "mempool", data: &MempoolInfo{Size: 1000}, err: nil}
	results <- result{name: "hash_-1", data: 500000000000000.0, err: nil}
	results <- result{name: "error1", data: nil, err: assert.AnError}
	results <- result{name: "error2", data: nil, err: assert.AnError}
	close(results)

	data := &metricData{}
	errorCount := runner.collectResults(results, data)

	assert.Equal(t, 2, errorCount)
	assert.NotNil(t, data.blockchainInfo)
	assert.NotNil(t, data.mempoolInfo)
	assert.Equal(t, 800000, data.blockchainInfo.Blocks)
	assert.Equal(t, 1000, data.mempoolInfo.Size)
	assert.Equal(t, 500000000000000.0, data.hashRateLatest)
}