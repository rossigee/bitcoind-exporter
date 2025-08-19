package security

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewSecureServer(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	tlsConfig := &TLSConfig{
		Enabled:    true,
		CertFile:   "/path/to/cert.pem",
		KeyFile:    "/path/to/key.pem",
		MinVersion: "1.2",
	}

	server := NewSecureServer(":8080", handler, tlsConfig)

	assert.NotNil(t, server)
	assert.NotNil(t, server.server)
	assert.NotNil(t, server.logger)
	assert.Equal(t, ":8080", server.server.Addr)
	assert.Equal(t, defaultReadTimeout, server.server.ReadTimeout)
	assert.Equal(t, defaultWriteTimeout, server.server.WriteTimeout)
	assert.Equal(t, defaultIdleTimeout, server.server.IdleTimeout)
	assert.Equal(t, defaultReadHeaderTimeout, server.server.ReadHeaderTimeout)
}

func TestCreateTLSConfig(t *testing.T) {
	tests := getTLSConfigTestCases()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runTLSConfigTest(t, tt)
		})
	}
}

func getTLSConfigTestCases() []struct {
	name           string
	config         *TLSConfig
	expectedResult bool
	expectedMin    uint16
} {
	basicTests := getBasicTLSTests()
	versionTests := getTLSVersionTests()
	return append(basicTests, versionTests...)
}

func getBasicTLSTests() []struct {
	name           string
	config         *TLSConfig
	expectedResult bool
	expectedMin    uint16
} {
	return []struct {
		name           string
		config         *TLSConfig
		expectedResult bool
		expectedMin    uint16
	}{
		{
			name: "TLS disabled",
			config: &TLSConfig{
				Enabled: false,
			},
			expectedResult: false,
		},
		{
			name: "Unknown version defaults to 1.2",
			config: &TLSConfig{
				Enabled:    true,
				MinVersion: "unknown",
			},
			expectedResult: true,
			expectedMin:    tls.VersionTLS12,
		},
	}
}

func getTLSVersionTests() []struct {
	name           string
	config         *TLSConfig
	expectedResult bool
	expectedMin    uint16
} {
	return []struct {
		name           string
		config         *TLSConfig
		expectedResult bool
		expectedMin    uint16
	}{
		{
			name: "TLS 1.0",
			config: &TLSConfig{
				Enabled:    true,
				MinVersion: "1.0",
			},
			expectedResult: true,
			expectedMin:    tls.VersionTLS10,
		},
		{
			name: "TLS 1.1",
			config: &TLSConfig{
				Enabled:    true,
				MinVersion: "1.1",
			},
			expectedResult: true,
			expectedMin:    tls.VersionTLS11,
		},
		{
			name: "TLS 1.2 (default)",
			config: &TLSConfig{
				Enabled:    true,
				MinVersion: "1.2",
			},
			expectedResult: true,
			expectedMin:    tls.VersionTLS12,
		},
		{
			name: "TLS 1.3",
			config: &TLSConfig{
				Enabled:    true,
				MinVersion: "1.3",
			},
			expectedResult: true,
			expectedMin:    tls.VersionTLS13,
		},
	}
}

func runTLSConfigTest(t *testing.T, tt struct {
	name           string
	config         *TLSConfig
	expectedResult bool
	expectedMin    uint16
}) {
	result := createTLSConfig(tt.config)

	if !tt.expectedResult {
		assert.Nil(t, result)
	} else {
		assert.NotNil(t, result)
		assert.Equal(t, tt.expectedMin, result.MinVersion)
		assert.False(t, result.InsecureSkipVerify)
		assert.NotEmpty(t, result.CipherSuites)
		assert.NotEmpty(t, result.CurvePreferences)
	}
}

func TestAddSecurityHeaders(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	secureHandler := addSecurityHeaders(handler)

	tests := []struct {
		name     string
		path     string
		expected map[string]string
	}{
		{
			name: "Regular endpoint",
			path: "/health",
			expected: map[string]string{
				"X-Content-Type-Options":   "nosniff",
				"X-Frame-Options":          "DENY",
				"X-XSS-Protection":         "1; mode=block",
				"Referrer-Policy":          "strict-origin-when-cross-origin",
				"Content-Security-Policy":  "default-src 'self'",
			},
		},
		{
			name: "Metrics endpoint",
			path: "/metrics",
			expected: map[string]string{
				"X-Content-Type-Options":   "nosniff",
				"X-Frame-Options":          "DENY",
				"X-XSS-Protection":         "1; mode=block",
				"Referrer-Policy":          "strict-origin-when-cross-origin",
				"Content-Security-Policy":  "default-src 'self'",
				"Cache-Control":            "no-cache, no-store, must-revalidate",
				"Pragma":                   "no-cache",
				"Expires":                  "0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			rr := httptest.NewRecorder()

			secureHandler.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusOK, rr.Code)
			for header, value := range tt.expected {
				assert.Equal(t, value, rr.Header().Get(header))
			}
		})
	}
}

