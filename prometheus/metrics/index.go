package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	TxIndexSynced = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "bitcoind_txindex_synced",
		Help: helpBlocks,
	})

	TxIndexBestHeight = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "bitcoind_txindex_best_height",
		Help: helpHeaders,
	})
)
