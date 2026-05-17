package zmq

import (
	"context"

	"github.com/rossigee/bitcoind-exporter/config"
	prometheus "github.com/rossigee/bitcoind-exporter/prometheus/metrics"
	"github.com/go-zeromq/zmq4"
	"github.com/sirupsen/logrus"
)

const (
	zmqMinFrames = 2 // ZMQ multipart minimum: topic + payload
)

var (
	log = logrus.WithFields(logrus.Fields{
		"prefix": "zmq",
	})
)

func Start() {
	address := config.C.ZmqAddress
	if address == "" {
		log.Debug("Zmq address not set, skipping zmq listener")
		return
	}

	sub := zmq4.NewSub(context.Background())
	defer func() { _ = sub.Close() }()

	err := sub.Dial("tcp://" + address)
	if err != nil {
		log.WithError(err).Fatal("could not dial")
	}

	err = sub.SetOption(zmq4.OptionSubscribe, "rawtx")
	if err != nil {
		log.WithError(err).Fatal("could not set option")
	}

	log.WithField("address", address).Info("Listening for zmq messages")

	for {
		// Read envelope
		msg, err := sub.Recv()
		if err != nil {
			log.WithError(err).Fatal("could not receive")
		}

		// ZMQ multipart: [topic, payload, sequence]. Guard against malformed messages.
		if len(msg.Frames) < zmqMinFrames {
			log.WithField("frames", len(msg.Frames)).Warn("Received malformed zmq message, skipping")
			continue
		}

		transaction := string(msg.Frames[1])
		log.WithField("transaction", transaction).Debug("Received transaction")

		prometheus.TransactionsPerSecond.Inc()
	}
}
