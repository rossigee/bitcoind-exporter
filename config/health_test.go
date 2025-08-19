package config

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfigHealthMonitor(t *testing.T) {
	cfg := &Config{
		RPC: RPCConfig{
			Address: "http://localhost:8332",
			User:    "testuser",
			Pass:    "testpass",
		},
		Metrics: MetricsConfig{
			Port: 3000,
		},
	}

	logger := logrus.WithField("test", true)
	monitor := NewConfigHealthMonitor(cfg, logger)

	assert.NotNil(t, monitor)
	assert.Equal(t, cfg, monitor.config)
	assert.Equal(t, logger, monitor.logger)
	assert.NotNil(t, monitor.checks)
	assert.Equal(t, defaultHealthCheckInterval, monitor.interval)
	assert.False(t, monitor.running)
}

func TestConfigHealthMonitor_StartStop(t *testing.T) {
	cfg := &Config{
		RPC: RPCConfig{
			Address: "http://localhost:8332",
			User:    "testuser",
			Pass:    "testpass",
		},
		Metrics: MetricsConfig{
			Port: 3001,
		},
	}

	logger := logrus.WithField("test", true)
	monitor := NewConfigHealthMonitor(cfg, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start monitoring
	err := monitor.Start(ctx)
	require.NoError(t, err)
	assert.True(t, monitor.running)

	// Try to start again - should fail
	err = monitor.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	// Stop monitoring
	monitor.Stop()
	assert.False(t, monitor.running)

	// Stop again - should not panic
	assert.NotPanics(t, func() {
		monitor.Stop()
	})
}

func TestConfigHealthMonitor_InitializeChecks(t *testing.T) {
	tests := getInitializeChecksTestCases()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runInitializeChecksTest(t, tt.config, tt.expectedChecks)
		})
	}
}

func getInitializeChecksTestCases() []struct {
	name           string
	config         *Config
	expectedChecks []string
} {
	return []struct {
		name           string
		config         *Config
		expectedChecks []string
	}{
		{
			name:           "minimal configuration",
			config:         createMinimalHealthConfig(),
			expectedChecks: []string{"rpc_connectivity", "port_availability", "config_validation"},
		},
		{
			name:           "configuration with ZMQ enabled",
			config:         createZMQHealthConfig(),
			expectedChecks: []string{"rpc_connectivity", "zmq_connectivity", "port_availability", "config_validation"},
		},
		{
			name:           "configuration with TLS enabled",
			config:         createTLSHealthConfig(),
			expectedChecks: []string{"rpc_connectivity", "tls_certificate", "port_availability", "config_validation"},
		},
		{
			name:           "full configuration",
			config:         createFullHealthConfig(),
			expectedChecks: []string{
				"rpc_connectivity", "zmq_connectivity", "tls_certificate", 
				"port_availability", "config_validation",
			},
		},
	}
}

func createMinimalHealthConfig() *Config {
	return &Config{
		RPC: RPCConfig{
			Address: "http://localhost:8332",
		},
		Metrics: MetricsConfig{
			Port: 3000,
		},
		Security: SecurityConfig{
			TLSEnabled: false,
		},
		ZMQ: ZMQConfig{
			Enabled: false,
		},
	}
}

func createZMQHealthConfig() *Config {
	config := createMinimalHealthConfig()
	config.ZMQ = ZMQConfig{
		Enabled: true,
		Address: "tcp://localhost:28332",
	}
	return config
}

func createTLSHealthConfig() *Config {
	config := createMinimalHealthConfig()
	config.Security = SecurityConfig{
		TLSEnabled:  true,
		TLSCertFile: "/path/to/cert.pem",
		TLSKeyFile:  "/path/to/key.pem",
	}
	return config
}

