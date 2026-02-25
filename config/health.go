package config

import (
	"context"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Health monitor configuration constants
const (
	defaultHealthCheckInterval = 30 * time.Second // Health check interval
	healthCheckTimeout         = 10 * time.Second // Individual check timeout
	httpClientTimeout          = 5 * time.Second  // HTTP client timeout
	tcpDialTimeout             = 5 * time.Second  // TCP dial timeout
)

// HealthStatus represents the health status of a configuration component
type HealthStatus int

const (
	StatusUnknown HealthStatus = iota
	StatusHealthy
	StatusUnhealthy
	StatusDegraded
)

func (s HealthStatus) String() string {
	switch s {
	case StatusUnknown:
		return "unknown"
	case StatusHealthy:
		return "healthy"
	case StatusUnhealthy:
		return "unhealthy"
	case StatusDegraded:
		return "degraded"
	default:
		return "unknown"
	}
}

// HealthCheck represents a single health check
type HealthCheck struct {
	Name        string
	Status      HealthStatus
	Message     string
	LastChecked time.Time
	Duration    time.Duration
	Error       error
}

// ConfigHealthMonitor monitors the health of configuration-dependent services
type ConfigHealthMonitor struct {
	config   *Config
	logger   *logrus.Entry
	checks   map[string]*HealthCheck
	mutex    sync.RWMutex
	interval time.Duration
	stopChan chan struct{}
	running  bool
}

// NewConfigHealthMonitor creates a new configuration health monitor
func NewConfigHealthMonitor(cfg *Config, logger *logrus.Entry) *ConfigHealthMonitor {
	return &ConfigHealthMonitor{
		config:   cfg,
		logger:   logger,
		checks:   make(map[string]*HealthCheck),
		interval: defaultHealthCheckInterval,
		stopChan: make(chan struct{}),
	}
}

// Start begins health monitoring
func (m *ConfigHealthMonitor) Start(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.running {
		return fmt.Errorf("health monitor is already running")
	}

	m.running = true
	m.logger.Info("Starting configuration health monitor")

	// Initialize health checks
	m.initializeChecks()

	// Start monitoring goroutine
	go m.monitorLoop(ctx)

	return nil
}

// Stop stops health monitoring
func (m *ConfigHealthMonitor) Stop() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.running {
		return
	}

	m.logger.Info("Stopping configuration health monitor")
	close(m.stopChan)
	m.running = false
}

// GetHealthStatus returns the current health status
func (m *ConfigHealthMonitor) GetHealthStatus() map[string]*HealthCheck {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Return a copy to prevent concurrent access issues
	result := make(map[string]*HealthCheck)
	for name, check := range m.checks {
		checkCopy := *check
		result[name] = &checkCopy
	}

	return result
}

// IsHealthy returns overall health status
func (m *ConfigHealthMonitor) IsHealthy() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for _, check := range m.checks {
		if check.Status == StatusUnhealthy {
			return false
		}
	}

	return true
}

// initializeChecks sets up health checks based on configuration
func (m *ConfigHealthMonitor) initializeChecks() {
	// RPC connectivity check
	m.checks["rpc_connectivity"] = &HealthCheck{
		Name:   "RPC Connectivity",
		Status: StatusUnknown,
	}

	// ZMQ connectivity check (if enabled)
	if m.config.ZMQ.Enabled {
		m.checks["zmq_connectivity"] = &HealthCheck{
			Name:   "ZMQ Connectivity",
			Status: StatusUnknown,
		}
	}

	// TLS certificate check (if enabled)
	if m.config.Security.TLSEnabled {
		m.checks["tls_certificate"] = &HealthCheck{
			Name:   "TLS Certificate",
			Status: StatusUnknown,
		}
	}

	// Port availability check
	m.checks["port_availability"] = &HealthCheck{
		Name:   "Port Availability",
		Status: StatusUnknown,
	}

	// Configuration validation check
	m.checks["config_validation"] = &HealthCheck{
		Name:   "Configuration Validation",
		Status: StatusUnknown,
	}
}

