# Alert Bridge

Bridge Alertmanager alerts to Slack and PagerDuty with bidirectional acknowledgment sync.

## Features

- Receive alerts from Alertmanager webhooks
- Send alerts to Slack and PagerDuty
- Bidirectional ack sync (Slack ↔ PagerDuty)
- Slack slash commands: `/alert-status`, `/summary`
- Persistent storage (SQLite/MySQL)
- Alert silence management
- Webhook security (HMAC-SHA256)

## Quick Start

**Prerequisites:** Go 1.24+

```bash
# Clone and build
git clone https://github.com/altuslabsxyz/alert-bridge.git
cd alert-bridge
go build -o alert-bridge ./cmd/alert-bridge

# Configure
cp config/config.example.yaml config/config.yaml
# Edit config/config.yaml with your credentials

# Run
./alert-bridge
```

### Minimal Configuration

```yaml
server:
  port: 8080

storage:
  type: sqlite
  sqlite:
    path: ./data/alert-bridge.db

slack:
  enabled: true
  bot_token: ${SLACK_BOT_TOKEN}
  signing_secret: ${SLACK_SIGNING_SECRET}
  channel_id: ${SLACK_CHANNEL_ID}

pagerduty:
  enabled: true
  api_token: ${PAGERDUTY_API_TOKEN}
  routing_key: ${PAGERDUTY_ROUTING_KEY}
  webhook_secret: ${PAGERDUTY_WEBHOOK_SECRET}
```

### Webhook Endpoints

- Alertmanager: `POST /webhook/alertmanager`
- PagerDuty: `POST /webhook/pagerduty`
- Health check: `GET /health`

## Storage Options

| Backend | Persistence | Multi-instance | Use Case |
|---------|-------------|----------------|----------|
| Memory | No | No | Development |
| SQLite | Yes | No | Production (single instance) |
| MySQL | Yes | Yes | Production (HA) |

See [docs/storage.md](docs/storage.md) for details.

## Documentation

- [Installation Guide](docs/installation.md)
- [Deployment Guide](docs/deployment.md) - Docker & Kubernetes
- [API Reference](docs/api.md)
- [Development Guide](docs/development.md)
- [Troubleshooting](docs/troubleshooting.md)

## Contributing

Contributions welcome! See [docs/development.md](docs/development.md) for guidelines.

## License

MIT License - Copyright (c) 2025
