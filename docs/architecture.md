# Architecture

## Overview

Alert Bridge follows Clean Architecture principles, separating concerns into distinct layers with clear dependencies.

## System Architecture

```
┌─────────────────┐         ┌─────────────────┐
│  Alertmanager   │         │     Slack       │
└────────┬────────┘         └────────┬────────┘
         │                           │
         │ Webhooks         Actions  │
         │                           │
         └──────────┬────────────────┘
                    │
              ┌─────▼─────┐
              │           │
              │   HTTP    │
              │  Server   │
              │           │
              └─────┬─────┘
                    │
         ┌──────────▼──────────┐
         │                     │
         │   Alert Bridge      │
         │   (Application)     │
         │                     │
         └──────────┬──────────┘
                    │
         ┌──────────▼──────────┐
         │    Persistence      │
         │  (SQLite/MySQL)     │
         └─────────────────────┘
```

## Clean Architecture Layers

### 1. Domain Layer (`internal/domain/`)

Core business logic with no external dependencies.

**Entities** (`entity/`):
- `Alert` - Alert data model
- `AckEvent` - Acknowledgment event
- `SilenceMark` - Silence rule
- `AlertState` - Alert states (firing, acked, resolved)
- `Severity` - Severity levels (critical, warning, info)

**Repository Interfaces** (`repository/`):
- `AlertRepository` - Alert persistence operations
- `AckEventRepository` - Ack event persistence
- `SilenceRepository` - Silence persistence

**Characteristics:**
- Pure business logic
- No framework dependencies
- No I/O operations
- Easily testable

### 2. Use Case Layer (`internal/usecase/`)

Application business rules that orchestrate domain entities.

**Use Cases:**
- `AlertProcessing` - Process incoming alerts
- `AckSync` - Bidirectional acknowledgment sync
- `SilenceManagement` - Create and manage silences
- `SlackIntegration` - Send alerts to Slack
- `PagerDutyIntegration` - Sync with PagerDuty

**Characteristics:**
- Depends only on domain layer
- Uses repository interfaces
- Contains business workflows
- Independent of frameworks

### 3. Infrastructure Layer (`internal/infrastructure/`)

External integrations and concrete implementations.

**Components:**

- **Config** (`config/`): Configuration loading and parsing
- **Persistence** (`persistence/`):
  - `memory/` - In-memory implementation
  - `sqlite/` - SQLite implementation
  - `mysql/` - MySQL implementation
- **Slack** (`slack/`): Slack API client
- **PagerDuty** (`pagerduty/`): PagerDuty API client
- **Server** (`server/`): HTTP server setup

**Characteristics:**
- Implements repository interfaces
- Handles external I/O
- Framework-specific code
- Pluggable implementations

### 4. Adapter Layer (`internal/adapter/`)

HTTP request handling and response formatting.

**Components:**
- **Handlers** (`handler/`): HTTP request handlers
- **DTOs** (`dto/`): Data transfer objects
- **Presenters** (`presenter/`): Response formatting
- **Middleware** (`handler/middleware/`): HTTP middleware

**Characteristics:**
- Transforms HTTP requests to use case inputs
- Converts use case outputs to HTTP responses
- Handles request validation
- Protocol-specific logic

## Data Flow

### Alert Processing Flow

```
1. Alertmanager → POST /alertmanager/webhook
2. Handler validates and parses request
3. Handler calls AlertProcessing use case
4. Use case processes alert logic
5. Use case saves via AlertRepository
6. Use case calls SlackIntegration
7. SlackIntegration sends message to Slack
8. Handler returns success response
```

### Acknowledgment Sync Flow (Slack → PagerDuty)

```
1. User clicks "Acknowledge" in Slack
2. Slack → POST /slack/interaction
3. Handler parses interaction payload
4. Handler calls AckSync use case
5. Use case updates alert state
6. Use case saves AckEvent
7. Use case calls PagerDutyIntegration
8. PagerDutyIntegration acknowledges incident
9. Handler updates Slack message
```

### Acknowledgment Sync Flow (PagerDuty → Slack)

