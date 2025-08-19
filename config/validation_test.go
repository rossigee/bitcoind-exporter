package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidator_ValidateRPCConfig(t *testing.T) {
	tests := getRPCValidationTestCases()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runRPCValidationTest(t, tt)
		})
	}
}

// getRPCValidationTestCases returns test cases for RPC config validation
func getRPCValidationTestCases() []struct {
	name      string
	config    RPCConfig
	strict    bool
	wantValid bool
	wantError string
} {
	var tests []struct {
		name      string
		config    RPCConfig
		strict    bool
		wantValid bool
		wantError string
	}

	tests = append(tests, getValidRPCTestCases()...)
	tests = append(tests, getInvalidRPCTestCases()...)

	return tests
}

// getValidRPCTestCases returns valid RPC configuration test cases
func getValidRPCTestCases() []struct {
	name      string
	config    RPCConfig
	strict    bool
	wantValid bool
	wantError string
} {
	return []struct {
		name      string
		config    RPCConfig
		strict    bool
		wantValid bool
		wantError string
	}{
		{
			name: "valid HTTP config",
			config: RPCConfig{
				Address: "http://localhost:8332",
				User:    "bitcoin",
				Pass:    "password123",
				Timeout: 30 * time.Second,
			},
			strict:    false,
			wantValid: true,
		},
		{
			name: "valid HTTPS config",
			config: RPCConfig{
				Address: "https://localhost:8332",
				User:    "bitcoin",
				Pass:    "password123",
				Timeout: 30 * time.Second,
			},
			strict:    true,
			wantValid: true,
		},
	}
}

// getInvalidRPCTestCases returns invalid RPC configuration test cases
func getInvalidRPCTestCases() []struct {
	name      string
	config    RPCConfig
	strict    bool
	wantValid bool
	wantError string
} {
	var tests []struct {
		name      string
		config    RPCConfig
		strict    bool
		wantValid bool
		wantError string
	}

	tests = append(tests, getInvalidRPCAddressTestCases()...)
	tests = append(tests, getInvalidRPCCredentialTestCases()...)
	tests = append(tests, getInvalidRPCTimeoutTestCases()...)

	return tests
}

// getInvalidRPCAddressTestCases returns invalid RPC address test cases
func getInvalidRPCAddressTestCases() []struct {
	name      string
	config    RPCConfig
	strict    bool
	wantValid bool
	wantError string
} {
	return []struct {
		name      string
		config    RPCConfig
		strict    bool
		wantValid bool
		wantError string
	}{
		{
			name: "empty address",
			config: RPCConfig{
				Address: "",
				User:    "bitcoin",
				Pass:    "password123",
			},
			strict:    false,
			wantValid: false,
			wantError: "RPC address cannot be empty",
		},
		{
			name: "invalid URL scheme",
			config: RPCConfig{
				Address: "ftp://localhost:8332",
				User:    "bitcoin",
				Pass:    "password123",
			},
			strict:    false,
			wantValid: false,
			wantError: "RPC address must use http or https scheme",
		},
	}
}

// getInvalidRPCCredentialTestCases returns invalid RPC credential test cases
func getInvalidRPCCredentialTestCases() []struct {
	name      string
	config    RPCConfig
	strict    bool
	wantValid bool
	wantError string
} {
	return []struct {
		name      string
		config    RPCConfig
		strict    bool
		wantValid bool
		wantError string
	}{
		{
			name: "empty username",
			config: RPCConfig{
				Address: "http://localhost:8332",
				User:    "",
				Pass:    "password123",
			},
			strict:    false,
			wantValid: false,
			wantError: "RPC username cannot be empty",
		},
		{
			name: "empty password",
			config: RPCConfig{
				Address: "http://localhost:8332",
				User:    "bitcoin",
				Pass:    "",
			},
			strict:    false,
			wantValid: false,
			wantError: "RPC password cannot be empty",
		},
	}
}

// getInvalidRPCTimeoutTestCases returns invalid RPC timeout test cases
func getInvalidRPCTimeoutTestCases() []struct {
	name      string
	config    RPCConfig
	strict    bool
	wantValid bool
	wantError string
} {
	return []struct {
		name      string
		config    RPCConfig
		strict    bool
		wantValid bool
		wantError string
	}{
		{
			name: "short timeout",
			config: RPCConfig{
				Address: "http://localhost:8332",
				User:    "bitcoin",
				Pass:    "password123",
				Timeout: 500 * time.Millisecond,
			},
			strict:    false,
			wantValid: false,
			wantError: "RPC timeout must be at least 1 second",
		},
	}
}