func TestNewAuthMiddleware(t *testing.T) {
	username := "testuser"
	password := "testpass"

	middleware := NewAuthMiddleware(username, password)

	assert.NotNil(t, middleware)
	assert.Equal(t, username, middleware.username)
	assert.Equal(t, password, middleware.password)
	assert.NotNil(t, middleware.logger)
}

func TestAuthMiddleware_Middleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	tests := getAuthMiddlewareTestCases()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runAuthMiddlewareTest(t, tt, handler)
		})
	}
}

func getAuthMiddlewareTestCases() []struct {
	name           string
	username       string
	password       string
	authHeader     string
	expectedStatus int
	expectAuth     bool
} {
	return []struct {
		name           string
		username       string
		password       string
		authHeader     string
		expectedStatus int
		expectAuth     bool
	}{
		{
			name:           "No auth configured",
			username:       "",
			password:       "",
			authHeader:     "",
			expectedStatus: http.StatusOK,
			expectAuth:     false,
		},
		{
			name:           "Valid credentials",
			username:       "admin",
			password:       "secret",
			authHeader:     "Basic YWRtaW46c2VjcmV0", // admin:secret
			expectedStatus: http.StatusOK,
			expectAuth:     false,
		},
		{
			name:           "Invalid credentials",
			username:       "admin",
			password:       "secret",
			authHeader:     "Basic YWRtaW46d3Jvbmc=", // admin:wrong
			expectedStatus: http.StatusUnauthorized,
			expectAuth:     false, // WWW-Authenticate is only set for missing auth, not invalid
		},
		{
			name:           "Missing auth header",
			username:       "admin",
			password:       "secret",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			expectAuth:     true,
		},
		{
			name:           "Invalid auth format",
			username:       "admin",
			password:       "secret",
			authHeader:     "Bearer token123",
			expectedStatus: http.StatusUnauthorized,
			expectAuth:     true,
		},
	}
}

func runAuthMiddlewareTest(t *testing.T, tt struct {
	name           string
	username       string
	password       string
	authHeader     string
	expectedStatus int
	expectAuth     bool
}, handler http.Handler) {
	middleware := NewAuthMiddleware(tt.username, tt.password)
	authHandler := middleware.Middleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	if tt.authHeader != "" {
		req.Header.Set("Authorization", tt.authHeader)
	}
	rr := httptest.NewRecorder()

	authHandler.ServeHTTP(rr, req)

	assert.Equal(t, tt.expectedStatus, rr.Code)
	if tt.expectAuth {
		assert.Equal(t, `Basic realm="Restricted"`, rr.Header().Get("WWW-Authenticate"))
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expected   string
	}{
		{
			name: "X-Forwarded-For header",
			headers: map[string]string{
				"X-Forwarded-For": "192.168.1.100",
			},
			remoteAddr: "10.0.0.1:12345",
			expected:   "192.168.1.100",
		},
		{
			name: "X-Real-IP header",
			headers: map[string]string{
				"X-Real-IP": "192.168.1.200",
			},
			remoteAddr: "10.0.0.1:12345",
			expected:   "192.168.1.200",
		},
		{
			name: "Both headers - X-Forwarded-For takes precedence",
			headers: map[string]string{
				"X-Forwarded-For": "192.168.1.100",
				"X-Real-IP":       "192.168.1.200",
			},
			remoteAddr: "10.0.0.1:12345",
			expected:   "192.168.1.100",
		},
		{
			name:       "Fall back to RemoteAddr",
			headers:    map[string]string{},
			remoteAddr: "10.0.0.1:12345",
			expected:   "10.0.0.1:12345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			for header, value := range tt.headers {
				req.Header.Set(header, value)
			}

			result := getClientIP(req)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewRateLimiter(t *testing.T) {
	config := &RateLimitConfig{
		Enabled:    true,
		Requests:   100,
		WindowSize: time.Minute,
		BlockTime:  5 * time.Minute,
	}

	limiter := NewRateLimiter(config)

	assert.NotNil(t, limiter)
	assert.Equal(t, config, limiter.config)
	assert.NotNil(t, limiter.clients)
	assert.NotNil(t, limiter.logger)
}

func TestRateLimiter_Middleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	t.Run("Rate limiting disabled", func(t *testing.T) {
		testRateLimitingDisabled(t, handler)
	})

	t.Run("Rate limiting enabled - under limit", func(t *testing.T) {
		testRateLimitingUnderLimit(t, handler)
	})

	t.Run("Rate limiting enabled - over limit", func(t *testing.T) {
		testRateLimitingOverLimit(t, handler)
	})
}

func testRateLimitingDisabled(t *testing.T, handler http.Handler) {
	config := &RateLimitConfig{Enabled: false}
	limiter := NewRateLimiter(config)
	rateLimitedHandler := limiter.Middleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	rateLimitedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func testRateLimitingUnderLimit(t *testing.T, handler http.Handler) {
	config := &RateLimitConfig{
		Enabled:    true,
		Requests:   5,
		WindowSize: time.Minute,
		BlockTime:  5 * time.Minute,
	}
	limiter := NewRateLimiter(config)
	rateLimitedHandler := limiter.Middleware(handler)

	// Make 3 requests (under limit of 5)
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		rr := httptest.NewRecorder()

		rateLimitedHandler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	}
}

func testRateLimitingOverLimit(t *testing.T, handler http.Handler) {
	config := &RateLimitConfig{
		Enabled:    true,
		Requests:   2,
		WindowSize: time.Minute,
		BlockTime:  5 * time.Minute,
	}
	limiter := NewRateLimiter(config)
	rateLimitedHandler := limiter.Middleware(handler)

	clientIP := "192.168.1.100:12345"

	// Make requests up to the limit
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = clientIP
		rr := httptest.NewRecorder()

		rateLimitedHandler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	}

	// Next request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = clientIP
	rr := httptest.NewRecorder()

	rateLimitedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusTooManyRequests, rr.Code)
	assert.Contains(t, rr.Body.String(), "Rate limit exceeded")

	// Subsequent requests should be blocked
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = clientIP
	rr = httptest.NewRecorder()

	rateLimitedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusTooManyRequests, rr.Code)
}

