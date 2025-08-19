package fetcher

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRunner(t *testing.T) {
	runner := NewRunner()

	assert.NotNil(t, runner)
	assert.NotNil(t, runner.client)
}

func TestRunner_getBlockchainInfo_Success(t *testing.T) {
	mockClient := NewMockBitcoinRPCClient()
	expectedInfo := &BlockchainInfo{
		Chain:                "main",
		Blocks:               800000,
		Headers:              800000,
		BestBlockhash:        "000000000000000000023456789abcdef",
		Difficulty:           50000000000000.0,
		VerificationProgress: 1.0,
		SizeOnDisk:           500000000000,
	}

	mockClient.SetResponse("getblockchaininfo", expectedInfo)

	adapter := &BitcoinRPCAdapter{Mock: mockClient}
	runner := &Runner{client: &Client{RpcClient: adapter}}

	result := runner.getBlockchainInfo(context.Background())

	assert.NotNil(t, result)
	assert.Equal(t, expectedInfo.Chain, result.Chain)
	assert.Equal(t, expectedInfo.Blocks, result.Blocks)
	assert.Equal(t, expectedInfo.Headers, result.Headers)
	assert.Equal(t, 1, mockClient.GetCallCount("getblockchaininfo"))
}

func TestRunner_getBlockchainInfo_Error(t *testing.T) {
	mockClient := NewMockBitcoinRPCClient()
	mockClient.SetError("getblockchaininfo", assert.AnError)

	adapter := &BitcoinRPCAdapter{Mock: mockClient}
	runner := &Runner{client: &Client{RpcClient: adapter}}

	result := runner.getBlockchainInfo(context.Background())

	assert.Nil(t, result)
	assert.Equal(t, 1, mockClient.GetCallCount("getblockchaininfo"))
}

func TestRunner_getMempoolInfo_Success(t *testing.T) {
	mockClient := NewMockBitcoinRPCClient()
	expectedInfo := &MempoolInfo{
		Loaded:     true,
		Size:       1000,
		Bytes:      5000000,
		Usage:      10000000,
		TotalFee:   0.05,
		MaxMempool: 50000000,
	}

	mockClient.SetResponse("getmempoolinfo", expectedInfo)

	adapter := &BitcoinRPCAdapter{Mock: mockClient}
	runner := &Runner{client: &Client{RpcClient: adapter}}

	result := runner.getMempoolInfo(context.Background())

	assert.NotNil(t, result)
	assert.Equal(t, expectedInfo.Size, result.Size)
	assert.Equal(t, expectedInfo.Usage, result.Usage)
	assert.Equal(t, 1, mockClient.GetCallCount("getmempoolinfo"))
}

func TestRunner_getSmartFee_Success(t *testing.T) {
	mockClient := NewMockBitcoinRPCClient()
	expectedFee := &SmartFee{
		Feerate: 0.00001000,
		Blocks:  2,
	}

	mockClient.SetResponse("estimatesmartfee", expectedFee)

	adapter := &BitcoinRPCAdapter{Mock: mockClient}
	runner := &Runner{client: &Client{RpcClient: adapter}}

	result := runner.getSmartFee(context.Background(), 2)

	assert.NotNil(t, result)
	assert.Equal(t, expectedFee.Feerate, result.Feerate)
	assert.Equal(t, expectedFee.Blocks, result.Blocks)
	assert.Equal(t, 1, mockClient.GetCallCount("estimatesmartfee"))
}

func TestRunner_getNetworkHashrate_Success(t *testing.T) {
	mockClient := NewMockBitcoinRPCClient()
	expectedHashrate := 500000000000000.0

	mockClient.SetResponse("getnetworkhashps", expectedHashrate)

	adapter := &BitcoinRPCAdapter{Mock: mockClient}
	runner := &Runner{client: &Client{RpcClient: adapter}}

	result := runner.getNetworkHashrate(context.Background(), -1)

	assert.Equal(t, expectedHashrate, result)
	assert.Equal(t, 1, mockClient.GetCallCount("getnetworkhashps"))
}

func TestRunner_getNetworkHashrate_Error(t *testing.T) {
	mockClient := NewMockBitcoinRPCClient()
	mockClient.SetError("getnetworkhashps", assert.AnError)

	adapter := &BitcoinRPCAdapter{Mock: mockClient}
	runner := &Runner{client: &Client{RpcClient: adapter}}

	result := runner.getNetworkHashrate(context.Background(), -1)

	assert.Equal(t, float64(0), result)
	assert.Equal(t, 1, mockClient.GetCallCount("getnetworkhashps"))
}