// runRPCValidationTest executes a single RPC validation test case
func runRPCValidationTest(t *testing.T, tt struct {
	name      string
	config    RPCConfig
	strict    bool
	wantValid bool
	wantError string
}) {
	validator := NewValidator(tt.strict)
	result := ValidationResult{Valid: true}

	validator.validateRPCConfig(&tt.config, &result)

	assert.Equal(t, tt.wantValid, result.Valid)
	if !tt.wantValid && tt.wantError != "" {
		require.NotEmpty(t, result.Errors)
		assert.Contains(t, result.Errors[0].Message, tt.wantError)
	}
}

func TestValidator_ValidateZMQConfig(t *testing.T) {
	tests := getZMQValidationTestCases()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runZMQValidationTest(t, tt)
		})
	}
}

// getZMQValidationTestCases returns test cases for ZMQ validation
func getZMQValidationTestCases() []struct {
	name      string
	config    ZMQConfig
	wantValid bool
	wantError string
} {
	return []struct {
		name      string
		config    ZMQConfig
		wantValid bool
		wantError string
	}{
		{
			name: "valid ZMQ config",
			config: ZMQConfig{
				Enabled: true,
				Address: "tcp://localhost:28332",
				Timeout: 10 * time.Second,
			},
			wantValid: true,
		},
		{
			name: "disabled ZMQ",
			config: ZMQConfig{
				Enabled: false,
				Address: "",
			},
			wantValid: true,
		},
		{
			name: "empty address when enabled",
			config: ZMQConfig{
				Enabled: true,
				Address: "",
			},
			wantValid: false,
			wantError: "ZMQ address cannot be empty when enabled",
		},
		{
			name: "invalid ZMQ address format",
			config: ZMQConfig{
				Enabled: true,
				Address: "http://localhost:28332",
			},
			wantValid: false,
			wantError: "ZMQ address must be in format tcp://host:port",
		},
		{
			name: "invalid port",
			config: ZMQConfig{
				Enabled: true,
				Address: "tcp://localhost:70000",
			},
			wantValid: false,
			wantError: "ZMQ port must be between 1 and 65535",
		},
	}
}

// runZMQValidationTest executes a single ZMQ validation test case
func runZMQValidationTest(t *testing.T, tt struct {
	name      string
	config    ZMQConfig
	wantValid bool
	wantError string
}) {
	validator := NewValidator(false)
	result := ValidationResult{Valid: true}

	validator.validateZMQConfig(&tt.config, &result)

	assert.Equal(t, tt.wantValid, result.Valid)
	if !tt.wantValid && tt.wantError != "" {
		require.NotEmpty(t, result.Errors)
		assert.Contains(t, result.Errors[0].Message, tt.wantError)
	}
}

func TestValidator_ValidateMetricsConfig(t *testing.T) {
	tests := getMetricsValidationTestCases()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runMetricsValidationTest(t, tt)
		})
	}
}

// getMetricsValidationTestCases returns test cases for metrics config validation
func getMetricsValidationTestCases() []struct {
	name      string
	config    MetricsConfig
	wantValid bool
	wantError string
} {
	return []struct {
		name      string
		config    MetricsConfig
		wantValid bool
		wantError string
	}{
		{
			name: "valid metrics config",
			config: MetricsConfig{
				Port:          3000,
				Path:          "/metrics",
				FetchInterval: 10 * time.Second,
			},
			wantValid: true,
		},
		{
			name: "invalid port too low",
			config: MetricsConfig{
				Port:          0,
				Path:          "/metrics",
				FetchInterval: 10 * time.Second,
			},
			wantValid: false,
			wantError: "metrics port must be between 1 and 65535",
		},
		{
			name: "invalid port too high",
			config: MetricsConfig{
				Port:          70000,
				Path:          "/metrics",
				FetchInterval: 10 * time.Second,
			},
			wantValid: false,
			wantError: "metrics port must be between 1 and 65535",
		},
		{
			name: "invalid path",
			config: MetricsConfig{
				Port:          3000,
				Path:          "metrics", // missing leading slash
				FetchInterval: 10 * time.Second,
			},
			wantValid: false,
			wantError: "metrics path must start with /",
		},
		{
			name: "fetch interval too short",
			config: MetricsConfig{
				Port:          3000,
				Path:          "/metrics",
				FetchInterval: 500 * time.Millisecond,
			},
			wantValid: false,
			wantError: "fetch interval must be at least 1 second",
		},
	}
}

