package fetcher

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/Primexz/bitcoind-exporter/config"
	prometheus "github.com/Primexz/bitcoind-exporter/prometheus/metrics"
	"github.com/Primexz/bitcoind-exporter/util"
	goprom "github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// Resilient fetcher configuration constants
const (
	circuitBreakerMaxFailures = 5               // Maximum failures before circuit breaker opens
	circuitBreakerResetTime   = time.Minute * 2 // Circuit breaker reset timeout
	totalRPCCallChannelSize   = 11              // Total number of RPC calls
	maxAllowedFailures        = 6               // Maximum allowed failures before error
)

// result represents the result of a metric fetch operation
type result struct {
	name string
	data interface{}
	err  error
}

// ResilientRunner wraps the original Runner with error handling and circuit breaker
type ResilientRunner struct {
	client         *Client
	errorHandler   *DefaultErrorHandler
	circuitBreaker *CircuitBreaker
	logger         *logrus.Entry
}

// NewResilientRunner creates a new resilient runner with error handling
func NewResilientRunner() *ResilientRunner {
	return &ResilientRunner{
		client:         NewClient(),
		errorHandler:   NewErrorHandler(),
		circuitBreaker: NewCircuitBreaker(circuitBreakerMaxFailures, circuitBreakerResetTime),
		logger: logrus.WithFields(logrus.Fields{
			"component": "resilient_fetcher",
		}),
	}
}

// StartResilient starts the resilient fetcher loop
func StartResilient() {
	runner := NewResilientRunner()

	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.C.FetchInterval-1)*time.Second)

		err := runner.runWithResilience(ctx)
		if err != nil {
			runner.logger.WithError(err).Error("Failed to collect metrics")
		}

		cancel()
		time.Sleep(time.Duration(config.C.FetchInterval) * time.Second)
	}
}

// runWithResilience executes the metrics collection with full error handling
func (r *ResilientRunner) runWithResilience(ctx context.Context) error {
	start := time.Now()

	// Use circuit breaker to protect against cascading failures
	err := r.circuitBreaker.Call(func() error {
		return r.collectAllMetrics(ctx)
	})

	// Always update scrape time, even on failure
	prometheus.ScrapeTime.Set(float64(time.Since(start).Milliseconds()))

	return err
}

// collectAllMetrics collects all metrics with concurrent execution and error handling
func (r *ResilientRunner) collectAllMetrics(ctx context.Context) error {
	// Create a context with timeout for all operations
	ctx, cancel := context.WithTimeout(ctx, time.Duration(config.C.FetchInterval-1)*time.Second)
	defer cancel()

	// Execute concurrent fetching
	results := r.executeConcurrentFetching(ctx)

	// Process results and update metrics
	return r.processResultsAndUpdateMetrics(results)
}

// executeConcurrentFetching runs all metric fetchers concurrently
func (r *ResilientRunner) executeConcurrentFetching(ctx context.Context) <-chan result {
	results := make(chan result, totalRPCCallChannelSize)
	var wg sync.WaitGroup

	// Get all fetchers
	fetchers := r.getFetchers()

	for name, fetcher := range fetchers {
		wg.Add(1)
		go func(name string, fetcher func(context.Context) (interface{}, error)) {
			defer wg.Done()
			data, err := fetcher(ctx)
			results <- result{name: name, data: data, err: err}
		}(name, fetcher)
	}

	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	return results
}

