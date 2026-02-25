package config

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Validation constants
const (
	minZMQAddressParts = 3     // Minimum parts in ZMQ address
	maxRateLimit       = 10000 // Maximum rate limit warning threshold
	maxCIDRParts       = 2     // Expected parts in CIDR notation
)

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Value   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("configuration validation failed for field '%s' with value '%s': %s",
		e.Field, e.Value, e.Message)
}

// ValidationResult holds validation results
type ValidationResult struct {
	Valid    bool
	Errors   []ValidationError
	Warnings []string
}

// AddError adds a validation error
func (v *ValidationResult) AddError(field, value, message string) {
	v.Valid = false
	v.Errors = append(v.Errors, ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	})
}

// AddWarning adds a validation warning
func (v *ValidationResult) AddWarning(message string) {
	v.Warnings = append(v.Warnings, message)
}

// Validator provides comprehensive configuration validation
type Validator struct {
	strictMode bool
}

// NewValidator creates a new configuration validator
func NewValidator(strictMode bool) *Validator {
	return &Validator{
		strictMode: strictMode,
	}
}

// ValidateConfig performs comprehensive validation of all configuration
func (v *Validator) ValidateConfig(cfg *Config) ValidationResult {
	result := ValidationResult{Valid: true}

	// Validate RPC configuration
	v.validateRPCConfig(&cfg.RPC, &result)

	// Validate ZMQ configuration
	v.validateZMQConfig(&cfg.ZMQ, &result)

	// Validate Metrics configuration
	v.validateMetricsConfig(&cfg.Metrics, &result)

	// Validate Security configuration
	v.validateSecurityConfig(&cfg.Security, &result)

	// Validate Application configuration
	v.validateAppConfig(&cfg.App, &result)

	// Validate cross-field dependencies
	v.validateCrossFieldDependencies(cfg, &result)

	return result
}

// validateRPCConfig validates RPC-related configuration
func (v *Validator) validateRPCConfig(rpc *RPCConfig, result *ValidationResult) {
	// Validate RPC URL
	if rpc.Address == "" {
		result.AddError("RPC_ADDRESS", rpc.Address, "RPC address cannot be empty")
		return
	}

	parsedURL, err := url.Parse(rpc.Address)
	if err != nil {
		result.AddError("RPC_ADDRESS", rpc.Address,
			fmt.Sprintf("invalid URL format: %v", err))
		return
	}

	// Validate URL scheme
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		result.AddError("RPC_ADDRESS", rpc.Address,
			"RPC address must use http or https scheme")
	}

	// Validate hostname
	if parsedURL.Hostname() == "" {
		result.AddError("RPC_ADDRESS", rpc.Address, "RPC address must include hostname")
	}

	// Validate port
	if parsedURL.Port() != "" {
		port, err := strconv.Atoi(parsedURL.Port())
		if err != nil || port < 1 || port > 65535 {
			result.AddError("RPC_ADDRESS", rpc.Address,
				"RPC address port must be between 1 and 65535")
		}
	}

	// Validate credentials
	if rpc.User == "" {
		result.AddError("RPC_USER", rpc.User, "RPC username cannot be empty")
	}

	if rpc.Pass == "" {
		result.AddError("RPC_PASS", "***", "RPC password cannot be empty")
	}

	// Security warnings
	if parsedURL.Scheme == "http" && v.strictMode {
		result.AddWarning("Using HTTP for RPC connection. Consider HTTPS for production")
	}

	if len(rpc.Pass) < 8 && v.strictMode {
		result.AddWarning("RPC password is shorter than 8 characters")
	}

	// Validate timeout
	if rpc.Timeout < 1*time.Second {
		result.AddError("RPC_TIMEOUT", rpc.Timeout.String(),
			"RPC timeout must be at least 1 second")
	}

	if rpc.Timeout > 300*time.Second {
		result.AddWarning("RPC timeout is very high (>5 minutes)")
	}
}

// validateZMQConfig validates ZMQ-related configuration
func (v *Validator) validateZMQConfig(zmq *ZMQConfig, result *ValidationResult) {
	if !zmq.Enabled {
		return
	}

	if zmq.Address == "" {
		result.AddError("ZMQ_ADDRESS", zmq.Address, "ZMQ address cannot be empty when enabled")
		return
	}

	// Validate ZMQ address format
	zmqRegex := regexp.MustCompile(`^tcp://[^:]+:\d+$`)
	if !zmqRegex.MatchString(zmq.Address) {
		result.AddError("ZMQ_ADDRESS", zmq.Address,
			"ZMQ address must be in format tcp://host:port")
	}

	// Extract and validate port
	parts := strings.Split(zmq.Address, ":")
	if len(parts) >= minZMQAddressParts {
		portStr := parts[len(parts)-1]
		port, err := strconv.Atoi(portStr)
		if err != nil || port < 1 || port > 65535 {
			result.AddError("ZMQ_ADDRESS", zmq.Address,
				"ZMQ port must be between 1 and 65535")
		}
	}

	// Validate timeout
	if zmq.Timeout < 1*time.Second {
		result.AddError("ZMQ_TIMEOUT", zmq.Timeout.String(),
			"ZMQ timeout must be at least 1 second")
	}
}

