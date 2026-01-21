# Alert-Bridge Project Overview

## Purpose
Alert-Bridge is a production-grade Alertmanager webhook receiver that bridges alerts from Prometheus Alertmanager to notification systems like Slack and PagerDuty. It provides:
- Alert deduplication to prevent notification spam
- State tracking (active → acknowledged → resolved)
- Multi-channel notification support
- Silence management

## Tech Stack
- **Language**: Go 1.24.2
- **Architecture**: Clean Architecture (Domain-Driven Design)
- **Databases**: In-memory, SQLite, MySQL support
- **Message Queue**: Slack Socket Mode for bi-directional communication
- **Metrics**: Prometheus client + OpenTelemetry
- **External APIs**: Slack API, PagerDuty Events API v2

## Directory Structure
```
alert-bridge/
├── cmd/alert-bridge/         # Main entry point
├── internal/
│   ├── domain/               # Core business entities and interfaces
│   │   ├── entity/           # Alert, AckEvent, Silence entities
│   │   └── repository/       # Repository interfaces
│   ├── usecase/              # Business logic use cases
│   │   ├── alert/            # ProcessAlertUseCase, Notifier interface
│   │   ├── slack/            # Slack interaction handling
│   │   └── pagerduty/        # PagerDuty webhook handling
│   ├── adapter/              # HTTP handlers and DTOs
│   │   ├── handler/          # HTTP request handlers
│   │   └── dto/              # Data transfer objects
│   └── infrastructure/       # External implementations
│       ├── slack/            # Slack client
│       ├── pagerduty/        # PagerDuty client
│       └── persistence/      # Repository implementations
├── test/
│   ├── e2e/                  # Fast mock-based e2e tests (~1s)
│   └── e2e-docker/           # Docker-based integration tests (~5min)
└── config/                   # Configuration templates
```

## Key Interfaces
- `alert.Notifier` - Contract for notification channels (Slack, PagerDuty)
- `repository.AlertRepository` - Alert persistence
- `repository.SilenceRepository` - Silence management