// getFetchers returns the map of all metric fetchers
func (r *ResilientRunner) getFetchers() map[string]func(context.Context) (interface{}, error) {
	return map[string]func(context.Context) (interface{}, error){
		"blockchain": r.fetchBlockchainInfoWithRetry,
		"mempool":    r.fetchMempoolInfoWithRetry,
		"memory":     r.fetchMemoryInfoWithRetry,
		"index":      r.fetchIndexInfoWithRetry,
		"network":    r.fetchNetworkInfoWithRetry,
		"fee_2": func(ctx context.Context) (interface{}, error) {
			return r.fetchSmartFeeWithRetry(ctx, feeEstimation2Blocks)
		},
		"fee_5": func(ctx context.Context) (interface{}, error) {
			return r.fetchSmartFeeWithRetry(ctx, feeEstimation5Blocks)
		},
		"fee_20": func(ctx context.Context) (interface{}, error) {
			return r.fetchSmartFeeWithRetry(ctx, feeEstimation20Blocks)
		},
		"hash_-1": func(ctx context.Context) (interface{}, error) {
			return r.fetchNetworkHashrateWithRetry(ctx, hashRateLatest)
		},
		"hash_1": func(ctx context.Context) (interface{}, error) {
			return r.fetchNetworkHashrateWithRetry(ctx, hashRate1Block)
		},
		"hash_120": func(ctx context.Context) (interface{}, error) {
			return r.fetchNetworkHashrateWithRetry(ctx, hashRate120Blocks)
		},
		"nettotals": r.fetchNetTotalsWithRetry,
	}
}

// metricData holds all collected metric data
type metricData struct {
	blockchainInfo *BlockchainInfo
	mempoolInfo    *MempoolInfo
	memoryInfo     *MemoryInfo
	indexInfo      *IndexInfo
	networkInfo    *NetworkInfo
	netTotals      *NetTotals
	feeRate2       *SmartFee
	feeRate5       *SmartFee
	feeRate20      *SmartFee
	hashRateLatest float64
	hashRate1      float64
	hashRate120    float64
}

// processResultsAndUpdateMetrics processes fetcher results and updates metrics
func (r *ResilientRunner) processResultsAndUpdateMetrics(results <-chan result) error {
	data := &metricData{}
	errorCount := r.collectResults(results, data)

	// If too many errors occurred, return error
	if errorCount > maxAllowedFailures {
		collectionErr := errors.New("too many metric collection failures")
		return r.errorHandler.HandleError("collect_metrics", collectionErr)
	}

	// Update metrics only if we have the data
	r.updateMetrics(data.blockchainInfo, data.mempoolInfo, data.memoryInfo, data.indexInfo,
		data.networkInfo, data.netTotals, data.feeRate2, data.feeRate5, data.feeRate20,
		data.hashRateLatest, data.hashRate1, data.hashRate120)

	return nil
}

// collectResults processes the results channel and populates metric data
func (r *ResilientRunner) collectResults(results <-chan result, data *metricData) int {
	errorCount := 0
	for res := range results {
		if res.err != nil {
			r.logger.WithError(res.err).WithField("metric", res.name).Error("Failed to fetch metric")
			errorCount++
			continue
		}

		switch res.name {
		case "blockchain":
			data.blockchainInfo = res.data.(*BlockchainInfo)
		case "mempool":
			data.mempoolInfo = res.data.(*MempoolInfo)
		case "memory":
			data.memoryInfo = res.data.(*MemoryInfo)
		case "index":
			data.indexInfo = res.data.(*IndexInfo)
		case "network":
			data.networkInfo = res.data.(*NetworkInfo)
		case "nettotals":
			data.netTotals = res.data.(*NetTotals)
		case "fee_2":
			data.feeRate2 = res.data.(*SmartFee)
		case "fee_5":
			data.feeRate5 = res.data.(*SmartFee)
		case "fee_20":
			data.feeRate20 = res.data.(*SmartFee)
		case "hash_-1":
			data.hashRateLatest = res.data.(float64)
		case "hash_1":
			data.hashRate1 = res.data.(float64)
		case "hash_120":
			data.hashRate120 = res.data.(float64)
		}
	}
	return errorCount
}

// Individual fetch methods with retry logic

func (r *ResilientRunner) fetchBlockchainInfoWithRetry(ctx context.Context) (interface{}, error) {
	var info *BlockchainInfo

	err := r.errorHandler.WithRetry(ctx, "getblockchaininfo", func() error {
		var fetchErr error
		info, fetchErr = r.getBlockchainInfo(ctx)
		return fetchErr
	})

	return info, err
}

