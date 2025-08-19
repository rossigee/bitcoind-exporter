package config

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfiguration_ValidConfig(t *testing.T) {
	// Set up environment variables
	oldEnv := os.Environ()
	defer func() {
		os.Clearenv()
		for _, env := range oldEnv {
			if len(env) > 0 {
				// Parse KEY=VALUE format
				parts := strings.SplitN(env, "=", 2)
				if len(parts) == 2 {
					_ = os.Setenv(parts[0], parts[1])
				}
			}
		}
	}()

	os.Clearenv()
	_ = os.Setenv("RPC_ADDRESS", "http://127.0.0.1:8332")
	_ = os.Setenv("RPC_USER", "testuser")
	_ = os.Setenv("RPC_PASS", "testpass")
	_ = os.Setenv("ZMQ_ADDRESS", "127.0.0.1:28333")
	_ = os.Setenv("FETCH_INTERVAL", "15")
	_ = os.Setenv("METRIC_PORT", "3001")
	_ = os.Setenv("LOG_LEVEL", "debug")

	// Load configuration
	loadConfiguration()

	// Verify configuration
	assert.Equal(t, "http://127.0.0.1:8332", C.RPCAddress)
	assert.Equal(t, "testuser", C.RPCUser)
	assert.Equal(t, "testpass", C.RPCPass)
	assert.Equal(t, "127.0.0.1:28333", C.ZmqAddress)
	assert.Equal(t, 15, C.FetchInterval)
	assert.Equal(t, 3001, C.MetricPort)
	assert.Equal(t, "debug", C.LogLevel)
}

func TestLoadConfiguration_CookieAuth(t *testing.T) {
	oldEnv := os.Environ()
	defer func() {
		os.Clearenv()
		for _, env := range oldEnv {
			if len(env) > 0 {
				parts := strings.SplitN(env, "=", 2)
				if len(parts) == 2 {
					_ = os.Setenv(parts[0], parts[1])
				}
			}
		}
	}()

	os.Clearenv()
	_ = os.Setenv("RPC_ADDRESS", "http://127.0.0.1:8332")
	_ = os.Setenv("RPC_COOKIE_FILE", "/path/to/.cookie")

	// This test verifies the configuration loads without panicking
	// The actual cookie file validation happens in the client
	loadConfiguration()

	assert.Equal(t, "http://127.0.0.1:8332", C.RPCAddress)
	assert.Equal(t, "/path/to/.cookie", C.RPCCookieFile)
	assert.Empty(t, C.RPCUser)
	assert.Empty(t, C.RPCPass)
}

func TestLoadConfiguration_DefaultValues(t *testing.T) {
	oldEnv := os.Environ()
	defer func() {
		os.Clearenv()
		for _, env := range oldEnv {
			if len(env) > 0 {
				parts := strings.SplitN(env, "=", 2)
				if len(parts) == 2 {
					_ = os.Setenv(parts[0], parts[1])
				}
			}
		}
	}()

	os.Clearenv()
	_ = os.Setenv("RPC_ADDRESS", "http://127.0.0.1:8332")
	_ = os.Setenv("RPC_USER", "testuser")
	_ = os.Setenv("RPC_PASS", "testpass")

	loadConfiguration()

	// Check default values
	assert.Equal(t, 10, C.FetchInterval) // Default from envDefault tag
	assert.Equal(t, 3000, C.MetricPort)  // Default from envDefault tag
	assert.Equal(t, "info", C.LogLevel)  // Default from envDefault tag
}

func TestConfiguration_Validation(t *testing.T) {
	tests := getValidationTestCases()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runValidationTest(t, tt)
		})
	}
}

// getValidationTestCases returns test cases for configuration validation
func getValidationTestCases() []struct {
	name        string
	rpcUser     string
	rpcPass     string
	cookieFile  string
	shouldPanic bool
} {
	return []struct {
		name        string
		rpcUser     string
		rpcPass     string
		cookieFile  string
		shouldPanic bool
	}{
		{
			name:        "valid user/pass auth",
			rpcUser:     "user",
			rpcPass:     "pass",
			cookieFile:  "",
			shouldPanic: false,
		},
		{
			name:        "valid cookie auth",
			rpcUser:     "",
			rpcPass:     "",
			cookieFile:  "/path/to/cookie",
			shouldPanic: false,
		},
		{
			name:        "invalid - no auth",
			rpcUser:     "",
			rpcPass:     "",
			cookieFile:  "",
			shouldPanic: true,
		},
		{
			name:        "invalid - partial user/pass",
			rpcUser:     "user",
			rpcPass:     "",
			cookieFile:  "",
			shouldPanic: true,
		},
	}
}

// runValidationTest executes a single validation test case
func runValidationTest(t *testing.T, tt struct {
	name        string
	rpcUser     string
	rpcPass     string
	cookieFile  string
	shouldPanic bool
}) {
	oldEnv := os.Environ()
	defer restoreEnvironment(oldEnv)

	setupTestEnvironment(tt.rpcUser, tt.rpcPass, tt.cookieFile)

	if tt.shouldPanic {
		testPanicScenario(t)
	} else {
		assert.NotPanics(t, func() {
			loadConfiguration()
		})
	}
}


// setupTestEnvironment sets up environment variables for testing
func setupTestEnvironment(rpcUser, rpcPass, cookieFile string) {
	os.Clearenv()
	_ = os.Setenv("RPC_ADDRESS", "http://127.0.0.1:8332")

	if rpcUser != "" {
		_ = os.Setenv("RPC_USER", rpcUser)
	}
	if rpcPass != "" {
		_ = os.Setenv("RPC_PASS", rpcPass)
	}
	if cookieFile != "" {
		_ = os.Setenv("RPC_COOKIE_FILE", cookieFile)
	}
}

// testPanicScenario tests that configuration loading panics as expected
func testPanicScenario(t *testing.T) {
	done := make(chan bool, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- true
			} else {
				done <- false
			}
		}()
		loadConfiguration()
	}()

	panicked := <-done
	assert.True(t, panicked, "Expected configuration to panic")
}

// Test environment variable parsing edge cases
func TestEnvironmentVariableParsing(t *testing.T) {
	tests := []struct {
		name          string
		envValue      string
		expectedValue int
		shouldDefault bool
	}{
		{
			name:          "valid integer",
			envValue:      "25",
			expectedValue: 25,
			shouldDefault: false,
		},
		{
			name:          "empty string uses default",
			envValue:      "",
			expectedValue: 10, // Default value
			shouldDefault: true,
		},
		{
			name:          "zero value",
			envValue:      "0",
			expectedValue: 0,
			shouldDefault: false,
		},
		{
			name:          "negative value",
			envValue:      "-5",
			expectedValue: -5,
			shouldDefault: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldEnv := os.Environ()
			defer func() {
				os.Clearenv()
				for _, env := range oldEnv {
					if len(env) > 0 {
						parts := strings.SplitN(env, "=", 2)
						if len(parts) == 2 {
							_ = os.Setenv(parts[0], parts[1])
						}
					}
				}
			}()

			os.Clearenv()
			_ = os.Setenv("RPC_ADDRESS", "http://127.0.0.1:8332")
			_ = os.Setenv("RPC_USER", "test")
			_ = os.Setenv("RPC_PASS", "test")

			if !tt.shouldDefault {
				_ = os.Setenv("FETCH_INTERVAL", tt.envValue)
			}

			loadConfiguration()

			assert.Equal(t, tt.expectedValue, C.FetchInterval)
		})
	}
}
