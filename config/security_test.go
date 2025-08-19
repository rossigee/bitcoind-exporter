package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadSecurityConfig(t *testing.T) {
	tests := getLoadSecurityConfigTestCases()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runLoadSecurityConfigTest(t, tt)
		})
	}
}

func getLoadSecurityConfigTestCases() []struct {
	name     string
	envVars  map[string]string
	wantErr  bool
	validate func(t *testing.T, cfg SecurityConfig)
} {
	validTests := getValidLoadSecurityConfigCases()
	invalidTests := getInvalidLoadSecurityConfigCases()
	
	allTests := make([]struct {
		name     string
		envVars  map[string]string
		wantErr  bool
		validate func(t *testing.T, cfg SecurityConfig)
	}, 0, len(validTests)+len(invalidTests))
	
	allTests = append(allTests, validTests...)
	allTests = append(allTests, invalidTests...)
	
	return allTests
}

func getValidLoadSecurityConfigCases() []struct {
	name     string
	envVars  map[string]string
	wantErr  bool
	validate func(t *testing.T, cfg SecurityConfig)
} {
	return []struct {
		name     string
		envVars  map[string]string
		wantErr  bool
		validate func(t *testing.T, cfg SecurityConfig)
	}{
		{
			name:     "default security configuration",
			envVars:  map[string]string{},
			wantErr:  false,
			validate: validateDefaultSecurityConfig,
		},
		{
			name:     "TLS configuration",
			envVars:  getTLSConfigEnvVars(),
			wantErr:  false,
			validate: validateTLSConfig,
		},
		{
			name:     "authentication configuration",
			envVars:  getAuthConfigEnvVars(),
			wantErr:  false,
			validate: validateAuthConfig,
		},
		{
			name:     "rate limiting configuration", 
			envVars:  getRateLimitConfigEnvVars(),
			wantErr:  false,
			validate: validateRateLimitConfig,
		},
		{
			name:     "IP filtering configuration",
			envVars:  getIPFilteringConfigEnvVars(),
			wantErr:  false,
			validate: validateIPFilteringConfig,
		},
		{
			name:     "server timeouts configuration",
			envVars:  getServerTimeoutsConfigEnvVars(),
			wantErr:  false,
			validate: validateServerTimeoutsConfig,
		},
	}
}

func getInvalidLoadSecurityConfigCases() []struct {
	name     string
	envVars  map[string]string
	wantErr  bool
	validate func(t *testing.T, cfg SecurityConfig)
} {
	return []struct {
		name     string
		envVars  map[string]string
		wantErr  bool
		validate func(t *testing.T, cfg SecurityConfig)
	}{
		{
			name:    "invalid TLS version",
			envVars: getInvalidTLSVersionEnvVars(),
			wantErr: true,
		},
		{
			name:    "short password",
			envVars: getShortPasswordEnvVars(),
			wantErr: true,
		},
		{
			name:    "invalid IP address",
			envVars: getInvalidIPEnvVars(),
			wantErr: true,
		},
	}
}

func validateDefaultSecurityConfig(t *testing.T, cfg SecurityConfig) {
	t.Helper()
	assert.False(t, cfg.TLSEnabled)
	assert.False(t, cfg.AuthEnabled)
	assert.False(t, cfg.RateLimitEnabled)
	assert.Equal(t, "1.2", cfg.TLSMinVersion)
	assert.Equal(t, 100, cfg.RateLimitRequests)
	assert.Equal(t, time.Minute, cfg.RateLimitWindow)
}

func getTLSConfigEnvVars() map[string]string {
	return map[string]string{
		"TLS_ENABLED":     "true",
		"TLS_CERT_FILE":   "/tmp/test-cert.pem",
		"TLS_KEY_FILE":    "/tmp/test-key.pem",
		"TLS_MIN_VERSION": "1.3",
	}
}