// runMetricsValidationTest executes a single metrics validation test case
func runMetricsValidationTest(t *testing.T, tt struct {
	name      string
	config    MetricsConfig
	wantValid bool
	wantError string
}) {
	validator := NewValidator(false)
	result := ValidationResult{Valid: true}

	validator.validateMetricsConfig(&tt.config, &result)

	assert.Equal(t, tt.wantValid, result.Valid)
	if !tt.wantValid && tt.wantError != "" {
		require.NotEmpty(t, result.Errors)
		assert.Contains(t, result.Errors[0].Message, tt.wantError)
	}
}

func TestValidator_ValidateSecurityConfig(t *testing.T) {
	tests := getSecurityValidationTestCases()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runSecurityValidationTest(t, tt)
		})
	}
}

// getSecurityValidationTestCases returns test cases for security config validation
func getSecurityValidationTestCases() []struct {
	name      string
	config    SecurityConfig
	wantValid bool
	wantError string
} {
	var tests []struct {
		name      string
		config    SecurityConfig
		wantValid bool
		wantError string
	}

	tests = append(tests, getValidSecurityTestCases()...)
	tests = append(tests, getInvalidSecurityTestCases()...)

	return tests
}

// getValidSecurityTestCases returns valid security configuration test cases
func getValidSecurityTestCases() []struct {
	name      string
	config    SecurityConfig
	wantValid bool
	wantError string
} {
	return []struct {
		name      string
		config    SecurityConfig
		wantValid bool
		wantError string
	}{
		{
			name: "valid security config with TLS and auth",
			config: SecurityConfig{
				TLSEnabled:   true,
				TLSCertFile:  "/path/to/cert.pem",
				TLSKeyFile:   "/path/to/key.pem",
				AuthEnabled:  true,
				AuthUsername: "admin",
				AuthPassword: "password123",
			},
			wantValid: true,
		},
	}
}

// getInvalidSecurityTestCases returns invalid security configuration test cases
func getInvalidSecurityTestCases() []struct {
	name      string
	config    SecurityConfig
	wantValid bool
	wantError string
} {
	return []struct {
		name      string
		config    SecurityConfig
		wantValid bool
		wantError string
	}{
		{
			name: "TLS enabled without cert file",
			config: SecurityConfig{
				TLSEnabled:  true,
				TLSCertFile: "",
				TLSKeyFile:  "/path/to/key.pem",
			},
			wantValid: false,
			wantError: "TLS certificate file must be specified when TLS is enabled",
		},
		{
			name: "TLS enabled without key file",
			config: SecurityConfig{
				TLSEnabled:  true,
				TLSCertFile: "/path/to/cert.pem",
				TLSKeyFile:  "",
			},
			wantValid: false,
			wantError: "TLS key file must be specified when TLS is enabled",
		},
		{
			name: "auth enabled without username",
			config: SecurityConfig{
				AuthEnabled:  true,
				AuthUsername: "",
				AuthPassword: "password123",
			},
			wantValid: false,
			wantError: "username must be specified when authentication is enabled",
		},
		{
			name: "auth enabled without password",
			config: SecurityConfig{
				AuthEnabled:  true,
				AuthUsername: "admin",
				AuthPassword: "",
			},
			wantValid: false,
			wantError: "password must be specified when authentication is enabled",
		},
	}
}

// runSecurityValidationTest executes a single security validation test case
func runSecurityValidationTest(t *testing.T, tt struct {
	name      string
	config    SecurityConfig
	wantValid bool
	wantError string
}) {
	validator := NewValidator(false)
	result := ValidationResult{Valid: true}

	validator.validateSecurityConfig(&tt.config, &result)

	assert.Equal(t, tt.wantValid, result.Valid)
	if !tt.wantValid && tt.wantError != "" {
		require.NotEmpty(t, result.Errors)
		assert.Contains(t, result.Errors[0].Message, tt.wantError)
	}
}

func TestValidator_ValidateAppConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    AppConfig
		wantValid bool
		wantError string
	}{
		{
			name: "valid app config",
			config: AppConfig{
				LogLevel:    "info",
				Environment: "production",
			},
			wantValid: true,
		},
		{
			name: "invalid log level",
			config: AppConfig{
				LogLevel:    "invalid",
				Environment: "production",
			},
			wantValid: false,
			wantError: "log level must be one of:",
		},
		{
			name: "valid debug level",
			config: AppConfig{
				LogLevel:    "debug",
				Environment: "development",
			},
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewValidator(false)
			result := ValidationResult{Valid: true}

			validator.validateAppConfig(&tt.config, &result)

			assert.Equal(t, tt.wantValid, result.Valid)
			if !tt.wantValid && tt.wantError != "" {
				require.NotEmpty(t, result.Errors)
				assert.Contains(t, result.Errors[0].Message, tt.wantError)
			}
		})
	}
}

