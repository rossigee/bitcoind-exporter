package security

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// Server timeout constants
const (
	defaultReadTimeout       = 10 * time.Second // HTTP read timeout
	defaultWriteTimeout      = 10 * time.Second // HTTP write timeout
	defaultIdleTimeout       = 60 * time.Second // HTTP idle timeout
	defaultReadHeaderTimeout = 5 * time.Second  // HTTP read header timeout
)

// TLSConfig holds TLS configuration options
type TLSConfig struct {
	Enabled    bool   `env:"TLS_ENABLED" envDefault:"false"`
	CertFile   string `env:"TLS_CERT_FILE"`
	KeyFile    string `env:"TLS_KEY_FILE"`
	MinVersion string `env:"TLS_MIN_VERSION" envDefault:"1.2"`
}

// SecureServer wraps http.Server with security configurations
type SecureServer struct {
	server *http.Server
	logger *logrus.Entry
}

// NewSecureServer creates a new secure HTTP server
func NewSecureServer(addr string, handler http.Handler, tlsConfig *TLSConfig) *SecureServer {
	server := &http.Server{
		Addr:    addr,
		Handler: handler,

		// Security timeouts
		ReadTimeout:       defaultReadTimeout,
		WriteTimeout:      defaultWriteTimeout,
		IdleTimeout:       defaultIdleTimeout,
		ReadHeaderTimeout: defaultReadHeaderTimeout,

		// Security headers
		TLSConfig: createTLSConfig(tlsConfig),
	}

	// Add security middleware
	server.Handler = addSecurityHeaders(handler)

	return &SecureServer{
		server: server,
		logger: logrus.WithFields(logrus.Fields{
			"component": "secure_server",
		}),
	}
}

// ListenAndServe starts the server with appropriate protocol
func (s *SecureServer) ListenAndServe(tlsConfig *TLSConfig) error {
	if tlsConfig.Enabled && tlsConfig.CertFile != "" && tlsConfig.KeyFile != "" {
		s.logger.Info("Starting HTTPS server")
		return s.server.ListenAndServeTLS(tlsConfig.CertFile, tlsConfig.KeyFile)
	}

	s.logger.Warn("Starting HTTP server (TLS disabled)")
	return s.server.ListenAndServe()
}

// createTLSConfig creates a secure TLS configuration
func createTLSConfig(config *TLSConfig) *tls.Config {
	if !config.Enabled {
		return nil
	}

	var minVersion uint16 = tls.VersionTLS12
	switch config.MinVersion {
	case "1.0":
		minVersion = tls.VersionTLS10
	case "1.1":
		minVersion = tls.VersionTLS11
	case "1.2":
		minVersion = tls.VersionTLS12
	case "1.3":
		minVersion = tls.VersionTLS13
	}

	return &tls.Config{
		MinVersion: minVersion,

		// Secure cipher suites (TLS 1.2) - only forward secrecy enabled
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},

		// Security preferences
		PreferServerCipherSuites: true,
		InsecureSkipVerify:       false,

		// Modern curves
		CurvePreferences: []tls.CurveID{
			tls.CurveP256,
			tls.X25519,
		},
	}
}

// addSecurityHeaders adds security headers to HTTP responses
func addSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'")

		// Cache control for metrics endpoint
		if r.URL.Path == "/metrics" {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
		}

		next.ServeHTTP(w, r)
	})
}

// AuthMiddleware provides basic authentication
type AuthMiddleware struct {
	username string
	password string
	logger   *logrus.Entry
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(username, password string) *AuthMiddleware {
	return &AuthMiddleware{
		username: username,
		password: password,
		logger: logrus.WithFields(logrus.Fields{
			"component": "auth_middleware",
		}),
	}
}

// Middleware returns the authentication middleware function
func (am *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if am.username == "" && am.password == "" {
			// No auth configured, pass through
			next.ServeHTTP(w, r)
			return
		}

		user, pass, ok := r.BasicAuth()
		if !ok {
			am.logger.Warn("Missing basic auth credentials")
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if user != am.username || pass != am.password {
			am.logger.WithFields(logrus.Fields{
				"user":      user,
				"remote_ip": getClientIP(r),
			}).Warn("Invalid auth credentials")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		am.logger.WithFields(logrus.Fields{
			"user":      user,
			"remote_ip": getClientIP(r),
		}).Debug("Successful authentication")

		next.ServeHTTP(w, r)
	})
}

// getClientIP extracts the real client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Enabled    bool          `env:"RATE_LIMIT_ENABLED" envDefault:"false"`
	Requests   int           `env:"RATE_LIMIT_REQUESTS" envDefault:"100"`
	WindowSize time.Duration `env:"RATE_LIMIT_WINDOW" envDefault:"1m"`
	BlockTime  time.Duration `env:"RATE_LIMIT_BLOCK_TIME" envDefault:"5m"`
}

// RateLimiter provides basic rate limiting
type RateLimiter struct {
	config  *RateLimitConfig
	clients map[string]*clientState
	logger  *logrus.Entry
}

type clientState struct {
	requests int
	window   time.Time
	blocked  time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config *RateLimitConfig) *RateLimiter {
	return &RateLimiter{
		config:  config,
		clients: make(map[string]*clientState),
		logger: logrus.WithFields(logrus.Fields{
			"component": "rate_limiter",
		}),
	}
}

// Middleware returns the rate limiting middleware function
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.config.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		clientIP := getClientIP(r)

		if rl.isBlocked(clientIP) {
			rl.logger.WithField("client_ip", clientIP).Warn("Rate limit exceeded")
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		if rl.isRateLimited(clientIP) {
			rl.blockClient(clientIP)
			rl.logger.WithField("client_ip", clientIP).Warn("Rate limit exceeded, blocking client")
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) isBlocked(clientIP string) bool {
	state, exists := rl.clients[clientIP]
	if !exists {
		return false
	}

	return time.Since(state.blocked) < rl.config.BlockTime
}

func (rl *RateLimiter) isRateLimited(clientIP string) bool {
	now := time.Now()
	state, exists := rl.clients[clientIP]

	if !exists || now.Sub(state.window) > rl.config.WindowSize {
		// New client or new window
		rl.clients[clientIP] = &clientState{
			requests: 1,
			window:   now,
		}
		return false
	}

	state.requests++
	return state.requests > rl.config.Requests
}

func (rl *RateLimiter) blockClient(clientIP string) {
	if state, exists := rl.clients[clientIP]; exists {
		state.blocked = time.Now()
	}
}

// SecurityConfig aggregates all security configurations
type SecurityConfig struct {
	TLS       *TLSConfig
	Auth      *AuthConfig
	RateLimit *RateLimitConfig
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Enabled  bool   `env:"AUTH_ENABLED" envDefault:"false"`
	Username string `env:"AUTH_USERNAME"`
	Password string `env:"AUTH_PASSWORD"` // #nosec G117 -- config field, not a hardcoded secret
}

// CreateSecureHandler wraps a handler with all security middleware
func CreateSecureHandler(handler http.Handler, config *SecurityConfig) http.Handler {
	// Rate limiting (outermost)
	if config.RateLimit.Enabled {
		rateLimiter := NewRateLimiter(config.RateLimit)
		handler = rateLimiter.Middleware(handler)
	}

	// Authentication
	if config.Auth.Enabled {
		authMiddleware := NewAuthMiddleware(config.Auth.Username, config.Auth.Password)
		handler = authMiddleware.Middleware(handler)
	}

	// Security headers (innermost)
	handler = addSecurityHeaders(handler)

	return handler
}