func validateTLSConfig(t *testing.T, cfg SecurityConfig) {
	t.Helper()
	assert.True(t, cfg.TLSEnabled)
	assert.NotEmpty(t, cfg.TLSCertFile)
	assert.NotEmpty(t, cfg.TLSKeyFile)
	assert.Equal(t, "1.3", cfg.TLSMinVersion)
}

func getAuthConfigEnvVars() map[string]string {
	return map[string]string{
		"AUTH_ENABLED":  "true",
		"AUTH_USERNAME": "admin",
		"AUTH_PASSWORD": "securepassword123",
	}
}

func validateAuthConfig(t *testing.T, cfg SecurityConfig) {
	t.Helper()
	assert.True(t, cfg.AuthEnabled)
	assert.Equal(t, "admin", cfg.AuthUsername)
	assert.Equal(t, "securepassword123", cfg.AuthPassword)
}

func getRateLimitConfigEnvVars() map[string]string {
	return map[string]string{
		"RATE_LIMIT_ENABLED":    "true",
		"RATE_LIMIT_REQUESTS":   "50",
		"RATE_LIMIT_WINDOW":     "30s",
		"RATE_LIMIT_BLOCK_TIME": "5m",
	}
}

func validateRateLimitConfig(t *testing.T, cfg SecurityConfig) {
	t.Helper()
	assert.True(t, cfg.RateLimitEnabled)
	assert.Equal(t, 50, cfg.RateLimitRequests)
	assert.Equal(t, 30*time.Second, cfg.RateLimitWindow)
	assert.Equal(t, 5*time.Minute, cfg.RateLimitBlockTime)
}

func getIPFilteringConfigEnvVars() map[string]string {
	return map[string]string{
		"ALLOWED_IPS": "192.168.1.0/24,10.0.0.1,127.0.0.1",
		"DENIED_IPS":  "192.168.1.100,10.0.0.0/8",
	}
}

func validateIPFilteringConfig(t *testing.T, cfg SecurityConfig) {
	t.Helper()
	assert.Equal(t, []string{"192.168.1.0/24", "10.0.0.1", "127.0.0.1"}, cfg.AllowedIPs)
	assert.Equal(t, []string{"192.168.1.100", "10.0.0.0/8"}, cfg.DeniedIPs)
}

func getServerTimeoutsConfigEnvVars() map[string]string {
	return map[string]string{
		"READ_TIMEOUT":        "15s",
		"WRITE_TIMEOUT":       "20s",
		"IDLE_TIMEOUT":        "120s",
		"READ_HEADER_TIMEOUT": "10s",
	}
}

func validateServerTimeoutsConfig(t *testing.T, cfg SecurityConfig) {
	t.Helper()
	assert.Equal(t, 15*time.Second, cfg.ServerTimeouts.ReadTimeout)
	assert.Equal(t, 20*time.Second, cfg.ServerTimeouts.WriteTimeout)
	assert.Equal(t, 120*time.Second, cfg.ServerTimeouts.IdleTimeout)
	assert.Equal(t, 10*time.Second, cfg.ServerTimeouts.ReadHeaderTimeout)
}

func getInvalidTLSVersionEnvVars() map[string]string {
	return map[string]string{
		"TLS_ENABLED":     "true",
		"TLS_CERT_FILE":   "/tmp/cert.pem",
		"TLS_KEY_FILE":    "/tmp/key.pem",
		"TLS_MIN_VERSION": "2.0",
	}
}

func getShortPasswordEnvVars() map[string]string {
	return map[string]string{
		"AUTH_ENABLED":  "true",
		"AUTH_USERNAME": "admin",
		"AUTH_PASSWORD": "short",
	}
}

func getInvalidIPEnvVars() map[string]string {
	return map[string]string{
		"ALLOWED_IPS": "invalid.ip,192.168.1.1",
	}
}