// monitorLoop runs the health monitoring loop
func (m *ConfigHealthMonitor) monitorLoop(ctx context.Context) {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	// Run initial checks
	m.runHealthChecks(ctx)

	for {
		select {
		case <-ctx.Done():
			m.logger.Info("Context canceled, stopping health monitor")
			return
		case <-m.stopChan:
			m.logger.Info("Stop signal received, stopping health monitor")
			return
		case <-ticker.C:
			m.runHealthChecks(ctx)
		}
	}
}

// runHealthChecks executes all health checks
func (m *ConfigHealthMonitor) runHealthChecks(ctx context.Context) {
	m.logger.Debug("Running configuration health checks")

	// Run checks concurrently
	var wg sync.WaitGroup

	for checkName := range m.checks {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			m.runSingleCheck(ctx, name)
		}(checkName)
	}

	wg.Wait()

	// Log overall status
	if m.IsHealthy() {
		m.logger.Debug("All configuration health checks passed")
	} else {
		m.logger.Warn("Some configuration health checks failed")
	}
}

// runSingleCheck executes a single health check
func (m *ConfigHealthMonitor) runSingleCheck(ctx context.Context, checkName string) {
	start := time.Now()
	var status HealthStatus
	var message string
	var err error

	// Create a timeout context for the check
	checkCtx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
	defer cancel()

	switch checkName {
	case "rpc_connectivity":
		status, message, err = m.checkRPCConnectivity(checkCtx)
	case "zmq_connectivity":
		status, message, err = m.checkZMQConnectivity(checkCtx)
	case "tls_certificate":
		status, message, err = m.checkTLSCertificate(checkCtx)
	case "port_availability":
		status, message, err = m.checkPortAvailability(checkCtx)
	case "config_validation":
		status, message, err = m.checkConfigValidation(checkCtx)
	default:
		status = StatusUnknown
		message = "Unknown check type"
		err = fmt.Errorf("unknown check: %s", checkName)
	}

	duration := time.Since(start)

	// Update check result
	m.mutex.Lock()
	if check, exists := m.checks[checkName]; exists {
		check.Status = status
		check.Message = message
		check.LastChecked = time.Now()
		check.Duration = duration
		check.Error = err
	}
	m.mutex.Unlock()

	// Log check result
	logEntry := m.logger.WithFields(logrus.Fields{
		"check":    checkName,
		"status":   status.String(),
		"duration": duration,
	})

	m.logHealthCheckResult(logEntry, err, status)
}

// logHealthCheckResult logs the health check result with appropriate log level
func (m *ConfigHealthMonitor) logHealthCheckResult(logEntry *logrus.Entry, err error, status HealthStatus) {
	switch {
	case err != nil:
		logEntry.WithError(err).Warn("Health check failed")
	case status == StatusDegraded:
		logEntry.Warn("Health check degraded")
	default:
		logEntry.Debug("Health check completed")
	}
}

// checkRPCConnectivity checks if RPC endpoint is reachable
func (m *ConfigHealthMonitor) checkRPCConnectivity(ctx context.Context) (HealthStatus, string, error) {
	// Simple HTTP connectivity check
	client := &http.Client{
		Timeout: httpClientTimeout,
	}

	req, err := http.NewRequestWithContext(ctx, "POST", m.config.RPC.Address, http.NoBody)
	if err != nil {
		return StatusUnhealthy, "Failed to create request", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return StatusUnhealthy, "Connection failed", err
	}
	defer func() { _ = resp.Body.Close() }()

	// Even if we get an error response, the connectivity is working
	if resp.StatusCode == http.StatusUnauthorized {
		return StatusHealthy, "RPC endpoint reachable (authentication required)", nil
	}

	return StatusHealthy, "RPC endpoint reachable", nil
}

