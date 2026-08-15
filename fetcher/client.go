package fetcher

import (
	"fmt"
	"os"
	"strings"

	"github.com/rossigee/bitcoind-exporter/config"
	"github.com/rossigee/bitcoind-exporter/util"
	"github.com/sirupsen/logrus"
	"github.com/ybbus/jsonrpc/v3"
)

var log = logrus.WithFields(logrus.Fields{
	"prefix": "fetcher",
})

type Client struct {
	RpcClient jsonrpc.RPCClient
}

func NewClient() *Client {
	auth, err := computeBasicAuth()
	if err != nil {
		// Never fatal here: NewClient may be called from HTTP handlers (e.g.
		// /ready), so a transient cookie-file problem must not crash the process.
		// Boot-time misconfiguration is caught by config.loadConfiguration().
		log.WithError(err).Error("Failed to compute RPC authentication; RPC requests will be unauthenticated")
	}

	headers := make(map[string]string)
	if auth != "" {
		headers["Authorization"] = "Basic " + auth
	}

	return &Client{
		RpcClient: jsonrpc.NewClientWithOpts(computeAddress(), &jsonrpc.RPCClientOpts{
			CustomHeaders: headers,
		}),
	}
}

func computeBasicAuth() (string, error) {
	user := config.C.RPCUser
	pass := config.C.RPCPass
	cookieFile := config.C.RPCCookieFile

	if cookieFile != "" {
		cookie, err := os.ReadFile(cookieFile) // #nosec G304 -- path is operator-configured
		if err != nil {
			return "", fmt.Errorf("failed to read cookie file: %w", err)
		}
		cookieStr := strings.TrimSpace(string(cookie))

		if !strings.Contains(cookieStr, ":") {
			return "", fmt.Errorf("invalid cookie file format: missing ':' separator")
		}

		return util.StringToBase64(cookieStr), nil
	}

	return util.StringToBase64(fmt.Sprintf("%s:%s", user, pass)), nil
}

func computeAddress() string {
	address := config.C.RPCAddress

	if strings.HasPrefix(address, "http://") || strings.HasPrefix(address, "https://") {
		return address
	} else {
		return fmt.Sprintf("http://%s", address)
	}
}