func runLoadSecurityConfigTest(t *testing.T, tt struct {
	name     string
	envVars  map[string]string
	wantErr  bool
	validate func(t *testing.T, cfg SecurityConfig)
}) {
	t.Helper()
	// Create temporary cert files if TLS is enabled
	if tt.envVars["TLS_ENABLED"] == "true" && !tt.wantErr {
		certFile, keyFile := createTestCertificates(t)
		tt.envVars["TLS_CERT_FILE"] = certFile
		tt.envVars["TLS_KEY_FILE"] = keyFile
		defer cleanupTestFile(certFile)
		defer cleanupTestFile(keyFile)
	}

	// Set up environment
	oldEnv := setupSecurityTestEnvironment(t, tt.envVars)
	defer restoreEnvironment(oldEnv)

	// Load security configuration
	err := LoadSecurityConfig()

	if tt.wantErr {
		assert.Error(t, err)
		return
	}

	require.NoError(t, err)

	// Validate specific fields
	if tt.validate != nil {
		tt.validate(t, Security)
	}
}

func TestGetSecurityHeaders(t *testing.T) {
	headers := GetSecurityHeaders()

	expectedHeaders := map[string]string{
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
		"X-XSS-Protection":          "1; mode=block",
		"Referrer-Policy":           "strict-origin-when-cross-origin",
		"Content-Security-Policy":   "default-src 'self'",
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
	}

	assert.Equal(t, expectedHeaders, headers)
}

func TestIsIPAllowed(t *testing.T) {
	tests := getIsIPAllowedTestCases()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runIsIPAllowedTest(t, tt)
		})
	}
}

func getIsIPAllowedTestCases() []struct {
	name       string
	allowedIPs []string
	deniedIPs  []string
	clientIP   string
	expected   bool
} {
	basicTests := getBasicIPAllowedTestCases()
	cidrTests := getCIDRIPAllowedTestCases()
	specialTests := getSpecialIPAllowedTestCases()
	
	allTests := make([]struct {
		name       string
		allowedIPs []string
		deniedIPs  []string
		clientIP   string
		expected   bool
	}, 0, len(basicTests)+len(cidrTests)+len(specialTests))
	
	allTests = append(allTests, basicTests...)
	allTests = append(allTests, cidrTests...)
	allTests = append(allTests, specialTests...)
	
	return allTests
}

func getBasicIPAllowedTestCases() []struct {
	name       string
	allowedIPs []string
	deniedIPs  []string
	clientIP   string
	expected   bool
} {
	return []struct {
		name       string
		allowedIPs []string
		deniedIPs  []string
		clientIP   string
		expected   bool
	}{
		{
			name:       "no restrictions - allow all",
			allowedIPs: []string{},
			deniedIPs:  []string{},
			clientIP:   "192.168.1.100",
			expected:   true,
		},
		{
			name:       "client in allowed list",
			allowedIPs: []string{"192.168.1.100", "10.0.0.1"},
			deniedIPs:  []string{},
			clientIP:   "192.168.1.100",
			expected:   true,
		},
		{
			name:       "client not in allowed list",
			allowedIPs: []string{"192.168.1.100", "10.0.0.1"},
			deniedIPs:  []string{},
			clientIP:   "192.168.1.101",
			expected:   false,
		},
		{
			name:       "client in denied list",
			allowedIPs: []string{},
			deniedIPs:  []string{"192.168.1.100"},
			clientIP:   "192.168.1.100",
			expected:   false,
		},
		{
			name:       "client allowed but also denied - denied wins",
			allowedIPs: []string{"192.168.1.100"},
			deniedIPs:  []string{"192.168.1.100"},
			clientIP:   "192.168.1.100",
			expected:   false,
		},
	}
}

