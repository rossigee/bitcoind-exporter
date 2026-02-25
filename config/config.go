package config

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/sirupsen/logrus"
)

// RPCConfig holds RPC connection configuration
type RPCConfig struct {
	Address    string        `env:"RPC_ADDRESS,required"`
	User       string        `env:"RPC_USER"`
	Pass       string        `env:"RPC_PASS"`
	CookieFile string        `env:"RPC_COOKIE_FILE"`
	Timeout    time.Duration `env:"RPC_TIMEOUT" envDefault:"30s"`
}

// ZMQConfig holds ZMQ connection configuration
type ZMQConfig struct {
	Enabled bool          `env:"ZMQ_ENABLED" envDefault:"false"`
	Address string        `env:"ZMQ_ADDRESS"`
	Timeout time.Duration `env:"ZMQ_TIMEOUT" envDefault:"10s"`
}

// MetricsConfig holds metrics server configuration
type MetricsConfig struct {
	Port          int           `env:"METRICS_PORT" envDefault:"3000"`
	Path          string        `env:"METRICS_PATH" envDefault:"/metrics"`
	FetchInterval time.Duration `env:"FETCH_INTERVAL" envDefault:"10s"`
}

// AppConfig holds application-level configuration
type AppConfig struct {
	LogLevel    string `env:"LOG_LEVEL" envDefault:"info"`
	Environment string `env:"ENVIRONMENT" envDefault:"development"`
}

// Config represents the complete application configuration
type Config struct {
	RPC      RPCConfig
	ZMQ      ZMQConfig
	Metrics  MetricsConfig
	Security SecurityConfig
	App      AppConfig
}

// Legacy config struct for backward compatibility
type config struct {
	RPCAddress string `env:"RPC_ADDRESS,required"`

	RPCUser       string `env:"RPC_USER"`
	RPCPass       string `env:"RPC_PASS"`
	RPCCookieFile string `env:"RPC_COOKIE_FILE"`

	ZmqAddress string `env:"ZMQ_ADDRESS"`

	FetchInterval int `env:"FETCH_INTERVAL" envDefault:"10"`
	MetricPort    int `env:"METRIC_PORT" envDefault:"3000"`

	LogLevel string `env:"LOG_LEVEL" envDefault:"info"`
}

var (
	log = logrus.WithFields(logrus.Fields{
		"prefix": "config",
	})

	// Legacy global config for backward compatibility
	C config

	// New structured configuration
	GlobalConfig  *Config
	validator     *Validator
	healthMonitor *ConfigHealthMonitor
)

// InitializeConfig loads and validates the complete configuration
// This replaces the init function to avoid automatic initialization
func InitializeConfig() {
	loadConfiguration()
	if err := LoadSecurityConfig(); err != nil {
		log.WithError(err).Fatal("Failed to load security configuration")
	}
}

// LoadConfig loads and validates the complete configuration
func LoadConfig() (*Config, error) {
	// Load individual config sections
	var rpcConfig RPCConfig
	if err := env.Parse(&rpcConfig); err != nil {
		return nil, fmt.Errorf("failed to parse RPC config: %w", err)
	}

	var zmqConfig ZMQConfig
	if err := env.Parse(&zmqConfig); err != nil {
		return nil, fmt.Errorf("failed to parse ZMQ config: %w", err)
	}

	var metricsConfig MetricsConfig
	if err := env.Parse(&metricsConfig); err != nil {
		return nil, fmt.Errorf("failed to parse metrics config: %w", err)
	}

	var securityConfig SecurityConfig
	if err := env.Parse(&securityConfig); err != nil {
		return nil, fmt.Errorf("failed to parse security config: %w", err)
	}

	var appConfig AppConfig
	if err := env.Parse(&appConfig); err != nil {
		return nil, fmt.Errorf("failed to parse app config: %w", err)
	}

	// Construct complete configuration
	cfg := &Config{
		RPC:      rpcConfig,
		ZMQ:      zmqConfig,
		Metrics:  metricsConfig,
		Security: securityConfig,
		App:      appConfig,
	}

	// Validate configuration
	if err := ValidateConfig(cfg); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Set global config
	GlobalConfig = cfg

	// Initialize health monitor
	healthMonitor = NewConfigHealthMonitor(cfg, log)

	log.Info("Configuration loaded and validated successfully")
	return cfg, nil
}

