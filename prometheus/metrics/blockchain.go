package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	helpBlocks  = "The number of blocks in the blockchain"
	helpHeaders = "The number of headers in the blockchain"
)

var (
	BlockchainBlocks = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "bitcoind_blockchain_blocks",
		Help: helpBlocks,
	})

	BlockchainHeaders = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "bitcoind_blockchain_headers",
		Help: helpHeaders,
	})

	BlockchainVerificationProgress = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "bitcoind_blockchain_verification_progress",
		Help: "The verification progress of the blockchain",
	})

	BlockchainSizeOnDisk = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "bitcoind_blockchain_size_on_disk",
		Help: "The size of the blockchain on disk",
	})
)