func getCIDRIPAllowedTestCases() []struct {
	name       string
	allowedIPs []string
	deniedIPs  []string
	clientIP   string
	expected   bool
} {
	return []struct {
		name       string
		allowedIPs []string
		deniedIPs  []string
		clientIP   string
		expected   bool
	}{
		{
			name:       "client in allowed CIDR range",
			allowedIPs: []string{"192.168.1.0/24"},
			deniedIPs:  []string{},
			clientIP:   "192.168.1.150",
			expected:   true,
		},
		{
			name:       "client outside allowed CIDR range",
			allowedIPs: []string{"192.168.1.0/24"},
			deniedIPs:  []string{},
			clientIP:   "192.168.2.150",
			expected:   false,
		},
		{
			name:       "localhost allowed",
			allowedIPs: []string{"127.0.0.0/8"},
			deniedIPs:  []string{},
			clientIP:   "127.0.0.1",
			expected:   true,
		},
	}
}

func getSpecialIPAllowedTestCases() []struct {
	name       string
	allowedIPs []string
	deniedIPs  []string
	clientIP   string
	expected   bool
} {
	return []struct {
		name       string
		allowedIPs []string
		deniedIPs  []string
		clientIP   string
		expected   bool
	}{
		{
			name:       "client with port in allowed list",
			allowedIPs: []string{"192.168.1.100"},
			deniedIPs:  []string{},
			clientIP:   "192.168.1.100:8080",
			expected:   true,
		},
		{
			name:       "IPv6 client allowed",
			allowedIPs: []string{"2001:db8::1"},
			deniedIPs:  []string{},
			clientIP:   "[2001:db8::1]:8080",
			expected:   true,
		},
	}
}

func runIsIPAllowedTest(t *testing.T, tt struct {
	name       string
	allowedIPs []string
	deniedIPs  []string
	clientIP   string
	expected   bool
}) {
	t.Helper()
	// Set up Security config
	Security.AllowedIPs = tt.allowedIPs
	Security.DeniedIPs = tt.deniedIPs

	result := IsIPAllowed(tt.clientIP)
	assert.Equal(t, tt.expected, result)
}


func TestIsValidIP(t *testing.T) {
	tests := []struct {
		ip       string
		expected bool
	}{
		// Valid IPv4
		{"192.168.1.1", true},
		{"127.0.0.1", true},
		{"0.0.0.0", true},
		{"255.255.255.255", true},
		{"10.0.0.1", true},

		// Valid IPv4 with CIDR
		{"192.168.1.0/24", true},
		{"10.0.0.0/8", true},
		{"172.16.0.0/12", true},

		// Valid IPv6 (simplified validation)
		{"2001:db8::1", true},
		{"::1", true},
		{"fe80::1", true},
		{"2001:db8:85a3::8a2e:370:7334", true},

		// Invalid IPs
		{"", false},
		{"256.1.1.1", false},
		{"192.168.1", false},
		{"192.168.1.1.1", false},
		{"not.an.ip", false},
		{"192.168.1.-1", false},
		{"192.168.1.256", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			result := isValidIP(tt.ip)
			assert.Equal(t, tt.expected, result, "IP validation failed for %s", tt.ip)
		})
	}
}

func TestContains(t *testing.T) {
	slice := []string{"apple", "banana", "cherry"}

	assert.True(t, contains(slice, "apple"))
	assert.True(t, contains(slice, "banana"))
	assert.True(t, contains(slice, "cherry"))
	assert.False(t, contains(slice, "orange"))
	assert.False(t, contains(slice, ""))
	assert.False(t, contains([]string{}, "apple"))
}

func TestValidateSecurityConfig(t *testing.T) {
	tests := getValidateSecurityConfigTestCases()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runValidateSecurityConfigTest(t, tt)
		})
	}
}

func getValidateSecurityConfigTestCases() []struct {
	name        string
	setupConfig func()
	expectedErr string
	setupFiles  func(t *testing.T) func()
} {
	validTests := getValidSecurityConfigTestCases()
	tlsErrorTests := getTLSErrorTestCases()
	authErrorTests := getAuthErrorTestCases()
	otherErrorTests := getOtherSecurityErrorTestCases()
	
	allTests := make([]struct {
		name        string
		setupConfig func()
		expectedErr string
		setupFiles  func(t *testing.T) func()
	}, 0, len(validTests)+len(tlsErrorTests)+len(authErrorTests)+len(otherErrorTests))
	
	allTests = append(allTests, validTests...)
	allTests = append(allTests, tlsErrorTests...)
	allTests = append(allTests, authErrorTests...)
	allTests = append(allTests, otherErrorTests...)
	
	return allTests
}

