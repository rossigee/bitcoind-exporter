package config

import (
	"context"
	"fmt"
	"net"
	"net/http"
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
			m.logger.Info("Context cancelled, stopping health monitor")
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

	req, err := http.NewRequestWithContext(ctx, "POST", m.config.RPC.Address, nil)
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

	// TODO: Add actual certificate validation
	// This would include checking file existence, readability, expiration, etc.

	return StatusHealthy, "TLS configuration appears valid", nil
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
