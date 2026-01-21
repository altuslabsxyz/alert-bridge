# Suggested Commands for Alert-Bridge Development

## Building
```bash
make build                    # Build binary to bin/alert-bridge
go build -o bin/alert-bridge ./cmd/alert-bridge
```

## Testing
```bash
make test                     # Run all tests (unit + e2e mock)
make test-unit                # Run only unit tests
make test-e2e                 # Run fast mock-based e2e tests (~1s)
make test-e2e-docker          # Run Docker-based integration tests (~5min)
make test-coverage            # Run tests with coverage report
```

## Linting & Formatting
```bash
make lint                     # Run golangci-lint
make fmt                      # Format code with gofmt and goimports
go vet ./...                  # Run go vet checks
```

## Dependency Management
```bash
make tidy                     # go mod tidy
make deps                     # go mod download
```

## Running
```bash
make run                      # Run the application
make dev                      # Development mode with hot reload (requires air)
make docker-run               # Run in Docker container
```

## Docker
```bash
make docker-build             # Build Docker image
docker-compose -f docker-compose.test.yaml up  # Run test environment
```

## Utility Commands (Darwin)
```bash
ls -la                        # List files
find . -name "*.go" -type f   # Find Go files
grep -r "pattern" .           # Search for pattern
```