func createFullHealthConfig() *Config {
	return &Config{
		RPC: RPCConfig{
			Address: "http://localhost:8332",
		},
		Metrics: MetricsConfig{
			Port: 3000,
		},
		Security: SecurityConfig{
			TLSEnabled:  true,
			TLSCertFile: "/path/to/cert.pem",
			TLSKeyFile:  "/path/to/key.pem",
		},
		ZMQ: ZMQConfig{
			Enabled: true,
			Address: "tcp://localhost:28332",
		},
	}
}

func runInitializeChecksTest(t *testing.T, config *Config, expectedChecks []string) {
	t.Helper()
	logger := logrus.WithField("test", true)
	monitor := NewConfigHealthMonitor(config, logger)
	
	// Call initializeChecks to populate the checks map
	monitor.initializeChecks()

	// Verify that all expected checks are initialized
	for _, expectedCheck := range expectedChecks {
		_, exists := monitor.checks[expectedCheck]
		assert.True(t, exists, "Expected check %s to be initialized", expectedCheck)
	}

	// Verify that only expected checks are present
	assert.Equal(t, len(expectedChecks), len(monitor.checks),
		"Expected %d checks but found %d", len(expectedChecks), len(monitor.checks))
}

func TestConfigHealthMonitor_GetHealthStatus(t *testing.T) {
	cfg := &Config{
		RPC: RPCConfig{
			Address: "http://localhost:8332",
		},
		Metrics: MetricsConfig{
			Port: 3000,
		},
	}

	logger := logrus.WithField("test", true)
	monitor := NewConfigHealthMonitor(cfg, logger)
	monitor.initializeChecks()

	// Update a check status
	monitor.checks["rpc_connectivity"].Status = StatusHealthy
	monitor.checks["rpc_connectivity"].Message = "Test message"
	monitor.checks["rpc_connectivity"].LastChecked = time.Now()

	status := monitor.GetHealthStatus()

	assert.NotNil(t, status)
	assert.Contains(t, status, "rpc_connectivity")
	
	rpcCheck := status["rpc_connectivity"]
	assert.Equal(t, StatusHealthy, rpcCheck.Status)
	assert.Equal(t, "Test message", rpcCheck.Message)
	assert.False(t, rpcCheck.LastChecked.IsZero())

	// Ensure we got a copy (modifying returned status shouldn't affect original)
	status["rpc_connectivity"].Status = StatusUnhealthy
	assert.Equal(t, StatusHealthy, monitor.checks["rpc_connectivity"].Status)
}

func TestConfigHealthMonitor_IsHealthy(t *testing.T) {
	cfg := &Config{
		RPC: RPCConfig{
			Address: "http://localhost:8332",
		},
		Metrics: MetricsConfig{
			Port: 3000,
		},
	}

	logger := logrus.WithField("test", true)
	monitor := NewConfigHealthMonitor(cfg, logger)
	monitor.initializeChecks()

	// All checks unknown - should be healthy
	assert.True(t, monitor.IsHealthy())

	// Set all checks to healthy
	for _, check := range monitor.checks {
		check.Status = StatusHealthy
	}
	assert.True(t, monitor.IsHealthy())

	// Set one check to degraded - should still be healthy
	for _, check := range monitor.checks {
		check.Status = StatusDegraded
		break
	}
	assert.True(t, monitor.IsHealthy())

	// Set one check to unhealthy - should be unhealthy
	for _, check := range monitor.checks {
		check.Status = StatusUnhealthy
		break
	}
	assert.False(t, monitor.IsHealthy())
}

