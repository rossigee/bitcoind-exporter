package fetcher

import (
	"context"
	"time"

	"github.com/Primexz/bitcoind-exporter/config"
	prometheus "github.com/Primexz/bitcoind-exporter/prometheus/metrics"
	goprom "github.com/prometheus/client_golang/prometheus"

	"github.com/Primexz/bitcoind-exporter/util"
	"github.com/sirupsen/logrus"
)

// Bitcoin network constants
const (
	// Fee estimation blocks
	feeEstimation2Blocks  = 2  // Fast confirmation
	feeEstimation5Blocks  = 5  // Medium confirmation
	feeEstimation20Blocks = 20 // Slow confirmation

	// Hash rate estimation blocks
	hashRateLatest    = -1  // Latest hash rate
	hashRate1Block    = 1   // 1 block average
	hashRate120Blocks = 120 // 120 block average (~20 hours)
)

var log = logrus.WithFields(logrus.Fields{
	"prefix": "fetcher",
})

func Start() {
	for {
		NewRunner().run()

		time.Sleep(time.Duration(config.C.FetchInterval) * time.Second)
	}
}

type Runner struct {
	client *Client
}

func NewRunner() *Runner {
	return &Runner{
		client: NewClient(),
	}
}

func (r *Runner) run() {
	start := time.Now()

	// Create context with timeout for all RPC calls
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.C.FetchInterval-1)*time.Second)
	defer cancel()

	// Fetch all data
	blockChainInfo := r.getBlockchainInfo(ctx)
	memPoolInfo := r.getMempoolInfo(ctx)
	memoryInfo := r.getMemoryInfo(ctx)
	indexInfo := r.getIndexInfo(ctx)
	networkInfo := r.getNetworkInfo(ctx)

	feeRate2 := r.getSmartFee(ctx, feeEstimation2Blocks)
	feeRate5 := r.getSmartFee(ctx, feeEstimation5Blocks)
	feeRate20 := r.getSmartFee(ctx, feeEstimation20Blocks)

	hasRateLatest := r.getNetworkHashrate(ctx, hashRateLatest)
	hashRate1 := r.getNetworkHashrate(ctx, hashRate1Block)
	hasthRate120 := r.getNetworkHashrate(ctx, hashRate120Blocks)

	netTotals := r.getNetTotals(ctx)

	if util.AnyNil(blockChainInfo, memPoolInfo, memoryInfo, indexInfo, networkInfo,
		feeRate2, feeRate5, feeRate20, hasRateLatest, hashRate1, hasthRate120, netTotals) {
		log.Error("Failed to fetch data")
		return
	}

	// Update metrics
	r.updateMetrics(blockChainInfo, memPoolInfo, memoryInfo, indexInfo, networkInfo, netTotals,
		feeRate2, feeRate5, feeRate20, hasRateLatest, hashRate1, hasthRate120)

	// Internal
	prometheus.ScrapeTime.Set(float64(time.Since(start).Milliseconds()))
}

