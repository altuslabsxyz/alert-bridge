# API Reference

## Endpoints

### Health Check

Check if the service is running and healthy.

```http
GET /health
```

**Response:**
```json
{
  "status": "ok"
}
```

### Alertmanager Webhook

Receive alerts from Alertmanager.

```http
POST /alertmanager/webhook
Content-Type: application/json
```

**Request Body:**
```json
{
  "receiver": "alert-bridge",
  "status": "firing",
  "alerts": [
    {
      "status": "firing",
      "labels": {
        "alertname": "HighCPU",
        "severity": "critical",
        "instance": "server-1"
      },
      "annotations": {
        "description": "CPU usage is above 90%",
        "summary": "High CPU on server-1"
      },
      "startsAt": "2025-01-15T10:00:00Z",
      "endsAt": "0001-01-01T00:00:00Z",
      "fingerprint": "abc123"
    }
  ]
}
```

**Response:**
```json
{
  "status": "success",
  "received": 1
}
```

### Slack Interaction

Handle button clicks and interactions from Slack messages.

```http
POST /slack/interaction
Content-Type: application/x-www-form-urlencoded
```

This endpoint handles:
- Acknowledge button clicks
- Add note actions
- Silence duration selections

**Request:** Form-encoded Slack interaction payload

**Response:**
```json
{
  "response_type": "in_channel",
  "text": "Alert acknowledged"
}
```

### Slack Events

Receive events from Slack Event API.

```http
POST /slack/events
Content-Type: application/json
```

**Request Body:**
```json
{
  "type": "event_callback",
  "event": {
    "type": "app_mention",
    "text": "@AlertBridge help",
    "user": "U123456",
    "channel": "C123456"
  }
}
```

**Response:**
```json
{
  "ok": true
}
```

### PagerDuty Webhook

Receive incident updates from PagerDuty.

```http
POST /pagerduty/webhook
Content-Type: application/json
```

**Request Body:**
```json
{
  "messages": [
    {
      "event": "incident.acknowledged",
      "incident": {
        "id": "PINC123",
        "status": "acknowledged",
        "title": "High CPU Alert"
      }
    }
  ]
}
```

**Response:**
```json
{
  "status": "success"
}
```

## Webhook Configuration

### Alertmanager

Add to your Alertmanager configuration:

```yaml
receivers:
  - name: 'alert-bridge'
    webhook_configs:
      - url: 'http://alert-bridge:8080/alertmanager/webhook'
        send_resolved: true
```

### Slack App

Configure your Slack App:

1. **Interactivity & Shortcuts**
   - Request URL: `https://your-domain.com/slack/interaction`

2. **Event Subscriptions**
   - Request URL: `https://your-domain.com/slack/events`
   - Subscribe to: `app_mention`, `message.channels`

### PagerDuty

Configure webhook in PagerDuty:

1. Go to **Integrations** > **Generic Webhooks**
2. Add webhook URL: `https://your-domain.com/pagerduty/webhook`
3. Subscribe to events: `incident.acknowledged`, `incident.resolved`

## Next Steps

- [Installation](installation.md) - Set up the application
- [Deployment](deployment.md) - Deploy to production