// validateMetricsConfig validates metrics-related configuration
func (v *Validator) validateMetricsConfig(metrics *MetricsConfig, result *ValidationResult) {
	// Validate port
	if metrics.Port < 1 || metrics.Port > 65535 {
		result.AddError("METRICS_PORT", strconv.Itoa(metrics.Port),
			"metrics port must be between 1 and 65535")
	}

	// Common ports warning
	commonPorts := []int{22, 23, 25, 53, 80, 110, 143, 443, 993, 995}
	for _, port := range commonPorts {
		if metrics.Port == port {
			result.AddWarning(fmt.Sprintf("Port %d is commonly used by other services", port))
			break
		}
	}

	// Validate path
	if !strings.HasPrefix(metrics.Path, "/") {
		result.AddError("METRICS_PATH", metrics.Path, "metrics path must start with /")
	}

	// Validate fetch interval
	if metrics.FetchInterval < 1*time.Second {
		result.AddError("FETCH_INTERVAL", metrics.FetchInterval.String(),
			"fetch interval must be at least 1 second")
	}

	if metrics.FetchInterval > 300*time.Second {
		result.AddWarning("Fetch interval is very high (>5 minutes)")
	}

	// Performance warnings
	if metrics.FetchInterval < 5*time.Second && v.strictMode {
		result.AddWarning("Very frequent fetch interval may impact Bitcoin node performance")
	}
}

// validateSecurityConfig validates security-related configuration
func (v *Validator) validateSecurityConfig(security *SecurityConfig, result *ValidationResult) {
	// TLS validation
	if security.TLSEnabled {
		if security.TLSCertFile == "" {
			result.AddError("TLS_CERT_FILE", security.TLSCertFile,
				"TLS certificate file must be specified when TLS is enabled")
		}

		if security.TLSKeyFile == "" {
			result.AddError("TLS_KEY_FILE", security.TLSKeyFile,
				"TLS key file must be specified when TLS is enabled")
		}
	}

	// Authentication validation
	if security.AuthEnabled {
		if security.AuthUsername == "" {
			result.AddError("AUTH_USERNAME", security.AuthUsername,
				"username must be specified when authentication is enabled")
		}

		if security.AuthPassword == "" {
			result.AddError("AUTH_PASSWORD", "***",
				"password must be specified when authentication is enabled")
		}

		if len(security.AuthPassword) < 8 && v.strictMode {
			result.AddWarning("Authentication password is shorter than 8 characters")
		}
	}

	// Rate limiting validation
	if security.RateLimitEnabled {
		if security.RateLimitRequests <= 0 {
			result.AddError("RATE_LIMIT_REQUESTS", strconv.Itoa(security.RateLimitRequests),
				"rate limit must be positive when rate limiting is enabled")
		}

		if security.RateLimitRequests > maxRateLimit {
			result.AddWarning(fmt.Sprintf("Rate limit is very high (>%d requests/minute)", maxRateLimit))
		}
	}

	// IP filtering validation
	if len(security.AllowedIPs) > 0 {
		for i, ip := range security.AllowedIPs {
			if !v.isValidIPOrCIDR(ip) {
				result.AddError("ALLOWED_IPS", ip,
					fmt.Sprintf("invalid IP address or CIDR notation at position %d", i))
			}
		}
	}

	// Security warnings
	if !security.TLSEnabled && v.strictMode {
		result.AddWarning("TLS is not enabled. Consider enabling for production")
	}

	if !security.AuthEnabled && v.strictMode {
		result.AddWarning("Authentication is not enabled. Consider enabling for production")
	}
}