func TestConfigHealthMonitor_GetHealthSummary(t *testing.T) {
	cfg := &Config{
		RPC: RPCConfig{
			Address: "http://localhost:8332",
		},
		Metrics: MetricsConfig{
			Port: 3000,
		},
	}

	logger := logrus.WithField("test", true)
	monitor := NewConfigHealthMonitor(cfg, logger)
	monitor.initializeChecks()

	// Update some check statuses
	now := time.Now()
	monitor.checks["rpc_connectivity"].Status = StatusHealthy
	monitor.checks["rpc_connectivity"].Message = "RPC is working"
	monitor.checks["rpc_connectivity"].LastChecked = now
	monitor.checks["rpc_connectivity"].Duration = 50 * time.Millisecond

	summary := monitor.GetHealthSummary()

	assert.NotNil(t, summary)
	assert.Contains(t, summary, "overall_healthy")
	assert.Contains(t, summary, "last_check")
	assert.Contains(t, summary, "checks")

	checks := summary["checks"].(map[string]interface{})
	assert.Contains(t, checks, "rpc_connectivity")

	rpcCheck := checks["rpc_connectivity"].(map[string]interface{})
	assert.Equal(t, "healthy", rpcCheck["status"])
	assert.Equal(t, "RPC is working", rpcCheck["message"])
	assert.Equal(t, int64(50), rpcCheck["duration_ms"])
}

func TestHealthStatus_String(t *testing.T) {
	tests := []struct {
		status   HealthStatus
		expected string
	}{
		{StatusUnknown, "unknown"},
		{StatusHealthy, "healthy"},
		{StatusUnhealthy, "unhealthy"},
		{StatusDegraded, "degraded"},
		{HealthStatus(999), "unknown"}, // Invalid status
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.status.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfigHealthMonitor_CheckRPCConnectivity(t *testing.T) {
	cfg := &Config{
		RPC: RPCConfig{
			Address: "http://httpbin.org/status/401", // Returns 401 Unauthorized
		},
	}

	logger := logrus.WithField("test", true)
	monitor := NewConfigHealthMonitor(cfg, logger)

	ctx := context.Background()
	status, message, err := monitor.checkRPCConnectivity(ctx)

	// Should be healthy even with 401 because connectivity is working
	assert.Equal(t, StatusHealthy, status)
	assert.Contains(t, message, "reachable")
	assert.NoError(t, err)
}

func TestConfigHealthMonitor_CheckZMQConnectivity(t *testing.T) {
	tests := getZMQConnectivityTestCases()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runZMQConnectivityTest(t, tt)
		})
	}
}

func getZMQConnectivityTestCases() []struct {
	name           string
	zmqEnabled     bool
	zmqAddress     string
	expectedStatus HealthStatus
} {
	return []struct {
		name           string
		zmqEnabled     bool
		zmqAddress     string
		expectedStatus HealthStatus
	}{
		{
			name:           "ZMQ disabled",
			zmqEnabled:     false,
			zmqAddress:     "",
			expectedStatus: StatusHealthy,
		},
		{
			name:           "invalid ZMQ address format",
			zmqEnabled:     true,
			zmqAddress:     "http://localhost:28332",
			expectedStatus: StatusUnhealthy,
		},
		{
			name:           "valid ZMQ address format but unreachable",
			zmqEnabled:     true,
			zmqAddress:     "tcp://localhost:99999",
			expectedStatus: StatusUnhealthy,
		},
	}
}

func runZMQConnectivityTest(t *testing.T, tt struct {
	name           string
	zmqEnabled     bool
	zmqAddress     string
	expectedStatus HealthStatus
}) {
	t.Helper()
	cfg := &Config{
		ZMQ: ZMQConfig{
			Enabled: tt.zmqEnabled,
			Address: tt.zmqAddress,
		},
	}

	logger := logrus.WithField("test", true)
	monitor := NewConfigHealthMonitor(cfg, logger)

	ctx := context.Background()
	status, _, _ := monitor.checkZMQConnectivity(ctx)

	assert.Equal(t, tt.expectedStatus, status)
}

func TestConfigHealthMonitor_CheckTLSCertificate(t *testing.T) {
	tests := getTLSCertificateTestCases()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runTLSCertificateTest(t, tt)
		})
	}
}

