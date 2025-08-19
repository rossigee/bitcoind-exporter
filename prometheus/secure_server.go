package prometheus

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/Primexz/bitcoind-exporter/config"
	"github.com/Primexz/bitcoind-exporter/fetcher"
	"github.com/Primexz/bitcoind-exporter/security"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

// Configuration constants
const (
	readinessTimeoutSeconds = 5 // Timeout for readiness check RPC call
)

var (
	secureLog = logrus.WithFields(logrus.Fields{
		"prefix": "secure_prometheus",
	})
)

// StartSecure starts the Prometheus metrics server with security features
func StartSecure() {
	port := strconv.Itoa(config.C.MetricPort)

	secureLog.WithField("port", port).Info("Starting secure Prometheus metrics server")

	// Create metrics handler
	metricsHandler := promhttp.Handler()

	// Create security configuration
	securityConfig := &security.SecurityConfig{
		TLS: &security.TLSConfig{
			Enabled:    config.Security.TLSEnabled,
			CertFile:   config.Security.TLSCertFile,
			KeyFile:    config.Security.TLSKeyFile,
			MinVersion: config.Security.TLSMinVersion,
		},
		Auth: &security.AuthConfig{
			Enabled:  config.Security.AuthEnabled,
			Username: config.Security.AuthUsername,
			Password: config.Security.AuthPassword,
		},
		RateLimit: &security.RateLimitConfig{
			Enabled:    config.Security.RateLimitEnabled,
			Requests:   config.Security.RateLimitRequests,
			WindowSize: config.Security.RateLimitWindow,
			BlockTime:  config.Security.RateLimitBlockTime,
		},
	}

	// Create secure handler with middleware
	secureHandler := security.CreateSecureHandler(metricsHandler, securityConfig)

	// Create route handler with health checks
	mux := http.NewServeMux()
	mux.Handle("/metrics", secureHandler)
	mux.HandleFunc("/health", healthCheckHandler)
	mux.HandleFunc("/ready", readinessCheckHandler)

	// Create secure server
	secureServer := security.NewSecureServer(":"+port, mux, securityConfig.TLS)

	// Start server
	if err := secureServer.ListenAndServe(securityConfig.TLS); err != nil {
		secureLog.WithError(err).Error("Failed to start secure Prometheus metrics server")
	}
}

// healthCheckHandler provides a health check endpoint
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"healthy","component":"bitcoind-exporter"}`))
}

// readinessCheckHandler provides a readiness check endpoint
func readinessCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Check Bitcoin RPC connectivity
	if !checkBitcoinRPCReadiness() {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"status":"not_ready","component":"bitcoind-exporter","reason":"bitcoin_rpc_unavailable"}`))
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ready","component":"bitcoind-exporter"}`))
}

// checkBitcoinRPCReadiness verifies Bitcoin RPC connectivity
func checkBitcoinRPCReadiness() bool {
	client := fetcher.NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), readinessTimeoutSeconds*time.Second)
	defer cancel()

	// Try a simple RPC call to verify connectivity
	var blockCount int64
	err := client.RpcClient.CallFor(ctx, &blockCount, "getblockcount")
	return err == nil
}

// IPFilterMiddleware provides IP-based access control
func IPFilterMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := getClientIP(r)

		if !config.IsIPAllowed(clientIP) {
			secureLog.WithField("client_ip", clientIP).Warn("Access denied for IP address")
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// getClientIP extracts the real client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (for proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}

	// Check X-Real-IP header (for reverse proxies)
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}
