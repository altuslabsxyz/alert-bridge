# Code Style and Conventions

## Go Conventions
- Follow standard Go naming conventions (camelCase for private, PascalCase for exported)
- Error variables prefixed with `Err` (e.g., `ErrAlertNotFound`)
- Context as first parameter in methods
- Return errors as last return value

## Architecture Patterns
- **Clean Architecture**: Domain → Use Case → Adapter → Infrastructure
- **Dependency Inversion**: All dependencies point inward toward domain
- **Interface Segregation**: Small, focused interfaces (e.g., `Notifier` with 3 methods)

## Linting
- Uses golangci-lint v2 with comprehensive ruleset
- Key enabled linters: errcheck, govet, staticcheck, gosec, gocritic, revive
- Max cyclomatic complexity: 20
- Local import prefix: `github.com/qj0r9j0vc2/alert-bridge`

## Entity Naming
- Severity: `SeverityCritical`, `SeverityWarning`, `SeverityInfo`
- State: `StateActive`, `StateAcked`, `StateResolved`

## Testing Conventions
- Test files: `*_test.go`
- Test function names: `Test<FunctionName><Scenario>`
- Use testify for assertions where appropriate
- E2E tests use harness pattern for setup/teardown