// updateMetrics updates all Prometheus metrics with fetched data
func (r *Runner) updateMetrics(blockChainInfo *BlockchainInfo, memPoolInfo *MempoolInfo,
	memoryInfo *MemoryInfo, indexInfo *IndexInfo, networkInfo *NetworkInfo, netTotals *NetTotals,
	feeRate2, feeRate5, feeRate20 *SmartFee, hasRateLatest, hashRate1, hasthRate120 float64) {
	// Blockchain
	prometheus.BlockchainBlocks.Set(float64(blockChainInfo.Blocks))
	prometheus.BlockchainHeaders.Set(float64(blockChainInfo.Headers))
	prometheus.BlockchainVerificationProgress.Set(blockChainInfo.VerificationProgress)
	prometheus.BlockchainSizeOnDisk.Set(float64(blockChainInfo.SizeOnDisk))

	// Mempool
	prometheus.MempoolUsage.Set(float64(memPoolInfo.Usage))
	prometheus.MempoolMax.Set(float64(memPoolInfo.MaxMempool))
	prometheus.MempoolTransactionCount.Set(float64(memPoolInfo.Size))

	// Memory
	prometheus.MemoryUsed.Set(float64(memoryInfo.Locked.Used))
	prometheus.MemoryFree.Set(float64(memoryInfo.Locked.Free))
	prometheus.MemoryTotal.Set(float64(memoryInfo.Locked.Total))
	prometheus.MemoryLocked.Set(float64(memoryInfo.Locked.Locked))
	prometheus.ChunksUsed.Set(float64(memoryInfo.Locked.ChunksUsed))
	prometheus.ChunksFree.Set(float64(memoryInfo.Locked.ChunksFree))

	// TxIndex
	prometheus.TxIndexSynced.Set(float64(util.BoolToFloat64(indexInfo.TxIndex.Synced)))
	prometheus.TxIndexBestHeight.Set(float64(indexInfo.TxIndex.BestBlockHeight))

	// Network
	prometheus.TotalConnections.Set(float64(networkInfo.TotalConnections))
	prometheus.ConnectionsIn.Set(float64(networkInfo.ConnectionsIn))
	prometheus.ConnectionsOut.Set(float64(networkInfo.TotalConnections - networkInfo.ConnectionsIn))
	prometheus.TotalBytesRecv.Set(float64(netTotals.TotalBytesRecv))
	prometheus.TotalBytesSent.Set(float64(netTotals.TotalBytesSent))

	// SmartFee
	prometheus.SmartFee.With(goprom.Labels{"blocks": "2"}).Set(util.ConvertBTCkBToSatVb(feeRate2.Feerate))
	prometheus.SmartFee.With(goprom.Labels{"blocks": "5"}).Set(util.ConvertBTCkBToSatVb(feeRate5.Feerate))
	prometheus.SmartFee.With(goprom.Labels{"blocks": "20"}).Set(util.ConvertBTCkBToSatVb(feeRate20.Feerate))

	// Mining
	prometheus.MiningHashrate.With(goprom.Labels{"blocks": "-1"}).Set(hasRateLatest)
	prometheus.MiningHashrate.With(goprom.Labels{"blocks": "1"}).Set(hashRate1)
	prometheus.MiningHashrate.With(goprom.Labels{"blocks": "120"}).Set(hasthRate120)
}

func (r *Runner) getBlockchainInfo(ctx context.Context) *BlockchainInfo {
	var info *BlockchainInfo
	err := r.client.RpcClient.CallFor(ctx, &info, "getblockchaininfo")
	if err != nil {
		log.WithError(err).Error("Failed to call RPC")
		return nil
	}

	return info
}

func (r *Runner) getMempoolInfo(ctx context.Context) *MempoolInfo {
	var info *MempoolInfo
	err := r.client.RpcClient.CallFor(ctx, &info, "getmempoolinfo")
	if err != nil {
		log.WithError(err).Error("Failed to call RPC")
		return nil
	}

	return info
}

func (r *Runner) getMemoryInfo(ctx context.Context) *MemoryInfo {
	var info *MemoryInfo
	err := r.client.RpcClient.CallFor(ctx, &info, "getmemoryinfo")
	if err != nil {
		log.WithError(err).Error("Failed to call RPC")
		return nil
	}

	return info
}

func (r *Runner) getIndexInfo(ctx context.Context) *IndexInfo {
	var info *IndexInfo
	err := r.client.RpcClient.CallFor(ctx, &info, "getindexinfo")
	if err != nil {
		log.WithError(err).Error("Failed to call RPC")
		return nil
	}

	return info
}

func (r *Runner) getNetworkInfo(ctx context.Context) *NetworkInfo {
	var info *NetworkInfo
	err := r.client.RpcClient.CallFor(ctx, &info, "getnetworkinfo")
	if err != nil {
		log.WithError(err).Error("Failed to call RPC")
		return nil
	}

	return info
}

func (r *Runner) getSmartFee(ctx context.Context, blocks int) *SmartFee {
	var info *SmartFee
	err := r.client.RpcClient.CallFor(ctx, &info, "estimatesmartfee", blocks)
	if err != nil {
		log.WithError(err).Error("Failed to call RPC")
		return nil
	}

	return info
}

func (r *Runner) getNetworkHashrate(ctx context.Context, blocks int) float64 {
	var info float64
	err := r.client.RpcClient.CallFor(ctx, &info, "getnetworkhashps", blocks)
	if err != nil {
		log.WithError(err).Error("Failed to call RPC")
		return 0
	}

	return info
}

func (r *Runner) getNetTotals(ctx context.Context) *NetTotals {
	var info *NetTotals
	err := r.client.RpcClient.CallFor(ctx, &info, "getnettotals")
	if err != nil {
		log.WithError(err).Error("Failed to call RPC")
		return nil
	}

	return info
}
