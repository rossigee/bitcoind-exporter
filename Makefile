# Makefile for bitcoind-exporter

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=bitcoind-exporter
BINARY_UNIX=$(BINARY_NAME)_unix

# Build info
VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT ?= $(shell git rev-parse --short HEAD)
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Linker flags
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE) -w -s"

# Docker
DOCKER_IMAGE=bitcoind-exporter
DOCKER_TAG=latest

.PHONY: all build clean test coverage lint security deps help

## Build
all: clean lint test build ## Run all build steps

build: ## Build the binary
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) -v .

build-linux: ## Build for Linux
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_UNIX) -v .

build-all: ## Build for all platforms
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 -v .
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-linux-arm64 -v .
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 -v .
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 -v .
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe -v .

## Testing
test: ## Run unit tests
	RPC_ADDRESS=http://localhost:8332 RPC_USER=test RPC_PASS=test $(GOTEST) -v -race -timeout 30s ./fetcher ./util ./zmq ./prometheus ./security

test-config: ## Run config tests (requires env vars)
	RPC_ADDRESS=http://localhost:8332 RPC_USER=test RPC_PASS=test $(GOTEST) -v -race -timeout 30s ./config -run "TestLoadConfiguration|TestEnvironmentVariableParsing"

test-all: ## Run all tests including config
	RPC_ADDRESS=http://localhost:8332 RPC_USER=test RPC_PASS=test $(GOTEST) -v -race -timeout 30s ./...

test-coverage: ## Run tests with coverage
	RPC_ADDRESS=http://localhost:8332 RPC_USER=test RPC_PASS=test $(GOTEST) -v -race -coverprofile=coverage.out -covermode=atomic ./fetcher ./util ./zmq ./prometheus ./security
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-integration: ## Run integration tests
	$(GOTEST) -v -race -tags=integration ./...

benchmark: ## Run benchmarks
	$(GOTEST) -bench=. -benchmem -run=^$$ ./...

## Quality
lint: ## Run linter
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run --timeout 5m

lint-fix: ## Run linter with auto-fix
	golangci-lint run --fix --timeout 5m

security: ## Run security checks
	@which gosec > /dev/null || (echo "Installing gosec..." && go install github.com/securego/gosec/v2/cmd/gosec@latest)
	gosec -fmt json -out gosec-report.json -stdout ./...

vulnerability-check: ## Check for vulnerabilities
	@which govulncheck > /dev/null || (echo "Installing govulncheck..." && go install golang.org/x/vuln/cmd/govulncheck@latest)
	govulncheck ./...

## Dependencies
deps: ## Download dependencies
	$(GOMOD) download
	$(GOMOD) verify

deps-update: ## Update dependencies
	$(GOMOD) get -u ./...
	$(GOMOD) tidy

deps-vendor: ## Vendor dependencies
	$(GOMOD) vendor

## Docker
docker-build: ## Build Docker image
	docker build -f Dockerfile.secure -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

docker-build-dev: ## Build development Docker image
	docker build -t $(DOCKER_IMAGE):dev .

docker-run: ## Run Docker container
	docker run --rm -p 3000:3000 \
		-e RPC_ADDRESS=http://host.docker.internal:8332 \
		-e RPC_USER=bitcoin \
		-e RPC_PASS=password \
		$(DOCKER_IMAGE):$(DOCKER_TAG)

docker-scan: ## Scan Docker image for vulnerabilities
	@which trivy > /dev/null || (echo "Installing trivy..." && go install github.com/aquasecurity/trivy/cmd/trivy@latest)
	trivy image $(DOCKER_IMAGE):$(DOCKER_TAG)

## Development
dev: ## Run in development mode
	$(GOCMD) run . \
		-e RPC_ADDRESS=http://127.0.0.1:18443 \
		-e RPC_USER=test \
		-e RPC_PASS=test \
		-e LOG_LEVEL=debug

dev-regtest: ## Run with Bitcoin regtest
	docker-compose -f docker-compose.dev.yml up

watch: ## Watch for changes and rebuild
	@which air > /dev/null || (echo "Installing air..." && go install github.com/cosmtrek/air@latest)
	air

## Maintenance
clean: ## Clean build artifacts
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)
	rm -f $(BINARY_NAME)-*
	rm -f coverage.out coverage.html
	rm -f gosec-report.json

clean-cache: ## Clean Go cache
	$(GOCMD) clean -cache -modcache -i -r

format: ## Format code
	$(GOCMD) fmt ./...
	goimports -w .

install-tools: ## Install development tools
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install github.com/cosmtrek/air@latest
	go install golang.org/x/tools/cmd/goimports@latest

## Information
version: ## Show version info
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Date: $(DATE)"

help: ## Show this help
	@echo 'Usage: make <OPTIONS> ... <TARGETS>'
	@echo ''
	@echo 'Available targets are:'
	@grep -E '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ''

# Default target
.DEFAULT_GOAL := help