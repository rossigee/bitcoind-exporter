package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/rossigee/bitcoind-exporter/config"
	"github.com/rossigee/bitcoind-exporter/fetcher"
	"github.com/rossigee/bitcoind-exporter/prometheus"
	"github.com/rossigee/bitcoind-exporter/zmq"
	log "github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

// Logger configuration constants
const (
	logFormatterSpacePadding = 45 // Space padding for log formatter
	healthCheckTimeout       = 5 * time.Second
)

// setupLogging configures the logging system
func setupLogging() {
	log.SetFormatter(&prefixed.TextFormatter{
		TimestampFormat:  "2006/01/02 - 15:04:05",
		FullTimestamp:    true,
		QuoteEmptyFields: true,
		SpacePadding:     logFormatterSpacePadding,
	})

	log.SetReportCaller(true)

	level, err := log.ParseLevel(config.C.LogLevel)
	if err != nil {
		log.WithError(err).Fatal("Invalid log level")
	}

	log.SetLevel(level)
}

func main() {
	healthCheck := flag.Bool("health-check", false,
		"Perform a one-shot bitcoind RPC connectivity check and exit (used by container healthchecks)")
	flag.Parse()

	config.InitializeConfig()
	setupLogging()
	log.WithFields(log.Fields{
		"commit":  commit,
		"date":    date,
		"runtime": runtime.Version(),
		"arch":    runtime.GOARCH,
	}).Infof("Bitcoind Exporter ₿ %s", version)

	if *healthCheck {
		os.Exit(runHealthCheck())
	}

	// Graceful shutdown on SIGINT/SIGTERM: all servers observe ctx and stop
	// cleanly, giving in-flight requests a chance to complete.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go prometheus.StartSecure(ctx)
	go zmq.Start(ctx)

	fetcher.StartResilient(ctx)
}

// runHealthCheck performs a lightweight RPC connectivity check and returns the
// process exit code (0 = healthy). It intentionally does not start the metrics
// or ZMQ servers.
func runHealthCheck() int {
	client := fetcher.NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), healthCheckTimeout)
	defer cancel()

	var blockCount int64
	if err := client.RpcClient.CallFor(ctx, &blockCount, "getblockcount"); err != nil {
		log.WithError(err).Error("Health check failed: bitcoind RPC unavailable")
		return 1
	}

	log.WithField("block_count", blockCount).Info("Health check passed")
	return 0
}
