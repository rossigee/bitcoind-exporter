package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/sirupsen/logrus"
)

// Security configuration constants
const (
	minPasswordLength = 8 // Minimum password length
	ipv4Parts         = 4 // Number of parts in IPv4 address
)

// SecurityConfig holds all security-related configuration
type SecurityConfig struct {
	// TLS Configuration
	TLSEnabled    bool   `env:"TLS_ENABLED" envDefault:"false"`
	TLSCertFile   string `env:"TLS_CERT_FILE"`
	TLSKeyFile    string `env:"TLS_KEY_FILE"`
	TLSMinVersion string `env:"TLS_MIN_VERSION" envDefault:"1.2"`

	// Authentication Configuration
	AuthEnabled  bool   `env:"AUTH_ENABLED" envDefault:"false"`
	AuthUsername string `env:"AUTH_USERNAME"`
	AuthPassword string `env:"AUTH_PASSWORD"`

	// Rate Limiting Configuration
	RateLimitEnabled   bool          `env:"RATE_LIMIT_ENABLED" envDefault:"false"`
	RateLimitRequests  int           `env:"RATE_LIMIT_REQUESTS" envDefault:"100"`
	RateLimitWindow    time.Duration `env:"RATE_LIMIT_WINDOW" envDefault:"1m"`
	RateLimitBlockTime time.Duration `env:"RATE_LIMIT_BLOCK_TIME" envDefault:"5m"`

	// Network Security
	AllowedIPs []string `env:"ALLOWED_IPS" envSeparator:","`
	DeniedIPs  []string `env:"DENIED_IPS" envSeparator:","`

	// Server Security
	ServerTimeouts ServerTimeoutConfig
}

// ServerTimeoutConfig holds server timeout configurations
type ServerTimeoutConfig struct {
	ReadTimeout       time.Duration `env:"READ_TIMEOUT" envDefault:"10s"`
	WriteTimeout      time.Duration `env:"WRITE_TIMEOUT" envDefault:"10s"`
	IdleTimeout       time.Duration `env:"IDLE_TIMEOUT" envDefault:"60s"`
	ReadHeaderTimeout time.Duration `env:"READ_HEADER_TIMEOUT" envDefault:"5s"`
}

var (
	securityLog = logrus.WithFields(logrus.Fields{
		"component": "security_config",
	})

	Security SecurityConfig
)

// LoadSecurityConfig loads and validates security configuration
func LoadSecurityConfig() error {
	if err := env.Parse(&Security); err != nil {
		return fmt.Errorf("failed to parse security config: %w", err)
	}

	// Load server timeouts
	if err := env.Parse(&Security.ServerTimeouts); err != nil {
		return fmt.Errorf("failed to parse server timeout config: %w", err)
	}

	// Validate configuration
	if err := validateSecurityConfig(); err != nil {
		return fmt.Errorf("security config validation failed: %w", err)
	}

	logSecurityConfig()
	return nil
}

// validateSecurityConfig validates the security configuration
func validateSecurityConfig() error {
	// TLS validation
	if Security.TLSEnabled {
		if Security.TLSCertFile == "" {
			return fmt.Errorf("TLS_CERT_FILE is required when TLS is enabled")
		}
		if Security.TLSKeyFile == "" {
			return fmt.Errorf("TLS_KEY_FILE is required when TLS is enabled")
		}

		// Check if cert and key files exist
		if _, err := os.Stat(Security.TLSCertFile); os.IsNotExist(err) {
			return fmt.Errorf("TLS cert file does not exist: %s", Security.TLSCertFile)
		}
		if _, err := os.Stat(Security.TLSKeyFile); os.IsNotExist(err) {
			return fmt.Errorf("TLS key file does not exist: %s", Security.TLSKeyFile)
		}

		// Validate TLS version
		validVersions := []string{"1.0", "1.1", "1.2", "1.3"}
		if !contains(validVersions, Security.TLSMinVersion) {
			return fmt.Errorf("invalid TLS_MIN_VERSION: %s, must be one of %v",
				Security.TLSMinVersion, validVersions)
		}
	}

	// Authentication validation
	if Security.AuthEnabled {
		if Security.AuthUsername == "" {
			return fmt.Errorf("AUTH_USERNAME is required when auth is enabled")
		}
		if Security.AuthPassword == "" {
			return fmt.Errorf("AUTH_PASSWORD is required when auth is enabled")
		}
		if len(Security.AuthPassword) < minPasswordLength {
			return fmt.Errorf("AUTH_PASSWORD must be at least %d characters long", minPasswordLength)
		}
	}

	// Rate limiting validation
	if Security.RateLimitEnabled {
		if Security.RateLimitRequests <= 0 {
			return fmt.Errorf("RATE_LIMIT_REQUESTS must be positive")
		}
		if Security.RateLimitWindow <= 0 {
			return fmt.Errorf("RATE_LIMIT_WINDOW must be positive")
		}
		if Security.RateLimitBlockTime <= 0 {
			return fmt.Errorf("RATE_LIMIT_BLOCK_TIME must be positive")
		}
	}

	// Validate IP addresses
	for _, ip := range Security.AllowedIPs {
		if !isValidIP(ip) {
			return fmt.Errorf("invalid IP address in ALLOWED_IPS: %s", ip)
		}
	}
	for _, ip := range Security.DeniedIPs {
		if !isValidIP(ip) {
			return fmt.Errorf("invalid IP address in DENIED_IPS: %s", ip)
		}
	}

	return nil
}