func getValidSecurityConfigTestCases() []struct {
	name        string
	setupConfig func()
	expectedErr string
	setupFiles  func(t *testing.T) func()
} {
	return []struct {
		name        string
		setupConfig func()
		expectedErr string
		setupFiles  func(t *testing.T) func()
	}{
		{
			name:        "valid configuration with all features disabled",
			setupConfig: setupValidDisabledConfig,
		},
		{
			name:        "valid TLS configuration with test files",
			setupConfig: func() {},
			setupFiles:  setupValidTLSConfigWithFiles,
		},
	}
}

func getTLSErrorTestCases() []struct {
	name        string
	setupConfig func()
	expectedErr string
	setupFiles  func(t *testing.T) func()
} {
	return []struct {
		name        string
		setupConfig func()
		expectedErr string
		setupFiles  func(t *testing.T) func()
	}{
		{
			name:        "TLS enabled without cert file",
			setupConfig: setupTLSWithoutCertConfig,
			expectedErr: "TLS_CERT_FILE is required when TLS is enabled",
		},
		{
			name:        "TLS enabled without key file",
			setupConfig: setupTLSWithoutKeyConfig,
			expectedErr: "TLS_KEY_FILE is required when TLS is enabled",
		},
		{
			name:        "TLS enabled with non-existent cert file",
			setupConfig: setupTLSWithNonexistentCertConfig,
			expectedErr: "TLS cert file does not exist",
		},
	}
}

func getAuthErrorTestCases() []struct {
	name        string
	setupConfig func()
	expectedErr string
	setupFiles  func(t *testing.T) func()
} {
	return []struct {
		name        string
		setupConfig func()
		expectedErr string
		setupFiles  func(t *testing.T) func()
	}{
		{
			name:        "auth enabled without username",
			setupConfig: setupAuthWithoutUsernameConfig,
			expectedErr: "AUTH_USERNAME is required when auth is enabled",
		},
		{
			name:        "auth enabled without password",
			setupConfig: setupAuthWithoutPasswordConfig,
			expectedErr: "AUTH_PASSWORD is required when auth is enabled",
		},
		{
			name:        "auth enabled with short password",
			setupConfig: setupAuthWithShortPasswordConfig,
			expectedErr: "AUTH_PASSWORD must be at least 8 characters long",
		},
	}
}

func getOtherSecurityErrorTestCases() []struct {
	name        string
	setupConfig func()
	expectedErr string
	setupFiles  func(t *testing.T) func()
} {
	return []struct {
		name        string
		setupConfig func()
		expectedErr string
		setupFiles  func(t *testing.T) func()
	}{
		{
			name:        "rate limiting with negative requests",
			setupConfig: setupRateLimitNegativeRequestsConfig,
			expectedErr: "RATE_LIMIT_REQUESTS must be positive",
		},
		{
			name:        "rate limiting with zero window",
			setupConfig: setupRateLimitZeroWindowConfig,
			expectedErr: "RATE_LIMIT_WINDOW must be positive",
		},
		{
			name:        "invalid allowed IP",
			setupConfig: setupInvalidAllowedIPConfig,
			expectedErr: "invalid IP address in ALLOWED_IPS",
		},
		{
			name:        "invalid denied IP",
			setupConfig: setupInvalidDeniedIPConfig,
			expectedErr: "invalid IP address in DENIED_IPS",
		},
	}
}

func setupValidDisabledConfig() {
	Security = SecurityConfig{
		TLSEnabled:       false,
		AuthEnabled:      false,
		RateLimitEnabled: false,
	}
}

