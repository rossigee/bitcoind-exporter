package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	TotalConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "bitcoind_total_connections",
		Help: helpBlocks,
	})

	ConnectionsIn = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "bitcoind_connections_in",
		Help: helpHeaders,
	})

	ConnectionsOut = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "bitcoind_connections_out",
		Help: helpHeaders,
	})

	TotalBytesRecv = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "bitcoind_total_bytes_recv",
		Help: "The number of bytes received",
	})

	TotalBytesSent = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "bitcoind_total_bytes_sent",
		Help: "The number of bytes sent",
	})
)