// checkZMQConnectivity checks if ZMQ endpoint is reachable
func (m *ConfigHealthMonitor) checkZMQConnectivity(ctx context.Context) (HealthStatus, string, error) {
	if !m.config.ZMQ.Enabled {
		return StatusHealthy, "ZMQ disabled", nil
	}

	// Extract host and port from ZMQ address (tcp://host:port)
	address := m.config.ZMQ.Address
	if len(address) < 6 || address[:6] != "tcp://" {
		return StatusUnhealthy, "Invalid ZMQ address format", fmt.Errorf("invalid address: %s", address)
	}

	hostPort := address[6:] // Remove "tcp://" prefix

	// Try to establish a TCP connection
	dialer := &net.Dialer{
		Timeout: tcpDialTimeout,
	}

	conn, err := dialer.DialContext(ctx, "tcp", hostPort)
	if err != nil {
		return StatusUnhealthy, "ZMQ connection failed", err
	}
	defer func() { _ = conn.Close() }()

	return StatusHealthy, "ZMQ endpoint reachable", nil
}

// checkTLSCertificate checks TLS certificate validity
func (m *ConfigHealthMonitor) checkTLSCertificate(ctx context.Context) (HealthStatus, string, error) {
	_ = ctx // TODO: Use context for timeout/cancellation in future TLS validation
	if !m.config.Security.TLSEnabled {
		return StatusHealthy, "TLS disabled", nil
	}

	// For now, just check if certificate files exist and are readable
	// A more comprehensive check would validate the certificate chain, expiration, etc.

	if m.config.Security.TLSCertFile == "" {
		return StatusUnhealthy, "TLS certificate file not specified", fmt.Errorf("cert file empty")
	}

	if m.config.Security.TLSKeyFile == "" {
		return StatusUnhealthy, "TLS key file not specified", fmt.Errorf("key file empty")
	}

	// Validate certificate file existence and readability
	if err := validateCertificateFile(m.config.Security.TLSCertFile); err != nil {
		return StatusUnhealthy, "TLS certificate validation failed", err
	}

	// Validate key file existence and readability
	if err := validateKeyFile(m.config.Security.TLSKeyFile); err != nil {
		return StatusUnhealthy, "TLS key validation failed", err
	}

	// Validate certificate and key compatibility
	if err := validateCertKeyPair(m.config.Security.TLSCertFile, m.config.Security.TLSKeyFile); err != nil {
		return StatusUnhealthy, "TLS certificate/key pair validation failed", err
	}

	return StatusHealthy, "TLS configuration validated successfully", nil
}

// checkPortAvailability checks if the metrics port is available
func (m *ConfigHealthMonitor) checkPortAvailability(ctx context.Context) (HealthStatus, string, error) {
	address := fmt.Sprintf(":%d", m.config.Metrics.Port)

	// Try to bind to the port
	lc := &net.ListenConfig{}
	listener, err := lc.Listen(ctx, "tcp", address)
	if err != nil {
		return StatusUnhealthy, "Port not available", err
	}
	defer func() { _ = listener.Close() }()

	return StatusHealthy, "Port available", nil
}

// checkConfigValidation runs configuration validation
func (m *ConfigHealthMonitor) checkConfigValidation(ctx context.Context) (HealthStatus, string, error) {
	_ = ctx                         // TODO: Use context for timeout/cancellation in future validation
	validator := NewValidator(true) // Use strict mode for health checks
	result := validator.ValidateConfig(m.config)

	if !result.Valid {
		return StatusUnhealthy, "Configuration validation failed",
			fmt.Errorf("validation errors: %d", len(result.Errors))
	}

	if len(result.Warnings) > 0 {
		return StatusDegraded, fmt.Sprintf("Configuration valid with %d warnings", len(result.Warnings)), nil
	}

	return StatusHealthy, "Configuration validation passed", nil
}

// GetHealthSummary returns a summary of health status
func (m *ConfigHealthMonitor) GetHealthSummary() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	summary := map[string]interface{}{
		"overall_healthy": m.IsHealthy(),
		"last_check":      time.Now(),
		"checks":          make(map[string]interface{}),
	}

	for name, check := range m.checks {
		summary["checks"].(map[string]interface{})[name] = map[string]interface{}{
			"status":       check.Status.String(),
			"message":      check.Message,
			"last_checked": check.LastChecked,
			"duration_ms":  check.Duration.Milliseconds(),
		}
	}

	return summary
}