// validateAppConfig validates application-level configuration
func (v *Validator) validateAppConfig(app *AppConfig, result *ValidationResult) {
	// Validate log level
	validLogLevels := []string{"trace", "debug", "info", "warn", "error", "fatal", "panic"}
	isValidLevel := false
	for _, level := range validLogLevels {
		if strings.EqualFold(app.LogLevel, level) {
			isValidLevel = true
			break
		}
	}

	if !isValidLevel {
		result.AddError("LOG_LEVEL", app.LogLevel,
			fmt.Sprintf("log level must be one of: %s", strings.Join(validLogLevels, ", ")))
	}

	// Environment-specific warnings
	if strings.EqualFold(app.LogLevel, "debug") && v.strictMode {
		result.AddWarning("Debug logging enabled. Consider using 'info' level for production")
	}

	// Validate environment
	if app.Environment != "" {
		validEnvs := []string{"development", "staging", "production"}
		isValidEnv := false
		for _, env := range validEnvs {
			if strings.EqualFold(app.Environment, env) {
				isValidEnv = true
				break
			}
		}

		if !isValidEnv {
			result.AddWarning(fmt.Sprintf("Unrecognized environment '%s'. Consider using: %s",
				app.Environment, strings.Join(validEnvs, ", ")))
		}
	}
}

// validateCrossFieldDependencies validates dependencies between configuration fields
func (v *Validator) validateCrossFieldDependencies(cfg *Config, result *ValidationResult) {
	// TLS and authentication compatibility
	type securityConfig struct{ tls, auth bool }
	switch (securityConfig{cfg.Security.TLSEnabled, cfg.Security.AuthEnabled}) {
	case securityConfig{true, true}:
		// This is good - both security measures enabled
	case securityConfig{true, false}:
		if v.strictMode {
			result.AddWarning("TLS enabled but authentication disabled. Consider enabling both")
		}
	case securityConfig{false, true}:
		if v.strictMode {
			result.AddWarning("Authentication enabled but TLS disabled. Credentials may be transmitted insecurely")
		}
	case securityConfig{false, false}:
		// Both disabled - no specific warning needed here
	}

	// Port conflicts
	if cfg.Metrics.Port == 443 && !cfg.Security.TLSEnabled {
		result.AddWarning("Using HTTPS port (443) without TLS enabled")
	}

	if cfg.Metrics.Port == 80 && cfg.Security.TLSEnabled {
		result.AddWarning("Using HTTP port (80) with TLS enabled")
	}

	// Performance considerations
	if cfg.Metrics.FetchInterval < 5*time.Second && cfg.ZMQ.Enabled {
		result.AddWarning("Fast fetch interval with ZMQ enabled may be redundant")
	}

	// Production readiness checks
	if v.strictMode && strings.EqualFold(cfg.App.Environment, "production") {
		if !cfg.Security.TLSEnabled {
			result.AddError("PRODUCTION_TLS", "false",
				"TLS must be enabled in production environment")
		}

		if !cfg.Security.AuthEnabled {
			result.AddError("PRODUCTION_AUTH", "false",
				"Authentication must be enabled in production environment")
		}

		if strings.EqualFold(cfg.App.LogLevel, "debug") || strings.EqualFold(cfg.App.LogLevel, "trace") {
			result.AddError("PRODUCTION_LOG_LEVEL", cfg.App.LogLevel,
				"Debug/trace logging should not be used in production")
		}
	}
}

// isValidIPOrCIDR validates IP address or CIDR notation
func (v *Validator) isValidIPOrCIDR(ipStr string) bool {
	// Simple regex for IP validation (IPv4)
	ipRegex := regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}(/\d{1,2})?$`)
	if !ipRegex.MatchString(ipStr) {
		return false
	}

	// Validate IP parts
	parts := strings.Split(strings.Split(ipStr, "/")[0], ".")
	for _, part := range parts {
		num, err := strconv.Atoi(part)
		if err != nil || num < 0 || num > 255 {
			return false
		}
	}

	// Validate CIDR if present
	if strings.Contains(ipStr, "/") {
		cidrParts := strings.Split(ipStr, "/")
		if len(cidrParts) != maxCIDRParts {
			return false
		}

		cidr, err := strconv.Atoi(cidrParts[1])
		if err != nil || cidr < 0 || cidr > 32 {
			return false
		}
	}

	return true
}

// ValidateAndReport validates configuration and returns formatted report
func (v *Validator) ValidateAndReport(cfg *Config) (valid bool, output string, err error) {
	result := v.ValidateConfig(cfg)

	if !result.Valid {
		var errorMsgs []string
		for _, e := range result.Errors {
			errorMsgs = append(errorMsgs, e.Error())
		}
		return false, "", errors.New(strings.Join(errorMsgs, "; "))
	}

	// Build report
	var report strings.Builder
	report.WriteString("Configuration validation passed\n")

	if len(result.Warnings) > 0 {
		report.WriteString("\nWarnings:\n")
		for _, warning := range result.Warnings {
			report.WriteString(fmt.Sprintf("  - %s\n", warning))
		}
	}

	return true, report.String(), nil
}