// ValidateConfig validates the configuration and returns detailed errors
func ValidateConfig(cfg *Config) error {
	if validator == nil {
		// Determine if we should use strict mode
		strictMode := strings.EqualFold(cfg.App.Environment, "production")
		validator = NewValidator(strictMode)
	}

	valid, report, err := validator.ValidateAndReport(cfg)
	if !valid {
		return err
	}

	if report != "" {
		log.Info("Configuration validation report:")
		for _, line := range strings.Split(report, "\n") {
			if strings.TrimSpace(line) != "" {
				log.Info(line)
			}
		}
	}

	return nil
}

// StartHealthMonitoring starts configuration health monitoring
func StartHealthMonitoring(ctx context.Context) error {
	if healthMonitor == nil {
		return fmt.Errorf("health monitor not initialized - call LoadConfig first")
	}

	return healthMonitor.Start(ctx)
}

// StopHealthMonitoring stops configuration health monitoring
func StopHealthMonitoring() {
	if healthMonitor != nil {
		healthMonitor.Stop()
	}
}

// GetHealthStatus returns current configuration health status
func GetHealthStatus() map[string]interface{} {
	if healthMonitor == nil {
		return map[string]interface{}{
			"error": "health monitor not initialized",
		}
	}

	return healthMonitor.GetHealthSummary()
}

// IsConfigHealthy returns whether configuration-dependent services are healthy
func IsConfigHealthy() bool {
	if healthMonitor == nil {
		return false
	}

	return healthMonitor.IsHealthy()
}

// ReloadConfig reloads and revalidates configuration
func ReloadConfig() error {
	log.Info("Reloading configuration...")

	newConfig, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to reload configuration: %w", err)
	}

	// Stop old health monitor
	if healthMonitor != nil {
		healthMonitor.Stop()
	}

	// Update global config
	GlobalConfig = newConfig

	log.Info("Configuration reloaded successfully")
	return nil
}

// Legacy function for backward compatibility
func loadConfiguration() {
	if config, err := env.ParseAs[config](); err == nil {
		log.Debug("Legacy configuration loaded")
		C = config
	} else {
		log.Error("Failed to parse configuration:", err)
		panic(err)
	}

	if C.RPCUser == "" && C.RPCPass == "" && C.RPCCookieFile == "" {
		log.Error("RPC_USER and RPC_PASS or RPC_COOKIE_FILE must be set")
		panic("RPC_USER and RPC_PASS or RPC_COOKIE_FILE must be set")
	}

	// Validate that if user/pass auth is being used, both must be provided
	if (C.RPCUser != "" || C.RPCPass != "") && (C.RPCUser == "" || C.RPCPass == "") && C.RPCCookieFile == "" {
		log.Error("Both RPC_USER and RPC_PASS must be provided when using username/password authentication")
		panic("Both RPC_USER and RPC_PASS must be provided when using username/password authentication")
	}
}

// GetLegacyConfig returns the legacy config structure for backward compatibility
func GetLegacyConfig() config {
	return C
}

// ConvertToLegacyConfig converts new Config to legacy config format
func ConvertToLegacyConfig(cfg *Config) config {
	return config{
		RPCAddress:    cfg.RPC.Address,
		RPCUser:       cfg.RPC.User,
		RPCPass:       cfg.RPC.Pass,
		RPCCookieFile: cfg.RPC.CookieFile,
		ZmqAddress:    cfg.ZMQ.Address,
		FetchInterval: int(cfg.Metrics.FetchInterval.Seconds()),
		MetricPort:    cfg.Metrics.Port,
		LogLevel:      cfg.App.LogLevel,
	}
}
