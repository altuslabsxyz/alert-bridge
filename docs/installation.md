# Installation

## Prerequisites

- Go 1.24 or later
- Slack workspace (optional)
- PagerDuty account (optional)

## Installation Steps

### 1. Clone the Repository

```bash
git clone https://github.com/qj0r9j0vc2/alert-bridge.git
cd alert-bridge
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Build the Application

```bash
go build -o alert-bridge ./cmd/alert-bridge
```

## Configuration

Create a `config/config.yaml` file based on the example:

```bash
cp config/config.example.yaml config/config.yaml
```

Edit `config/config.yaml` with your settings:

```yaml
server:
  port: 8080
  read_timeout: 5s
  write_timeout: 10s
  shutdown_timeout: 30s

# Storage Configuration
storage:
  type: sqlite  # Options: memory, sqlite, mysql
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
  service_id: ${PAGERDUTY_SERVICE_ID}
  webhook_secret: ${PAGERDUTY_WEBHOOK_SECRET}
  from_email: ${PAGERDUTY_FROM_EMAIL}

alerting:
  deduplication_window: 5m
  resend_interval: 30m
  silence_durations: [15m, 1h, 4h, 24h]

logging:
  level: info
  format: json
```

## Running the Application

### Basic Usage

Start the server:

```bash
./alert-bridge
```

### Custom Config Path

```bash
CONFIG_PATH=/path/to/config.yaml ./alert-bridge
```

### Verify Running

```bash
curl http://localhost:8080/health
```

## Environment Variables

You can use environment variables in the config file:

```yaml
slack:
  bot_token: ${SLACK_BOT_TOKEN}
  signing_secret: ${SLACK_SIGNING_SECRET}
```

Or override config path:

```bash
export CONFIG_PATH=/path/to/config.yaml
./alert-bridge
```

## Next Steps

- [Configure Storage](storage.md) - Choose and configure your storage backend
- [API Reference](api.md) - Learn about available endpoints
- [Deployment](deployment.md) - Deploy to production environments