func getTLSCertificateTestCases() []struct {
	name           string
	tlsEnabled     bool
	setupFiles     func(t *testing.T) (string, string, func())
	expectedStatus HealthStatus
} {
	return []struct {
		name           string
		tlsEnabled     bool
		setupFiles     func(t *testing.T) (string, string, func())
		expectedStatus HealthStatus
	}{
		{
			name:           "TLS disabled",
			tlsEnabled:     false,
			expectedStatus: StatusHealthy,
		},
		{
			name:       "TLS enabled with valid certificates",
			tlsEnabled: true,
			setupFiles: createTLSTestFiles,
			expectedStatus: StatusHealthy,
		},
		{
			name:           "TLS enabled without cert file",
			tlsEnabled:     true,
			expectedStatus: StatusUnhealthy,
		},
	}
}

func createTLSTestFiles(t *testing.T) (string, string, func()) {
	t.Helper()
	certFile, keyFile := createTestCertificates(t)
	cleanup := func() {
		cleanupTestFile(certFile)
		cleanupTestFile(keyFile)
	}
	return certFile, keyFile, cleanup
}

func runTLSCertificateTest(t *testing.T, tt struct {
	name           string
	tlsEnabled     bool
	setupFiles     func(t *testing.T) (string, string, func())
	expectedStatus HealthStatus
}) {
	t.Helper()
	cfg := &Config{
		Security: SecurityConfig{
			TLSEnabled: tt.tlsEnabled,
		},
	}

	var cleanup func()
	if tt.setupFiles != nil {
		certFile, keyFile, cleanupFunc := tt.setupFiles(t)
		cfg.Security.TLSCertFile = certFile
		cfg.Security.TLSKeyFile = keyFile
		cleanup = cleanupFunc
	}
	defer func() {
		if cleanup != nil {
			cleanup()
		}
	}()

	logger := logrus.WithField("test", true)
	monitor := NewConfigHealthMonitor(cfg, logger)

	ctx := context.Background()
	status, _, _ := monitor.checkTLSCertificate(ctx)

	assert.Equal(t, tt.expectedStatus, status)
}

func TestConfigHealthMonitor_CheckPortAvailability(t *testing.T) {
	cfg := &Config{
		Metrics: MetricsConfig{
			Port: 0, // Use port 0 to get a random available port
		},
	}

	logger := logrus.WithField("test", true)
	monitor := NewConfigHealthMonitor(cfg, logger)

	ctx := context.Background()
	status, message, err := monitor.checkPortAvailability(ctx)

	// Port 0 should always be available (system assigns a free port)
	assert.Equal(t, StatusHealthy, status)
	assert.Contains(t, message, "available")
	assert.NoError(t, err)
}

func TestConfigHealthMonitor_CheckConfigValidation(t *testing.T) {
	tests := getConfigValidationTestCases()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runConfigValidationTest(t, tt)
		})
	}
}

func getConfigValidationTestCases() []struct {
	name           string
	config         *Config
	expectedStatus HealthStatus
} {
	return []struct {
		name           string
		config         *Config
		expectedStatus HealthStatus
	}{
		{
			name:           "valid configuration",
			config:         createValidDevConfig(),
			expectedStatus: StatusDegraded, // Will have security warnings in strict mode
		},
		{
			name:           "invalid configuration",
			config:         createInvalidConfig(),
			expectedStatus: StatusUnhealthy,
		},
	}
}

func createValidDevConfig() *Config {
	return &Config{
		RPC: RPCConfig{
			Address: "http://localhost:8332",
			User:    "testuser",
			Pass:    "testpass",
			Timeout: 30 * time.Second,
		},
		ZMQ: ZMQConfig{
			Enabled: false,
		},
		Metrics: MetricsConfig{
			Port:          3000,
			Path:          "/metrics",
			FetchInterval: 10 * time.Second,
		},
		App: AppConfig{
			LogLevel:    "info",
			Environment: "development", // Not production, so no security warnings
		},
		Security: SecurityConfig{}, // Minimal security in dev mode
	}
}

