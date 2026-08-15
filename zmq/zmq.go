package zmq

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rossigee/bitcoind-exporter/config"
	prometheus "github.com/rossigee/bitcoind-exporter/prometheus/metrics"
	"github.com/go-zeromq/zmq4"
	"github.com/sirupsen/logrus"
)

const (
	zmqMinFrames      = 2               // ZMQ multipart minimum: topic + payload
	zmqReconnectDelay = 5 * time.Second // Delay between reconnect attempts
	zmqRateWindow     = time.Minute     // Window over which the tx rate is computed
)

var (
	log = logrus.WithFields(logrus.Fields{
		"prefix": "zmq",
	})
)

// Start listens for ZMQ rawtx notifications and never exits the process on
// connection problems: a dropped ZMQ connection (e.g. bitcoind restart) is
// retried with a delay instead of terminating the exporter. It stops cleanly
// when ctx is canceled.
func Start(ctx context.Context) {
	address := config.C.ZmqAddress
	if strings.TrimSpace(address) == "" {
		log.Debug("Zmq address not set, skipping zmq listener")
		return
	}

	// Accept both "host:port" (legacy) and "tcp://host:port" forms.
	if !strings.HasPrefix(address, "tcp://") {
		address = "tcp://" + address
	}

	for {
		if err := runListener(ctx, address); err != nil {
			if ctx.Err() != nil {
				log.Info("Shutting down zmq listener")
				return
			}
			log.WithError(err).Error("ZMQ listener failed, reconnecting")
			select {
			case <-ctx.Done():
				log.Info("Shutting down zmq listener")
				return
			case <-time.After(zmqReconnectDelay):
			}
			continue
		}
		return
	}
}

func runListener(ctx context.Context, address string) error {
	sub := zmq4.NewSub(ctx)
	defer func() { _ = sub.Close() }()

	if err := sub.Dial(address); err != nil {
		return fmt.Errorf("could not dial %s: %w", address, err)
	}

	if err := sub.SetOption(zmq4.OptionSubscribe, "rawtx"); err != nil {
		return fmt.Errorf("could not set option: %w", err)
	}

	log.WithField("address", address).Info("Listening for zmq messages")

	var txCount int64
	windowStart := time.Now()

	for {
		// Read envelope
		msg, err := sub.Recv()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("could not receive: %w", err)
		}

		// ZMQ multipart: [topic, payload, sequence]. Guard against malformed messages.
		if len(msg.Frames) < zmqMinFrames {
			log.WithField("frames", len(msg.Frames)).Warn("Received malformed zmq message, skipping")
			continue
		}

		transaction := string(msg.Frames[1])
		log.WithField("transaction", transaction).Debug("Received transaction")

		txCount++
		now := time.Now()
		if elapsed := now.Sub(windowStart); elapsed >= zmqRateWindow {
			// Report the actual per-second rate over the elapsed window instead
			// of a monotonically increasing gauge value.
			prometheus.TransactionsPerSecond.Set(float64(txCount) / elapsed.Seconds())
			txCount = 0
			windowStart = now
		}
	}
}