// Benchmark tests for performance optimization
func BenchmarkRunner_getBlockchainInfo(b *testing.B) {
	mockClient := NewMockBitcoinRPCClient()
	mockClient.SetResponse("getblockchaininfo", &BlockchainInfo{
		Chain:  "main",
		Blocks: 800000,
	})

	adapter := &BitcoinRPCAdapter{Mock: mockClient}
	runner := &Runner{client: &Client{RpcClient: adapter}}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runner.getBlockchainInfo(context.Background())
	}
}

func BenchmarkRunner_getAllMetrics(b *testing.B) {
	mockClient := NewMockBitcoinRPCClient()

	// Set up all required responses
	mockClient.SetResponse("getblockchaininfo", &BlockchainInfo{Chain: "main", Blocks: 800000})
	mockClient.SetResponse("getmempoolinfo", &MempoolInfo{Size: 1000})
	mockClient.SetResponse("getmemoryinfo", &MemoryInfo{})
	mockClient.SetResponse("getindexinfo", &IndexInfo{})
	mockClient.SetResponse("getnetworkinfo", &NetworkInfo{TotalConnections: 10})
	mockClient.SetResponse("estimatesmartfee", &SmartFee{Feerate: 0.00001})
	mockClient.SetResponse("getnetworkhashps", 500000000000000.0)
	mockClient.SetResponse("getnettotals", &NetTotals{})

	adapter := &BitcoinRPCAdapter{Mock: mockClient}
	runner := &Runner{client: &Client{RpcClient: adapter}}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate the main collection process
		ctx := context.Background()
		runner.getBlockchainInfo(ctx)
		runner.getMempoolInfo(ctx)
		runner.getMemoryInfo(ctx)
		runner.getIndexInfo(ctx)
		runner.getNetworkInfo(ctx)
		runner.getSmartFee(ctx, 2)
		runner.getSmartFee(ctx, 5)
		runner.getSmartFee(ctx, 20)
		runner.getNetworkHashrate(ctx, -1)
		runner.getNetworkHashrate(ctx, 1)
		runner.getNetworkHashrate(ctx, 120)
		runner.getNetTotals(ctx)
	}
}

// Integration test helpers
func TestCreateTestRunner(t *testing.T) {
	// Helper function to create a test runner with mocked dependencies
	createTestRunner := func() (*Runner, *MockBitcoinRPCClient) {
		mockClient := NewMockBitcoinRPCClient()
		adapter := &BitcoinRPCAdapter{Mock: mockClient}
		runner := &Runner{
			client: &Client{RpcClient: adapter},
		}
		return runner, mockClient
	}

	runner, mockClient := createTestRunner()

	assert.NotNil(t, runner)
	assert.NotNil(t, mockClient)
	assert.NotNil(t, runner.client)
}

// Test timeout handling
func TestRunner_withTimeout(t *testing.T) {
	mockClient := NewMockBitcoinRPCClient()
	mockClient.SetResponse("getblockchaininfo", &BlockchainInfo{Chain: "main"})

	adapter := &BitcoinRPCAdapter{Mock: mockClient}
	runner := &Runner{client: &Client{RpcClient: adapter}}

	// This should complete quickly with mock
	start := time.Now()
	result := runner.getBlockchainInfo(context.Background())
	duration := time.Since(start)

	assert.NotNil(t, result)
	assert.Less(t, duration, time.Millisecond*50) // Should be very fast with mock
}

// Table-driven tests for multiple scenarios
func TestRunner_getSmartFee_MultipleBlocks(t *testing.T) {
	tests := []struct {
		name     string
		blocks   int
		expected *SmartFee
	}{
		{
			name:     "2 blocks",
			blocks:   2,
			expected: &SmartFee{Feerate: 0.00002000, Blocks: 2},
		},
		{
			name:     "5 blocks",
			blocks:   5,
			expected: &SmartFee{Feerate: 0.00001500, Blocks: 5},
		},
		{
			name:     "20 blocks",
			blocks:   20,
			expected: &SmartFee{Feerate: 0.00001000, Blocks: 20},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := NewMockBitcoinRPCClient()
			mockClient.SetResponse("estimatesmartfee", tt.expected)

			adapter := &BitcoinRPCAdapter{Mock: mockClient}
			runner := &Runner{client: &Client{RpcClient: adapter}}

			result := runner.getSmartFee(context.Background(), tt.blocks)

			require.NotNil(t, result)
			assert.Equal(t, tt.expected.Feerate, result.Feerate)
			assert.Equal(t, tt.expected.Blocks, result.Blocks)
		})
	}
}