func TestCreateSecureHandler(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	config := &SecurityConfig{
		TLS: &TLSConfig{
			Enabled:    true,
			MinVersion: "1.2",
		},
		Auth: &AuthConfig{
			Enabled:  true,
			Username: "admin",
			Password: "secret",
		},
		RateLimit: &RateLimitConfig{
			Enabled:    true,
			Requests:   100,
			WindowSize: time.Minute,
			BlockTime:  5 * time.Minute,
		},
	}

	secureHandler := CreateSecureHandler(handler, config)

	assert.NotNil(t, secureHandler)

	// Test with valid auth
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Basic YWRtaW46c2VjcmV0") // admin:secret
	rr := httptest.NewRecorder()

	secureHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	// Verify security headers are applied
	assert.Equal(t, "nosniff", rr.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", rr.Header().Get("X-Frame-Options"))

	// Test with invalid auth
	req = httptest.NewRequest("GET", "/test", nil)
	rr = httptest.NewRecorder()

	secureHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestCreateSecureHandler_DisabledFeatures(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	config := &SecurityConfig{
		TLS: &TLSConfig{
			Enabled: false,
		},
		Auth: &AuthConfig{
			Enabled: false,
		},
		RateLimit: &RateLimitConfig{
			Enabled: false,
		},
	}

	secureHandler := CreateSecureHandler(handler, config)

	assert.NotNil(t, secureHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	secureHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	// Security headers should still be applied
	assert.Equal(t, "nosniff", rr.Header().Get("X-Content-Type-Options"))
}

func TestRateLimiter_WindowReset(t *testing.T) {
	config := &RateLimitConfig{
		Enabled:    true,
		Requests:   2,
		WindowSize: 100 * time.Millisecond, // Short window for testing
		BlockTime:  5 * time.Minute,
	}
	limiter := NewRateLimiter(config)

	clientIP := "192.168.1.100"

	// First window - use up the limit
	assert.False(t, limiter.isRateLimited(clientIP)) // Request 1
	assert.False(t, limiter.isRateLimited(clientIP)) // Request 2
	assert.True(t, limiter.isRateLimited(clientIP))  // Request 3 - over limit

	// Wait for window to reset
	time.Sleep(150 * time.Millisecond)

	// New window - should be able to make requests again
	assert.False(t, limiter.isRateLimited(clientIP)) // Request 1 in new window
}

func TestSecureServer_ListenAndServe_HTTPS(t *testing.T) {
	// This test would require actual certificate files to test HTTPS
	// For now, we'll test the logic paths only
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tlsConfig := &TLSConfig{
		Enabled:  true,
		CertFile: "/nonexistent/cert.pem",
		KeyFile:  "/nonexistent/key.pem",
	}

	server := NewSecureServer(":0", handler, tlsConfig)

	// We can't actually start the server in tests without valid certs
	// This test primarily verifies the configuration is set up correctly
	assert.NotNil(t, server.server.TLSConfig)
}

func TestSecureServer_ListenAndServe_HTTP(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tlsConfig := &TLSConfig{
		Enabled: false,
	}

	server := NewSecureServer(":0", handler, tlsConfig)

	// Test with TLS disabled - should prepare for HTTP
	assert.Nil(t, server.server.TLSConfig)
}

// Test edge cases and error paths
func TestAuthMiddleware_EdgeCases(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := NewAuthMiddleware("admin", "secret")
	authHandler := middleware.Middleware(handler)

	t.Run("Malformed basic auth", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Basic invalid-base64")
		rr := httptest.NewRecorder()

		authHandler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("Basic auth with colon in password", func(t *testing.T) {
		// Test handling of credentials containing colons
		middleware := NewAuthMiddleware("user", "pass:word")
		authHandler := middleware.Middleware(handler)

		// user:pass:word -> dXNlcjpwYXNzOndvcmQ=
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Basic dXNlcjpwYXNzOndvcmQ=")
		rr := httptest.NewRecorder()

		authHandler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})
}