func TestValidator_ValidateCompleteConfig(t *testing.T) {
	tests := getCompleteConfigTestCases()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runCompleteConfigTest(t, tt)
		})
	}
}

// getCompleteConfigTestCases returns test cases for complete config validation
func getCompleteConfigTestCases() []struct {
	name      string
	config    *Config
	strict    bool
	wantValid bool
	wantError string
} {
	var tests []struct {
		name      string
		config    *Config
		strict    bool
		wantValid bool
		wantError string
	}

	tests = append(tests, getValidCompleteConfigTestCases()...)
	tests = append(tests, getInvalidCompleteConfigTestCases()...)

	return tests
}

// getValidCompleteConfigTestCases returns valid complete configuration test cases
func getValidCompleteConfigTestCases() []struct {
	name      string
	config    *Config
	strict    bool
	wantValid bool
	wantError string
} {
	return []struct {
		name      string
		config    *Config
		strict    bool
		wantValid bool
		wantError string
	}{
		{
			name: "valid complete config",
			config: &Config{
				RPC: RPCConfig{
					Address: "https://localhost:8332",
					User:    "bitcoin",
					Pass:    "password123",
					Timeout: 30 * time.Second,
				},
				ZMQ: ZMQConfig{
					Enabled: true,
					Address: "tcp://localhost:28332",
					Timeout: 10 * time.Second,
				},
				Metrics: MetricsConfig{
					Port:          3000,
					Path:          "/metrics",
					FetchInterval: 10 * time.Second,
				},
				Security: SecurityConfig{
					TLSEnabled:   true,
					TLSCertFile:  "/path/to/cert.pem",
					TLSKeyFile:   "/path/to/key.pem",
					AuthEnabled:  true,
					AuthUsername: "admin",
					AuthPassword: "password123",
				},
				App: AppConfig{
					LogLevel:    "info",
					Environment: "production",
				},
			},
			strict:    true,
			wantValid: true,
		},
	}
}

// getInvalidCompleteConfigTestCases returns invalid complete configuration test cases
func getInvalidCompleteConfigTestCases() []struct {
	name      string
	config    *Config
	strict    bool
	wantValid bool
	wantError string
} {
	return []struct {
		name      string
		config    *Config
		strict    bool
		wantValid bool
		wantError string
	}{
		{
			name: "production config without TLS",
			config: &Config{
				RPC: RPCConfig{
					Address: "http://localhost:8332",
					User:    "bitcoin",
					Pass:    "password123",
					Timeout: 30 * time.Second,
				},
				Metrics: MetricsConfig{
					Port:          3000,
					Path:          "/metrics",
					FetchInterval: 10 * time.Second,
				},
				Security: SecurityConfig{
					TLSEnabled: false,
				},
				App: AppConfig{
					LogLevel:    "info",
					Environment: "production",
				},
			},
			strict:    true,
			wantValid: false,
			wantError: "TLS must be enabled in production environment",
		},
	}
}

// runCompleteConfigTest executes a single complete config validation test case
func runCompleteConfigTest(t *testing.T, tt struct {
	name      string
	config    *Config
	strict    bool
	wantValid bool
	wantError string
}) {
	validator := NewValidator(tt.strict)
	result := validator.ValidateConfig(tt.config)

	assert.Equal(t, tt.wantValid, result.Valid)
	if !tt.wantValid && tt.wantError != "" {
		require.NotEmpty(t, result.Errors)
		assert.Contains(t, result.Errors[0].Message, tt.wantError)
	}
}

func TestValidator_IsValidIPOrCIDR(t *testing.T) {
	tests := []struct {
		ip    string
		valid bool
	}{
		{"192.168.1.1", true},
		{"10.0.0.0/8", true},
		{"172.16.0.0/12", true},
		{"192.168.1.0/24", true},
		{"127.0.0.1", true},
		{"0.0.0.0", true},
		{"255.255.255.255", true},
		{"256.1.1.1", false},
		{"192.168.1", false},
		{"192.168.1.1.1", false},
		{"192.168.1.1/33", false},
		{"192.168.1.1/-1", false},
		{"not.an.ip", false},
		{"", false},
	}

	validator := NewValidator(false)
	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			result := validator.isValidIPOrCIDR(tt.ip)
			assert.Equal(t, tt.valid, result, "IP validation failed for %s", tt.ip)
		})
	}
}