```
1. User acknowledges in PagerDuty
2. PagerDuty → POST /pagerduty/webhook
3. Handler validates webhook signature
4. Handler calls AckSync use case
5. Use case finds alert by PD incident ID
6. Use case updates alert state
7. Use case saves AckEvent
8. Use case calls SlackIntegration
9. SlackIntegration updates message
```

## Dependency Rule

Dependencies point inward:

```
Adapter → Use Case → Domain
   ↓          ↓
Infrastructure
```

- Domain layer has no dependencies
- Use case depends only on domain
- Infrastructure implements domain interfaces
- Adapters depend on use cases and infrastructure

## Persistence Architecture

### Repository Pattern

All persistence operations go through repository interfaces:

```go
type AlertRepository interface {
    Save(ctx context.Context, alert *entity.Alert) error
    FindByID(ctx context.Context, id string) (*entity.Alert, error)
    Update(ctx context.Context, alert *entity.Alert) error
    // ... other methods
}
```

### Multiple Implementations

- **Memory**: Fast, ephemeral, for development
- **SQLite**: File-based, single instance, good performance
- **MySQL**: Network-based, multi-instance, high availability

### Factory Pattern

Storage implementation selected at startup based on configuration:

```go
func NewRepositories(cfg *config.Config) (*Repositories, error) {
    switch cfg.Storage.Type {
    case "memory":
        return memory.NewRepositories(), nil
    case "sqlite":
        return sqlite.NewRepositories(db), nil
    case "mysql":
        return mysql.NewRepositories(db), nil
    }
}
```

## Concurrency Model

### SQLite

- File-based locking
- Single writer, multiple readers (WAL mode)
- Single instance only

### MySQL

- Row-level locking
- Optimistic locking (version field)
- Supports multiple instances
- Connection pooling per instance

## Error Handling

### Domain Errors

Custom error types for business logic:

```go
var (
    ErrAlertNotFound = errors.New("alert not found")
    ErrDuplicateAlert = errors.New("duplicate alert")
)
```

### Infrastructure Errors

Wrapped with context:

```go
return fmt.Errorf("failed to save alert: %w", err)
```

### HTTP Errors

Mapped to appropriate status codes:

- `400 Bad Request` - Invalid input
- `404 Not Found` - Resource not found
- `500 Internal Server Error` - Unexpected errors

## Configuration Management

### Layered Configuration

1. Default values (in code)
2. Configuration file (config.yaml)
3. Environment variables (override config file)

### Environment Variable Substitution

Config file supports ${VAR} syntax:

```yaml
slack:
  bot_token: ${SLACK_BOT_TOKEN}
```

## Testing Strategy

### Unit Tests

- Test domain entities in isolation
- Mock repository interfaces for use cases
- No external dependencies

### Integration Tests

- Test with real database
- Verify persistence operations
- Test concurrent operations (MySQL)

### Benchmark Tests

- Measure read/write performance
- Compare storage implementations
- Identify bottlenecks

## Scalability Considerations

### Horizontal Scaling (MySQL Only)

- Multiple instances share database
- Load balancer distributes requests
- Optimistic locking prevents conflicts

### Performance Optimization

- Connection pooling
- Prepared statements
- Batch operations where possible
- Database indexes

### Resource Management

- Graceful shutdown
- Connection cleanup
- Database checkpointing (SQLite)

## Security Considerations

### Authentication

- Slack: Signature verification
- PagerDuty: Webhook secret validation
- Alertmanager: No authentication (internal network)

### Data Protection

- Secrets in environment variables
- No secrets in logs
- Secure credential storage

### Input Validation

- Request payload validation
- SQL injection prevention (parameterized queries)
- XSS prevention (no user HTML)

## Monitoring and Observability

### Logging

- Structured JSON logging
- Configurable log levels
- Context propagation

### Health Checks

- `/health` endpoint
- Database connectivity check
- Ready/live probe support

## Future Enhancements

Potential improvements:

- Metrics endpoint (Prometheus)
- Distributed tracing (OpenTelemetry)
- Event sourcing for audit trail
- GraphQL API
- Additional integrations (Discord, Teams, etc.)

## Next Steps

- [Development Guide](development.md) - Understand the codebase
- [API Reference](api.md) - Learn about endpoints
- [Storage](storage.md) - Configure persistence
