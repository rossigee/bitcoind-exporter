package fetcher

import (
	"context"

	"github.com/ybbus/jsonrpc/v3"
)

// TestClient wraps MockBitcoinRPCClient for testing
type TestClient struct {
	Mock *MockBitcoinRPCClient
}

// NewTestClient creates a new test client
func NewTestClient() *TestClient {
	return &TestClient{
		Mock: NewMockBitcoinRPCClient(),
	}
}

// TestRunner creates a runner for testing
type TestRunner struct {
	*Runner
	MockClient *MockBitcoinRPCClient
}

// NewTestRunner creates a new test runner with mock client
func NewTestRunner() *TestRunner {
	mockClient := NewMockBitcoinRPCClient()

	// Create a wrapper that implements both interfaces
	adapter := &BitcoinRPCAdapter{
		Mock: mockClient,
	}

	runner := &Runner{
		client: &Client{RpcClient: adapter},
	}
	return &TestRunner{
		Runner:     runner,
		MockClient: mockClient,
	}
}

// BitcoinRPCAdapter adapts our mock to the jsonrpc.RPCClient interface
type BitcoinRPCAdapter struct {
	Mock BitcoinRPCClient
}

// Call implements jsonrpc.RPCClient interface
func (a *BitcoinRPCAdapter) Call(ctx context.Context, method string,
	params ...interface{}) (*jsonrpc.RPCResponse, error) {
	var result interface{}
	err := a.Mock.CallFor(ctx, &result, method, params...)
	if err != nil {
		return nil, err
	}

	// Create a mock response (simplified)
	return &jsonrpc.RPCResponse{
		Result: result,
		Error:  nil,
	}, nil
}

// CallFor implements jsonrpc.RPCClient interface
func (a *BitcoinRPCAdapter) CallFor(ctx context.Context, out interface{}, method string, params ...interface{}) error {
	return a.Mock.CallFor(ctx, out, method, params...)
}

// CallBatch implements jsonrpc.RPCClient interface (not used in our tests)
func (a *BitcoinRPCAdapter) CallBatch(ctx context.Context, requests jsonrpc.RPCRequests) (jsonrpc.RPCResponses, error) {
	// Simple implementation for testing - not used in our current tests
	return nil, nil
}

// CallBatchFor implements jsonrpc.RPCClient interface (not used in our tests)
func (a *BitcoinRPCAdapter) CallBatchFor(ctx context.Context, out []interface{}, requests jsonrpc.RPCRequests) error {
	// Simple implementation for testing - not used in our current tests
	return nil
}

// CallBatchRaw implements jsonrpc.RPCClient interface (not used in our tests)
func (a *BitcoinRPCAdapter) CallBatchRaw(ctx context.Context,
	requests jsonrpc.RPCRequests) (jsonrpc.RPCResponses, error) {
	// Simple implementation for testing - not used in our current tests
	return nil, nil
}

// CallRaw implements jsonrpc.RPCClient interface (not used in our tests)
func (a *BitcoinRPCAdapter) CallRaw(ctx context.Context, request *jsonrpc.RPCRequest) (*jsonrpc.RPCResponse, error) {
	// Simple implementation for testing - not used in our current tests
	return nil, nil
}