func createInvalidConfig() *Config {
	return &Config{
		RPC: RPCConfig{
			Address: "", // Invalid - empty address
		},
		Metrics: MetricsConfig{
			Port: 99999, // Invalid - port too high
		},
	}
}

func runConfigValidationTest(t *testing.T, tt struct {
	name           string
	config         *Config
	expectedStatus HealthStatus
}) {
	t.Helper()
	logger := logrus.WithField("test", true)
	monitor := NewConfigHealthMonitor(tt.config, logger)

	ctx := context.Background()
	status, _, _ := monitor.checkConfigValidation(ctx)

	assert.Equal(t, tt.expectedStatus, status)
}

func TestValidateCertificateFile(t *testing.T) {
	runCertFileValidationTests(t)
}

func runCertFileValidationTests(t *testing.T) {
	t.Helper()
	runFileValidationTest(t, "empty cert file path", "", "invalid certificate file path", 
		nil, validateCertificateFile)
	runFileValidationTest(t, "cert file with path traversal", "../../../etc/passwd", 
		"invalid certificate file path", nil, validateCertificateFile)
	runFileValidationTest(t, "non-existent cert file", "/nonexistent/cert.pem", 
		"certificate file not accessible", nil, validateCertificateFile)
	
	runFileValidationTest(t, "valid certificate file", "", "", func(t *testing.T) (string, func()) {
		certFile, _ := createTestCertificates(t)
		cleanup := func() {
			cleanupTestFile(certFile)
		}
		return certFile, cleanup
	}, validateCertificateFile)
}

func TestValidateKeyFile(t *testing.T) {
	runKeyFileValidationTests(t)
}

func runKeyFileValidationTests(t *testing.T) {
	t.Helper()
	runFileValidationTest(t, "empty key file path", "", "invalid key file path", 
		nil, validateKeyFile)
	runFileValidationTest(t, "key file with path traversal", "../../../etc/passwd", 
		"invalid key file path", nil, validateKeyFile)
	runFileValidationTest(t, "non-existent key file", "/nonexistent/key.pem", 
		"key file not accessible", nil, validateKeyFile)
	
	runFileValidationTest(t, "valid key file", "", "", func(t *testing.T) (string, func()) {
		_, keyFile := createTestCertificates(t)
		cleanup := func() {
			cleanupTestFile(keyFile)
		}
		return keyFile, cleanup
	}, validateKeyFile)
}

func TestValidateCertKeyPair(t *testing.T) {
	tests := getCertKeyPairTestCases()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runCertKeyPairTest(t, tt)
		})
	}
}

func getCertKeyPairTestCases() []struct {
	name        string
	setupFiles  func(t *testing.T) (string, string, func())
	expectedErr string
} {
	return []struct {
		name        string
		setupFiles  func(t *testing.T) (string, string, func())
		expectedErr string
	}{
		{
			name: "valid certificate and key pair",
			setupFiles: func(t *testing.T) (string, string, func()) {
				certFile, keyFile := createTestCertificates(t)
				cleanup := func() {
					cleanupTestFile(certFile)
					cleanupTestFile(keyFile)
				}
				return certFile, keyFile, cleanup
			},
		},
		{
			name: "non-existent certificate file",
			setupFiles: func(t *testing.T) (string, string, func()) {
				return "/nonexistent/cert.pem", "/nonexistent/key.pem", func() {}
			},
			expectedErr: "failed to load certificate/key pair",
		},
	}
}

func runCertKeyPairTest(t *testing.T, tt struct {
	name        string
	setupFiles  func(t *testing.T) (string, string, func())
	expectedErr string
}) {
	t.Helper()
	certFile, keyFile, cleanup := tt.setupFiles(t)
	defer cleanup()

	err := validateCertKeyPair(certFile, keyFile)

	if tt.expectedErr != "" {
		require.Error(t, err)
		assert.Contains(t, err.Error(), tt.expectedErr)
	} else {
		assert.NoError(t, err)
	}
}