func setupTLSWithoutCertConfig() {
	Security = SecurityConfig{
		TLSEnabled: true,
		TLSKeyFile: "/tmp/key.pem",
	}
}

func setupTLSWithoutKeyConfig() {
	Security = SecurityConfig{
		TLSEnabled:  true,
		TLSCertFile: "/tmp/cert.pem",
	}
}

func setupTLSWithNonexistentCertConfig() {
	Security = SecurityConfig{
		TLSEnabled:  true,
		TLSCertFile: "/nonexistent/cert.pem",
		TLSKeyFile:  "/tmp/key.pem",
	}
}

func setupAuthWithoutUsernameConfig() {
	Security = SecurityConfig{
		AuthEnabled:  true,
		AuthPassword: "password123",
	}
}

func setupAuthWithoutPasswordConfig() {
	Security = SecurityConfig{
		AuthEnabled:  true,
		AuthUsername: "admin",
	}
}

func setupAuthWithShortPasswordConfig() {
	Security = SecurityConfig{
		AuthEnabled:  true,
		AuthUsername: "admin",
		AuthPassword: "short",
	}
}

func setupRateLimitNegativeRequestsConfig() {
	Security = SecurityConfig{
		RateLimitEnabled:  true,
		RateLimitRequests: -1,
	}
}

func setupRateLimitZeroWindowConfig() {
	Security = SecurityConfig{
		RateLimitEnabled:  true,
		RateLimitRequests: 100,
		RateLimitWindow:   0,
	}
}

func setupInvalidAllowedIPConfig() {
	Security = SecurityConfig{
		AllowedIPs: []string{"invalid.ip"},
	}
}

func setupInvalidDeniedIPConfig() {
	Security = SecurityConfig{
		DeniedIPs: []string{"999.999.999.999"},
	}
}

func setupValidTLSConfigWithFiles(t *testing.T) func() {
	t.Helper()
	certFile, keyFile := createTestCertificates(t)
	Security = SecurityConfig{
		TLSEnabled:    true,
		TLSCertFile:   certFile,
		TLSKeyFile:    keyFile,
		TLSMinVersion: "1.2",
	}
	return func() {
		cleanupTestFile(certFile)
		cleanupTestFile(keyFile)
	}
}

func runValidateSecurityConfigTest(t *testing.T, tt struct {
	name        string
	setupConfig func()
	expectedErr string
	setupFiles  func(t *testing.T) func()
}) {
	t.Helper()
	var cleanup func()
	if tt.setupFiles != nil {
		cleanup = tt.setupFiles(t)
	}
	defer func() {
		if cleanup != nil {
			cleanup()
		}
	}()

	tt.setupConfig()

	err := validateSecurityConfig()

	if tt.expectedErr != "" {
		require.Error(t, err)
		assert.Contains(t, err.Error(), tt.expectedErr)
	} else {
		assert.NoError(t, err)
	}
}

// setupSecurityTestEnvironment sets up environment variables for testing
func setupSecurityTestEnvironment(_ *testing.T, envVars map[string]string) []string {
	// Save current environment
	oldEnv := os.Environ()
	
	// Clear environment
	os.Clearenv()
	
	// Set test environment variables
	for key, value := range envVars {
		_ = os.Setenv(key, value)
	}
	
	return oldEnv
}

func TestLogSecurityConfig(t *testing.T) {
	// Set up a configuration
	Security = SecurityConfig{
		TLSEnabled:        true,
		TLSMinVersion:     "1.3",
		AuthEnabled:       true,
		AuthUsername:      "admin",
		RateLimitEnabled:  true,
		RateLimitRequests: 50,
		RateLimitWindow:   time.Minute,
		AllowedIPs:        []string{"192.168.1.0/24", "127.0.0.1"},
		DeniedIPs:         []string{"192.168.1.100"},
	}

	// This test mainly ensures the function doesn't panic
	// In a real scenario, you might want to capture log output
	assert.NotPanics(t, func() {
		logSecurityConfig()
	})
}