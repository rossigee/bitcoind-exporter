package main

import (
	"runtime"

	"github.com/Primexz/bitcoind-exporter/config"
	"github.com/Primexz/bitcoind-exporter/fetcher"
	"github.com/Primexz/bitcoind-exporter/prometheus"
	"github.com/Primexz/bitcoind-exporter/zmq"
	log "github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

// Logger configuration constants
const (
	logFormatterSpacePadding = 45 // Space padding for log formatter
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
	config.InitializeConfig()
	setupLogging()
	log.WithFields(log.Fields{
		"commit":  commit,
		"runtime": runtime.Version(),
		"arch":    runtime.GOARCH,
	}).Infof("Bitcoind Exporter ₿ %s", version)

	go prometheus.Start()
	go zmq.Start()

	fetcher.Start()
}