func (r *ResilientRunner) fetchMempoolInfoWithRetry(ctx context.Context) (interface{}, error) {
	var info *MempoolInfo

	err := r.errorHandler.WithRetry(ctx, "getmempoolinfo", func() error {
		var fetchErr error
		info, fetchErr = r.getMempoolInfo(ctx)
		return fetchErr
	})

	return info, err
}

func (r *ResilientRunner) fetchMemoryInfoWithRetry(ctx context.Context) (interface{}, error) {
	var info *MemoryInfo

	err := r.errorHandler.WithRetry(ctx, "getmemoryinfo", func() error {
		var fetchErr error
		info, fetchErr = r.getMemoryInfo(ctx)
		return fetchErr
	})

	return info, err
}

func (r *ResilientRunner) fetchIndexInfoWithRetry(ctx context.Context) (interface{}, error) {
	var info *IndexInfo

	err := r.errorHandler.WithRetry(ctx, "getindexinfo", func() error {
		var fetchErr error
		info, fetchErr = r.getIndexInfo(ctx)
		return fetchErr
	})

	return info, err
}

func (r *ResilientRunner) fetchNetworkInfoWithRetry(ctx context.Context) (interface{}, error) {
	var info *NetworkInfo

	err := r.errorHandler.WithRetry(ctx, "getnetworkinfo", func() error {
		var fetchErr error
		info, fetchErr = r.getNetworkInfo(ctx)
		return fetchErr
	})

	return info, err
}

func (r *ResilientRunner) fetchSmartFeeWithRetry(ctx context.Context, blocks int) (interface{}, error) {
	var info *SmartFee

	err := r.errorHandler.WithRetry(ctx, "estimatesmartfee", func() error {
		var fetchErr error
		info, fetchErr = r.getSmartFee(ctx, blocks)
		return fetchErr
	})

	return info, err
}

func (r *ResilientRunner) fetchNetworkHashrateWithRetry(ctx context.Context, blocks int) (interface{}, error) {
	var hashrate float64

	err := r.errorHandler.WithRetry(ctx, "getnetworkhashps", func() error {
		var fetchErr error
		hashrate, fetchErr = r.getNetworkHashrate(ctx, blocks)
		return fetchErr
	})

	return hashrate, err
}

func (r *ResilientRunner) fetchNetTotalsWithRetry(ctx context.Context) (interface{}, error) {
	var info *NetTotals

	err := r.errorHandler.WithRetry(ctx, "getnettotals", func() error {
		var fetchErr error
		info, fetchErr = r.getNetTotals(ctx)
		return fetchErr
	})

	return info, err
}

// Context-aware RPC call methods

