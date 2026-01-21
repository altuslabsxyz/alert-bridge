.PHONY: build run test test-unit test-e2e test-e2e-docker clean docker-build docker-run lint fmt help

# Variables
BINARY_NAME=alert-bridge
DOCKER_IMAGE=alert-bridge
DOCKER_TAG=latest
GO=go

# Build the binary
build:
	$(GO) build -o bin/$(BINARY_NAME) ./cmd/alert-bridge

# Run the application
run:
	$(GO) run ./cmd/alert-bridge

# Run all tests (unit + e2e mock-based)
test:
	$(GO) test -v ./...

# Run only unit tests (excludes e2e)
test-unit:
	$(GO) test -v ./internal/... ./cmd/...

# Run mock-based e2e tests (fast, no Docker required)
test-e2e:
	$(GO) test -v ./test/e2e/... -timeout 60s

# Run Docker-based e2e tests (comprehensive, requires Docker)
test-e2e-docker:
	./scripts/e2e-setup.sh

# Run tests with coverage
test-coverage:
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Build Docker image
docker-build:
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

# Run Docker container
docker-run:
	docker run --rm -p 8080:8080 \
		-e SLACK_BOT_TOKEN \
		-e SLACK_SIGNING_SECRET \
		-e SLACK_CHANNEL_ID \
		-e PAGERDUTY_API_TOKEN \
		-e PAGERDUTY_ROUTING_KEY \
		$(DOCKER_IMAGE):$(DOCKER_TAG)

# Run linter
lint:
	golangci-lint run ./...

# Format code
fmt:
	$(GO) fmt ./...
	goimports -w .

# Tidy dependencies
tidy:
	$(GO) mod tidy

# Download dependencies
deps:
	$(GO) mod download

# Generate mocks (requires mockery)
mocks:
	mockery --all --dir=internal/domain/repository --output=internal/mocks --outpkg=mocks

# Development mode with hot reload (requires air)
dev:
	air

# Help
help:
	@echo "Available targets:"
	@echo "  build           - Build the binary"
	@echo "  run             - Run the application"
	@echo "  test            - Run all tests (unit + e2e mock-based)"
	@echo "  test-unit       - Run only unit tests"
	@echo "  test-e2e        - Run mock-based e2e tests (fast, no Docker)"
	@echo "  test-e2e-docker - Run Docker-based e2e tests (comprehensive)"
	@echo "  test-coverage   - Run tests with coverage"
	@echo "  clean           - Clean build artifacts"
	@echo "  docker-build    - Build Docker image"
	@echo "  docker-run      - Run Docker container"
	@echo "  lint            - Run linter"
	@echo "  fmt             - Format code"
	@echo "  tidy            - Tidy dependencies"
	@echo "  deps            - Download dependencies"
	@echo "  mocks           - Generate mocks"
	@echo "  dev             - Development mode with hot reload"
	@echo "  help            - Show this help"
