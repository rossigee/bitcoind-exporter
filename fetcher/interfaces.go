package fetcher

import (
	"context"
	"time"
)

// BitcoinRPCClient defines the interface for Bitcoin RPC operations
type BitcoinRPCClient interface {
	CallFor(ctx context.Context, result interface{}, method string, params ...interface{}) error
}

// MetricsCollector defines the interface for collecting and updating metrics
type MetricsCollector interface {
	UpdateBlockchainMetrics(info *BlockchainInfo)
	UpdateMempoolMetrics(info *MempoolInfo)
	UpdateMemoryMetrics(info *MemoryInfo)
	UpdateIndexMetrics(info *IndexInfo)
	UpdateNetworkMetrics(info *NetworkInfo, totals *NetTotals)
	UpdateFeeMetrics(feeRate2, feeRate5, feeRate20 *SmartFee)
	UpdateMiningMetrics(hashRateLatest, hashRate1, hashRate120 float64)
	UpdateScrapeTime(duration time.Duration)
}

// DataFetcher defines the interface for fetching Bitcoin node data
type DataFetcher interface {
	GetBlockchainInfo(ctx context.Context) (*BlockchainInfo, error)
	GetMempoolInfo(ctx context.Context) (*MempoolInfo, error)
	GetMemoryInfo(ctx context.Context) (*MemoryInfo, error)
	GetIndexInfo(ctx context.Context) (*IndexInfo, error)
	GetNetworkInfo(ctx context.Context) (*NetworkInfo, error)
	GetSmartFee(ctx context.Context, blocks int) (*SmartFee, error)
	GetNetworkHashrate(ctx context.Context, blocks int) (float64, error)
	GetNetTotals(ctx context.Context) (*NetTotals, error)
}

// ErrorHandler defines the interface for handling errors with retry logic
type ErrorHandler interface {
	HandleError(operation string, err error) error
	ShouldRetry(err error) bool
	GetRetryDelay(attempt int) time.Duration
}
