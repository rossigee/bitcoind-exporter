package config

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// RSA key size for test certificates
	testRSAKeySize = 2048
	// IPv4 localhost bytes
	localhostIPv4Byte3 = 127
	// Environment variable split parts
	envSplitParts = 2
)

// createTestCertificates creates valid test certificate and key files for testing
func createTestCertificates(t *testing.T) (certFile, keyFile string) {
	t.Helper()
	
	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, testRSAKeySize)
	require.NoError(t, err)

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"Test Org"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"Test City"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses: []net.IP{net.IPv4(localhostIPv4Byte3, 0, 0, 1), net.IPv6loopback},
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)

	// Write certificate to file
	tempDir := t.TempDir()
	certFile = filepath.Join(tempDir, "cert.pem")
	certOut, err := os.Create(filepath.Clean(certFile))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, certOut.Close())
	}()

	err = pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	require.NoError(t, err)

	// Write private key to file
	keyFile = filepath.Join(tempDir, "key.pem")
	keyOut, err := os.Create(filepath.Clean(keyFile))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, keyOut.Close())
	}()

	privKeyPEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}
	err = pem.Encode(keyOut, privKeyPEM)
	require.NoError(t, err)

	return certFile, keyFile
}

// restoreEnvironment restores the original environment variables
func restoreEnvironment(oldEnv []string) {
	os.Clearenv()
	for _, env := range oldEnv {
		if len(env) > 0 {
			parts := strings.SplitN(env, "=", envSplitParts)
			if len(parts) == envSplitParts {
				_ = os.Setenv(parts[0], parts[1])
			}
		}
	}
}

// cleanupTestFile safely removes a test file, ignoring errors in test cleanup
func cleanupTestFile(filePath string) {
	_ = os.Remove(filePath)
}

// runFileValidationTest runs a generic file validation test
func runFileValidationTest(
	t *testing.T, 
	name, testFile, expectedErr string, 
	setupFile func(t *testing.T) (string, func()), 
	validator func(string) error,
) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		var filePath string
		var cleanup func()

		if setupFile != nil {
			filePath, cleanup = setupFile(t)
		} else {
			filePath = testFile
		}

		defer func() {
			if cleanup != nil {
				cleanup()
			}
		}()

		err := validator(filePath)

		if expectedErr != "" {
			require.Error(t, err)
			assert.Contains(t, err.Error(), expectedErr)
		} else {
			assert.NoError(t, err)
		}
	})
}