// logSecurityConfig logs the current security configuration (without sensitive data)
func logSecurityConfig() {
	securityLog.WithFields(logrus.Fields{
		"tls_enabled":         Security.TLSEnabled,
		"tls_min_version":     Security.TLSMinVersion,
		"auth_enabled":        Security.AuthEnabled,
		"auth_username":       Security.AuthUsername,
		"rate_limit_enabled":  Security.RateLimitEnabled,
		"rate_limit_requests": Security.RateLimitRequests,
		"rate_limit_window":   Security.RateLimitWindow,
		"allowed_ips_count":   len(Security.AllowedIPs),
		"denied_ips_count":    len(Security.DeniedIPs),
	}).Info("Security configuration loaded")

	// Warn about security issues
	if !Security.TLSEnabled {
		securityLog.Warn("TLS is disabled - metrics will be transmitted in plain text")
	}
	if !Security.AuthEnabled {
		securityLog.Warn("Authentication is disabled - metrics endpoint is publicly accessible")
	}
	if !Security.RateLimitEnabled {
		securityLog.Warn("Rate limiting is disabled - no protection against DoS attacks")
	}
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func isValidIP(ip string) bool {
	// Basic IP validation - accepts IPv4, IPv6, and CIDR notation
	if ip == "" {
		return false
	}

	// Remove CIDR suffix if present
	if idx := strings.Index(ip, "/"); idx != -1 {
		ip = ip[:idx]
	}

	// Check for IPv4
	parts := strings.Split(ip, ".")
	if len(parts) == ipv4Parts {
		for _, part := range parts {
			if num, err := strconv.Atoi(part); err != nil || num < 0 || num > 255 {
				return false
			}
		}
		return true
	}

	// Check for IPv6 (basic check)
	if strings.Contains(ip, ":") && len(ip) >= 2 {
		return true // Simplified IPv6 validation
	}

	return false
}

// GetSecurityHeaders returns recommended security headers
func GetSecurityHeaders() map[string]string {
	return map[string]string{
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
		"X-XSS-Protection":          "1; mode=block",
		"Referrer-Policy":           "strict-origin-when-cross-origin",
		"Content-Security-Policy":   "default-src 'self'",
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
	}
}

// IsIPAllowed checks if an IP address is allowed
func IsIPAllowed(clientIP string) bool {
	// If no restrictions configured, allow all
	if len(Security.AllowedIPs) == 0 && len(Security.DeniedIPs) == 0 {
		return true
	}

	// Check denied IPs first
	for _, deniedIP := range Security.DeniedIPs {
		if matchesIP(clientIP, deniedIP) {
			return false
		}
	}

	// If no allowed IPs configured, allow (unless denied above)
	if len(Security.AllowedIPs) == 0 {
		return true
	}

	// Check allowed IPs
	for _, allowedIP := range Security.AllowedIPs {
		if matchesIP(clientIP, allowedIP) {
			return true
		}
	}

	return false
}

// matchesIP checks if a client IP matches an allowed/denied IP pattern
func matchesIP(clientIP, pattern string) bool {
	// Handle IPv6 addresses with brackets and ports like [::1]:8080
	if strings.HasPrefix(clientIP, "[") {
		if idx := strings.LastIndex(clientIP, "]:"); idx != -1 {
			clientIP = clientIP[1:idx] // Remove brackets and port
		} else if strings.HasSuffix(clientIP, "]") {
			clientIP = clientIP[1 : len(clientIP)-1] // Remove brackets only
		}
	} else {
		// For IPv4, remove port if present (but be careful with IPv6)
		if strings.Count(clientIP, ":") == 1 {
			if idx := strings.LastIndex(clientIP, ":"); idx != -1 {
				clientIP = clientIP[:idx]
			}
		}
	}

	// Exact match
	if clientIP == pattern {
		return true
	}

	// CIDR notation matching
	if strings.Contains(pattern, "/") {
		_, cidr, err := net.ParseCIDR(pattern)
		if err != nil {
			return false
		}
		
		ip := net.ParseIP(clientIP)
		if ip == nil {
			return false
		}
		
		return cidr.Contains(ip)
	}

	return false
}
