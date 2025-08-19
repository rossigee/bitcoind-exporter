# Bitcoind Prometheus Exporter ₿

**Prometheus metrics for a bitcoin node made simple**

![Build](https://img.shields.io/github/actions/workflow/status/rossigee/bitcoind-exporter/ci.yml?branch=master)
![Go Version](https://img.shields.io/badge/go-1.25-blue.svg)
![Test Coverage](https://img.shields.io/badge/coverage-64.3%25-green.svg)
![Security](https://img.shields.io/badge/security-scanned-green.svg)
![License](https://img.shields.io/github/license/rossigee/bitcoind-exporter)

## 🔍 About the project

A Prometheus Exporter, which provides a deep insight into a Bitcoin full node.

## ⚙️ Configuration

This tool is configured via environment variables. Some environment variables are required and some activate additional functionalities.

| Variable                | Description                                                                                                                                       | Required | Default |
| ----------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- | -------- | ------- |
| `RPC_ADDRESS`           | The RPC address for the Bitcoin full node, e.g. `http://127.0.0.1:8332`                                                                           | ✅       |         |
| `RPC_USER`              | The user name that was defined in the Bitcoin Node configuration                                                                                  | ✅       |         |
| `RPC_PASS`              | The password that was set in the Bitcoin Node configuration                                                                                       | ✅       |         |
| `RPC_COOKIE_FILE`       | The path to the cookie file                                                                                                                       | ✅       |         |
| `ZMQ_ADDRESS`           | The address to the ZeroMQ interface of the Bitcoin Fullnode. This variable is required to determine the transcation rates. e.g. `127.0.0.1:28333` | ❌       |         |
| `FETCH_INTERVAL`        | The interval at which the metrics are to be recalculated.                                                                                         | ❌       | `10`    |
| `METRIC_PORT`           | The port via which the metrics are provided.                                                                                                      | ❌       | `3000`  |
| `LOG_LEVEL`             | The log level for the service                                                                                                                     | ❌       | `info`  |
| `TLS_ENABLED`           | Enable HTTPS with TLS encryption                                                                                                                  | ❌       | `false` |
| `TLS_CERT_FILE`         | Path to TLS certificate file                                                                                                                      | ❌       |         |
| `TLS_KEY_FILE`          | Path to TLS private key file                                                                                                                      | ❌       |         |
| `TLS_MIN_VERSION`       | Minimum TLS version (1.0, 1.1, 1.2, 1.3)                                                                                                          | ❌       | `1.2`   |
| `AUTH_ENABLED`          | Enable HTTP Basic Authentication                                                                                                                  | ❌       | `false` |
| `AUTH_USERNAME`         | Username for HTTP Basic Authentication                                                                                                            | ❌       |         |
| `AUTH_PASSWORD`         | Password for HTTP Basic Authentication                                                                                                            | ❌       |         |
| `RATE_LIMIT_ENABLED`    | Enable rate limiting                                                                                                                              | ❌       | `false` |
| `RATE_LIMIT_REQUESTS`   | Maximum requests per window                                                                                                                       | ❌       | `100`   |
| `RATE_LIMIT_WINDOW`     | Rate limit window duration                                                                                                                        | ❌       | `1m`    |
| `RATE_LIMIT_BLOCK_TIME` | Block duration after rate limit exceeded                                                                                                          | ❌       | `5m`    |

Please note that either `RPC_USER` and `RPC_PASS` or `RPC_COOKIE_FILE` must be set.

## 🔗 Endpoints

The exporter provides several endpoints:

| Endpoint   | Description                                                   |
| ---------- | ------------------------------------------------------------- |
| `/metrics` | Prometheus metrics endpoint                                   |
| `/health`  | Health check endpoint (always returns HTTP 200)               |
| `/ready`   | Readiness check endpoint (validates Bitcoin RPC connectivity) |

## 🔒 Security Features

The exporter includes comprehensive security features:

- **TLS/HTTPS Support**: Full TLS encryption with configurable minimum versions
- **HTTP Basic Authentication**: Optional username/password protection
- **Rate Limiting**: Configurable request rate limiting with IP-based tracking
- **Security Headers**: Automatic security headers (X-Frame-Options, CSP, etc.)
- **Kubernetes Integration**: Health and readiness checks for container orchestration

### Security Configuration Example

```bash
# Enable TLS
export TLS_ENABLED=true
export TLS_CERT_FILE=/path/to/cert.pem
export TLS_KEY_FILE=/path/to/key.pem
export TLS_MIN_VERSION=1.2

# Enable authentication
export AUTH_ENABLED=true
export AUTH_USERNAME=admin
export AUTH_PASSWORD=secure_password

# Enable rate limiting
export RATE_LIMIT_ENABLED=true
export RATE_LIMIT_REQUESTS=50
export RATE_LIMIT_WINDOW=1m
export RATE_LIMIT_BLOCK_TIME=5m
```

## 💻 Grafana Dashboard

The official Grafana dashboard can be found here: https://grafana.com/grafana/dashboards/21351

## 🐳 Run with Docker

### Docker-CLI

```bash
docker run -d --name bitcoind_exporter \
  -e RPC_ADDRESS=http://127.0.0.1:8332 \
  -e RPC_USER=mempool \
  -e RPC_PASS=mempool \
  -e ZMQ_ADDRESS=127.0.0.1:28333 \
  -e AUTH_ENABLED=true \
  -e AUTH_USERNAME=admin \
  -e AUTH_PASSWORD=secure_password \
  -e RATE_LIMIT_ENABLED=true \
  -v /path/to/cookie/.cookie:/.cookie:ro \
  -v /path/to/certs:/certs:ro \
   ghcr.io/primexz/bitcoind-exporter:latest
```

### 🚀 Docker-Compose

```bash
vim docker-compose.yml
```

```yaml
version: "3.8"
services:
  bitcoind_exporter:
    image: ghcr.io/primexz/bitcoind-exporter:latest
    environment:
      - RPC_ADDRESS=http://127.0.0.1:8332
      - RPC_USER=mempool
      - RPC_PASS=mempool
      - ZMQ_ADDRESS=127.0.0.1:28333
      # Security configuration
      - TLS_ENABLED=true
      - TLS_CERT_FILE=/certs/cert.pem
      - TLS_KEY_FILE=/certs/key.pem
      - AUTH_ENABLED=true
      - AUTH_USERNAME=admin
      - AUTH_PASSWORD=secure_password
      - RATE_LIMIT_ENABLED=true
      - RATE_LIMIT_REQUESTS=100
    volumes:
      # Optional, only necessary if Cookie-Auth is to be used
      - /path/to/cookie/.cookie:/.cookie:ro
      # TLS certificates (if TLS is enabled)
      - /path/to/certs:/certs:ro
    ports:
      - "3000:3000"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:3000/health"]
      interval: 30s
      timeout: 10s
      retries: 3
    restart: always
```

```bash
docker-compose up -d
```

## 🧪 Development & Testing

The project maintains high code quality standards:

- **Test Coverage**: 64.3% overall coverage with comprehensive test suites
- **Security Scanning**: Automated vulnerability scanning with govulncheck
- **Code Quality**: Zero linting issues with golangci-lint
- **Race Condition Testing**: Thread-safe implementation with race detection
- **CI/CD Pipeline**: Automated testing, security scanning, and multi-platform builds

### Running Tests

```bash
# Run all tests with coverage
make test

# Run linting
make lint

# Run security scan
govulncheck ./...

# Run benchmarks
go test -bench=. ./...
```

### Pre-commit Hooks

The project uses pre-commit hooks to ensure code quality:

```bash
# Install pre-commit hooks
./scripts/install-pre-commit.sh

# Run pre-commit manually
pre-commit run --all-files

# Skip hooks (if needed)
git commit --no-verify
```

Pre-commit hooks include:

- Go formatting, imports, and vet
- Golangci-lint
- Security scanning (govulncheck)
- Test coverage validation (≥60%)
- Secret detection
- YAML/JSON validation
- Dockerfile linting

### Test Coverage by Package

| Package            | Coverage |
| ------------------ | -------- |
| fetcher            | 64.3%    |
| security           | 94.3%    |
| prometheus/metrics | 100%     |
| config             | 85%+     |
| zmq                | 19.0%    |