// Certificate validation helper functions

// validateCertificateFile validates that a certificate file exists, is readable, and contains valid certificate data
func validateCertificateFile(certFile string) error {
	// Validate file path for security
	if certFile == "" || strings.Contains(certFile, "..") {
		return fmt.Errorf("invalid certificate file path")
	}

	// Check file existence
	if _, err := os.Stat(certFile); err != nil {
		return fmt.Errorf("certificate file not accessible: %w", err)
	}

	// Read and parse certificate
	certPEM, err := os.ReadFile(certFile) // #nosec G304 - file path validated above
	if err != nil {
		return fmt.Errorf("failed to read certificate file: %w", err)
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return fmt.Errorf("failed to decode PEM block containing certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Check certificate expiration
	now := time.Now()
	if now.Before(cert.NotBefore) {
		return fmt.Errorf("certificate is not yet valid (valid from %s)", cert.NotBefore.Format(time.RFC3339))
	}
	if now.After(cert.NotAfter) {
		return fmt.Errorf("certificate has expired (expired on %s)", cert.NotAfter.Format(time.RFC3339))
	}

	// Warn if certificate expires soon (within 30 days)
	if now.Add(30 * 24 * time.Hour).After(cert.NotAfter) {
		// Log warning but don't fail the health check
		log.WithField("expiry", cert.NotAfter.Format(time.RFC3339)).
			Warning("TLS certificate expires within 30 days")
	}

	return nil
}

// validateKeyFile validates that a private key file exists, is readable, and contains valid key data
func validateKeyFile(keyFile string) error {
	// Validate file path for security
	if keyFile == "" || strings.Contains(keyFile, "..") {
		return fmt.Errorf("invalid key file path")
	}

	// Check file existence
	if _, err := os.Stat(keyFile); err != nil {
		return fmt.Errorf("key file not accessible: %w", err)
	}

	// Read key file
	keyPEM, err := os.ReadFile(keyFile) // #nosec G304 - file path validated above
	if err != nil {
		return fmt.Errorf("failed to read key file: %w", err)
	}

	block, _ := pem.Decode(keyPEM)
	if block == nil {
		return fmt.Errorf("failed to decode PEM block containing private key")
	}

	// Try to parse as different key types
	_, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		_, err = x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			_, err = x509.ParseECPrivateKey(block.Bytes)
			if err != nil {
				return fmt.Errorf("failed to parse private key: key type not supported")
			}
		}
	}

	return nil
}

// validateCertKeyPair validates that a certificate and private key are compatible
func validateCertKeyPair(certFile, keyFile string) error {
	// Load certificate and key pair
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return fmt.Errorf("failed to load certificate/key pair: %w", err)
	}

	// Parse the certificate to get public key
	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return fmt.Errorf("failed to parse certificate for validation: %w", err)
	}

	// Verify the private key matches the certificate's public key
	switch pub := x509Cert.PublicKey.(type) {
	case *rsa.PublicKey:
		priv, ok := cert.PrivateKey.(*rsa.PrivateKey)
		if !ok {
			return fmt.Errorf("certificate has RSA public key but private key is not RSA")
		}
		if pub.N.Cmp(priv.N) != 0 {
			return fmt.Errorf("RSA private key does not match certificate public key")
		}
	case *ecdsa.PublicKey:
		priv, ok := cert.PrivateKey.(*ecdsa.PrivateKey)
		if !ok {
			return fmt.Errorf("certificate has ECDSA public key but private key is not ECDSA")
		}
		if pub.X.Cmp(priv.X) != 0 || pub.Y.Cmp(priv.Y) != 0 {
			return fmt.Errorf("ECDSA private key does not match certificate public key")
		}
	default:
		return fmt.Errorf("unsupported public key type: %T", pub)
	}

	return nil
}