func (r *ResilientRunner) getBlockchainInfo(ctx context.Context) (*BlockchainInfo, error) {
	var info *BlockchainInfo
	err := r.client.RpcClient.CallFor(ctx, &info, "getblockchaininfo")
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (r *ResilientRunner) getMempoolInfo(ctx context.Context) (*MempoolInfo, error) {
	var info *MempoolInfo
	err := r.client.RpcClient.CallFor(ctx, &info, "getmempoolinfo")
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (r *ResilientRunner) getMemoryInfo(ctx context.Context) (*MemoryInfo, error) {
	var info *MemoryInfo
	err := r.client.RpcClient.CallFor(ctx, &info, "getmemoryinfo")
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (r *ResilientRunner) getIndexInfo(ctx context.Context) (*IndexInfo, error) {
	var info *IndexInfo
	err := r.client.RpcClient.CallFor(ctx, &info, "getindexinfo")
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (r *ResilientRunner) getNetworkInfo(ctx context.Context) (*NetworkInfo, error) {
	var info *NetworkInfo
	err := r.client.RpcClient.CallFor(ctx, &info, "getnetworkinfo")
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (r *ResilientRunner) getSmartFee(ctx context.Context, blocks int) (*SmartFee, error) {
	var info *SmartFee
	err := r.client.RpcClient.CallFor(ctx, &info, "estimatesmartfee", blocks)
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (r *ResilientRunner) getNetworkHashrate(ctx context.Context, blocks int) (float64, error) {
	var info float64
	err := r.client.RpcClient.CallFor(ctx, &info, "getnetworkhashps", blocks)
	if err != nil {
		return 0, err
	}
	return info, nil
}

func (r *ResilientRunner) getNetTotals(ctx context.Context) (*NetTotals, error) {
	var info *NetTotals
	err := r.client.RpcClient.CallFor(ctx, &info, "getnettotals")
	if err != nil {
		return nil, err
	}
	return info, nil
}

// updateMetrics updates Prometheus metrics (same logic as original)
func (r *ResilientRunner) updateMetrics(blockchainInfo *BlockchainInfo, mempoolInfo *MempoolInfo,
	memoryInfo *MemoryInfo, indexInfo *IndexInfo, networkInfo *NetworkInfo, netTotals *NetTotals,
	feeRate2, feeRate5, feeRate20 *SmartFee, hashRateLatest, hashRate1, hashRate120 float64) {
	// Only update metrics if we have the data
	if blockchainInfo != nil {
		prometheus.BlockchainBlocks.Set(float64(blockchainInfo.Blocks))
		prometheus.BlockchainHeaders.Set(float64(blockchainInfo.Headers))
		prometheus.BlockchainVerificationProgress.Set(blockchainInfo.VerificationProgress)
		prometheus.BlockchainSizeOnDisk.Set(float64(blockchainInfo.SizeOnDisk))
	}

	if mempoolInfo != nil {
		prometheus.MempoolUsage.Set(float64(mempoolInfo.Usage))
		prometheus.MempoolMax.Set(float64(mempoolInfo.MaxMempool))
		prometheus.MempoolTransactionCount.Set(float64(mempoolInfo.Size))
	}

	if memoryInfo != nil {
		prometheus.MemoryUsed.Set(float64(memoryInfo.Locked.Used))
		prometheus.MemoryFree.Set(float64(memoryInfo.Locked.Free))
		prometheus.MemoryTotal.Set(float64(memoryInfo.Locked.Total))
		prometheus.MemoryLocked.Set(float64(memoryInfo.Locked.Locked))
		prometheus.ChunksUsed.Set(float64(memoryInfo.Locked.ChunksUsed))
		prometheus.ChunksFree.Set(float64(memoryInfo.Locked.ChunksFree))
	}

	if indexInfo != nil {
		prometheus.TxIndexSynced.Set(util.BoolToFloat64(indexInfo.TxIndex.Synced))
		prometheus.TxIndexBestHeight.Set(float64(indexInfo.TxIndex.BestBlockHeight))
	}

	if networkInfo != nil && netTotals != nil {
		prometheus.TotalConnections.Set(float64(networkInfo.TotalConnections))
		prometheus.ConnectionsIn.Set(float64(networkInfo.ConnectionsIn))
		prometheus.ConnectionsOut.Set(float64(networkInfo.TotalConnections - networkInfo.ConnectionsIn))
		prometheus.TotalBytesRecv.Set(float64(netTotals.TotalBytesRecv))
		prometheus.TotalBytesSent.Set(float64(netTotals.TotalBytesSent))
	}

	// Update fee metrics if available
	if feeRate2 != nil {
		prometheus.SmartFee.With(goprom.Labels{"blocks": "2"}).Set(util.ConvertBTCkBToSatVb(feeRate2.Feerate))
	}
	if feeRate5 != nil {
		prometheus.SmartFee.With(goprom.Labels{"blocks": "5"}).Set(util.ConvertBTCkBToSatVb(feeRate5.Feerate))
	}
	if feeRate20 != nil {
		prometheus.SmartFee.With(goprom.Labels{"blocks": "20"}).Set(util.ConvertBTCkBToSatVb(feeRate20.Feerate))
	}

	// Update mining metrics
	prometheus.MiningHashrate.With(goprom.Labels{"blocks": "-1"}).Set(hashRateLatest)
	prometheus.MiningHashrate.With(goprom.Labels{"blocks": "1"}).Set(hashRate1)
	prometheus.MiningHashrate.With(goprom.Labels{"blocks": "120"}).Set(hashRate120)
}
