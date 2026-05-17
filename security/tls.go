package security

import (
	"crypto/subtle"
	"crypto/tls"
	"net"
	"net/http"
	"sync"
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

	// Only TLS 1.2+ is accepted; TLS 1.0/1.1 are insecure and disabled.
	minVersion := tls.VersionTLS12
	if config.MinVersion == "1.3" {
		minVersion = tls.VersionTLS13
	}

	return &tls.Config{ // #nosec G402 -- MinVersion is set to TLS 1.2 or 1.3 above
		MinVersion: uint16(minVersion), // #nosec G115 -- tls version constants fit safely in uint16

		// Forward-secrecy cipher suites for TLS 1.2; TLS 1.3 suites are
		// controlled by Go's runtime and cannot be restricted here.
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},

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

		// Use constant-time comparison to prevent timing attacks.
		userMatch := subtle.ConstantTimeCompare([]byte(user), []byte(am.username))
		passMatch := subtle.ConstantTimeCompare([]byte(pass), []byte(am.password))
		if (userMatch & passMatch) != 1 {
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

// getClientIP extracts the client IP from the request's RemoteAddr.
// X-Forwarded-For is intentionally ignored; it is client-controlled and
// cannot be trusted for rate-limiting or access-control decisions.
func getClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Enabled    bool          `env:"RATE_LIMIT_ENABLED" envDefault:"false"`
	Requests   int           `env:"RATE_LIMIT_REQUESTS" envDefault:"100"`
	WindowSize time.Duration `env:"RATE_LIMIT_WINDOW" envDefault:"1m"`
	BlockTime  time.Duration `env:"RATE_LIMIT_BLOCK_TIME" envDefault:"5m"`
}

// RateLimiter provides basic rate limiting.
// The clients map is protected by a RWMutex to prevent data races.
type RateLimiter struct {
	config      *RateLimitConfig
	clients     map[string]*clientState
	mu          sync.RWMutex
	lastCleanup time.Time
	logger      *logrus.Entry
}

type clientState struct {
	requests int
	window   time.Time
	blocked  time.Time
}

const rateLimiterCleanupInterval = 5 * time.Minute

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config *RateLimitConfig) *RateLimiter {
	return &RateLimiter{
		config:      config,
		clients:     make(map[string]*clientState),
		lastCleanup: time.Now(),
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
	rl.mu.RLock()
	state, exists := rl.clients[clientIP]
	rl.mu.RUnlock()
	if !exists {
		return false
	}
	return time.Since(state.blocked) < rl.config.BlockTime
}

func (rl *RateLimiter) isRateLimited(clientIP string) bool {
	now := time.Now()
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Periodically evict stale entries to prevent unbounded map growth.
	if now.Sub(rl.lastCleanup) > rateLimiterCleanupInterval {
		for ip, state := range rl.clients {
			if now.Sub(state.window) > rl.config.WindowSize && now.Sub(state.blocked) > rl.config.BlockTime {
				delete(rl.clients, ip)
			}
		}
		rl.lastCleanup = now
	}

	state, exists := rl.clients[clientIP]
	if !exists || now.Sub(state.window) > rl.config.WindowSize {
		rl.clients[clientIP] = &clientState{requests: 1, window: now}
		return false
	}

	state.requests++
	return state.requests > rl.config.Requests
}

func (rl *RateLimiter) blockClient(clientIP string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
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
