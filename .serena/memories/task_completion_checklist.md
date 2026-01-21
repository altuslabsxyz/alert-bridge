# Task Completion Checklist

When completing a task, ensure the following:

## Before Committing
1. **Tests pass**: `make test` runs without failures
2. **Linting passes**: `make lint` has no errors
3. **Formatting applied**: `make fmt` was run

## Code Quality Checks
- [ ] No hardcoded secrets or sensitive data
- [ ] Error handling is complete (no ignored errors)
- [ ] Context propagation is correct
- [ ] Thread-safety considered for concurrent code
- [ ] Interface contracts are satisfied

## E2E Testing
- For fast feedback: `make test-e2e` (mock-based, ~1s)
- For full integration: `make test-e2e-docker` (requires Docker)

## Documentation
- Update README if public API changes
- Add/update godoc comments for exported symbols