func TestValidationResult_AddError(t *testing.T) {
	result := ValidationResult{Valid: true}

	result.AddError("test_field", "test_value", "test error message")

	assert.False(t, result.Valid)
	assert.Len(t, result.Errors, 1)
	assert.Equal(t, "test_field", result.Errors[0].Field)
	assert.Equal(t, "test_value", result.Errors[0].Value)
	assert.Equal(t, "test error message", result.Errors[0].Message)
}

func TestValidationResult_AddWarning(t *testing.T) {
	result := ValidationResult{Valid: true}

	result.AddWarning("test warning message")

	assert.True(t, result.Valid) // Warnings don't affect validity
	assert.Len(t, result.Warnings, 1)
	assert.Equal(t, "test warning message", result.Warnings[0])
}

func TestValidationError_Error(t *testing.T) {
	err := ValidationError{
		Field:   "test_field",
		Value:   "test_value",
		Message: "test message",
	}

	expected := "configuration validation failed for field 'test_field' with value 'test_value': test message"
	assert.Equal(t, expected, err.Error())
}

func TestMatchesIP(t *testing.T) {
	tests := getMatchesIPTestCases()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesIP(tt.clientIP, tt.pattern)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func getMatchesIPTestCases() []struct {
	name     string
	clientIP string
	pattern  string
	expected bool
} {
	basicTests := getBasicIPMatchTests()
	cidrTests := getCIDRMatchTests()
	ipv6Tests := getIPv6MatchTests()
	edgeTests := getIPMatchEdgeTests()
	
	return append(append(append(basicTests, cidrTests...), ipv6Tests...), edgeTests...)
}

func getBasicIPMatchTests() []struct {
	name     string
	clientIP string
	pattern  string
	expected bool
} {
	return []struct {
		name     string
		clientIP string
		pattern  string
		expected bool
	}{
		{
			name:     "Exact IP match",
			clientIP: "192.168.1.100",
			pattern:  "192.168.1.100",
			expected: true,
		},
		{
			name:     "IP with port matches exact IP",
			clientIP: "192.168.1.100:8080",
			pattern:  "192.168.1.100",
			expected: true,
		},
		{
			name:     "Different IPs don't match",
			clientIP: "192.168.1.100",
			pattern:  "192.168.1.101",
			expected: false,
		},
	}
}

func getCIDRMatchTests() []struct {
	name     string
	clientIP string
	pattern  string
	expected bool
} {
	return []struct {
		name     string
		clientIP string
		pattern  string
		expected bool
	}{
		{
			name:     "CIDR match - client in subnet",
			clientIP: "192.168.1.100",
			pattern:  "192.168.1.0/24",
			expected: true,
		},
		{
			name:     "CIDR match - client in subnet with port",
			clientIP: "192.168.1.100:8080",
			pattern:  "192.168.1.0/24",
			expected: true,
		},
		{
			name:     "CIDR no match - client outside subnet",
			clientIP: "192.168.2.100",
			pattern:  "192.168.1.0/24",
			expected: false,
		},
		{
			name:     "CIDR /32 exact match",
			clientIP: "192.168.1.100",
			pattern:  "192.168.1.100/32",
			expected: true,
		},
		{
			name:     "CIDR /32 no match",
			clientIP: "192.168.1.100",
			pattern:  "192.168.1.101/32",
			expected: false,
		},
		{
			name:     "Localhost CIDR",
			clientIP: "127.0.0.1",
			pattern:  "127.0.0.0/8",
			expected: true,
		},
	}
}

func getIPv6MatchTests() []struct {
	name     string
	clientIP string
	pattern  string
	expected bool
} {
	return []struct {
		name     string
		clientIP string
		pattern  string
		expected bool
	}{
		{
			name:     "IPv6 exact match",
			clientIP: "2001:db8::1",
			pattern:  "2001:db8::1",
			expected: true,
		},
		{
			name:     "IPv6 CIDR match",
			clientIP: "2001:db8::1",
			pattern:  "2001:db8::/32",
			expected: true,
		},
		{
			name:     "IPv6 CIDR no match",
			clientIP: "2001:db9::1",
			pattern:  "2001:db8::/32",
			expected: false,
		},
	}
}

func getIPMatchEdgeTests() []struct {
	name     string
	clientIP string
	pattern  string
	expected bool
} {
	return []struct {
		name     string
		clientIP string
		pattern  string
		expected bool
	}{
		{
			name:     "Invalid CIDR pattern",
			clientIP: "192.168.1.100",
			pattern:  "192.168.1.0/invalid",
			expected: false,
		},
		{
			name:     "Invalid client IP",
			clientIP: "invalid-ip",
			pattern:  "192.168.1.0/24",
			expected: false,
		},
	}
}
