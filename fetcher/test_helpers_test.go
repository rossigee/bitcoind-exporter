package fetcher

import (
	"context"

	"github.com/ybbus/jsonrpc/v3"
)

// BitcoinRPCAdapter adapts our mock to the jsonrpc.RPCClient interface
type BitcoinRPCAdapter struct {
	Mock *MockBitcoinRPCClient
}

// Call implements jsonrpc.RPCClient interface
func (a *BitcoinRPCAdapter) Call(ctx context.Context, method string,
	params ...interface{}) (*jsonrpc.RPCResponse, error) {
	var result interface{}
	err := a.Mock.CallFor(ctx, &result, method, params...)
	if err != nil {
		return nil, err
	}

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
	return nil, nil
}

// CallBatchFor implements jsonrpc.RPCClient interface (not used in our tests)
func (a *BitcoinRPCAdapter) CallBatchFor(ctx context.Context, out []interface{}, requests jsonrpc.RPCRequests) error {
	return nil
}

// CallBatchRaw implements jsonrpc.RPCClient interface (not used in our tests)
func (a *BitcoinRPCAdapter) CallBatchRaw(ctx context.Context,
	requests jsonrpc.RPCRequests) (jsonrpc.RPCResponses, error) {
	return nil, nil
}

// CallRaw implements jsonrpc.RPCClient interface (not used in our tests)
func (a *BitcoinRPCAdapter) CallRaw(ctx context.Context, request *jsonrpc.RPCRequest) (*jsonrpc.RPCResponse, error) {
	return nil, nil
}