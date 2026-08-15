package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	helpConnections    = "The total number of peer connections"
	helpConnectionsIn  = "The number of inbound peer connections"
	helpConnectionsOut = "The number of outbound peer connections"
)

var (
	TotalConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "bitcoind_total_connections",
		Help: helpConnections,
	})

	ConnectionsIn = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "bitcoind_connections_in",
		Help: helpConnectionsIn,
	})

	ConnectionsOut = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "bitcoind_connections_out",
		Help: helpConnectionsOut,
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
