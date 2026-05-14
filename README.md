# Instana Event Bridge

Instana Event Bridge is a lightweight event normalization and forwarding service for Instana webhook alerts.

The bridge receives Instana webhook payloads, normalizes the event structure, converts them into syslog-compatible event messages, and forwards them to configurable outputs such as:

- Syslog (UDP/TCP)
- Local file output

The project is designed for integrations with systems such as:

- SolarWinds
- SIEM platforms
- Syslog collectors
- Log processing pipelines

---

# Features

- Lightweight Go-based webhook listener
- Syslog-compatible event formatting
- Local event file output support
- UDP/TCP syslog forwarding
- Configurable hostname support
- Field normalization and enrichment
- Safe handling for missing/null fields
- Array and custom payload support
- Docker-ready
- systemd service support
- Sample webhook payloads included

---

# Architecture

```text
Instana Webhook
        ↓
Instana Event Bridge
        ↓
+-------------------+
| Syslog UDP/TCP    |
| Local Event File  |
+-------------------+
```

---

# Supported Event Types

The bridge currently supports Instana webhook events such as:

- Open issues/incidents
- Closed issues/incidents
- Change events
- Presence events
- Monitoring issues

Official Instana reference:

https://www.ibm.com/docs/en/instana-observability?topic=alerting-webhooks

---

# Project Structure

```text
instana-event-bridge/
├── main.go
├── config.json
├── Dockerfile
├── README.md
├── samples/
│   ├── open-issue.json
│   ├── closed-issue.json
│   ├── change-event.json
│   ├── presence-event.json
│   ├── monitoring-issue.json
│   └── enriched-issue.json
├── systemd/
│   └── instana-event-bridge.service
└── events/
```

---

# Configuration

Example `config.json`:

```json
{
  "server": {
    "port": "8080",
    "hostname": ""
  },
  "limits": {
    "maxFieldLength": 2000
  },
  "outputs": {
    "syslog": {
      "enabled": true,
      "protocol": "udp",
      "host": "127.0.0.1",
      "port": 514
    },
    "file": {
      "enabled": true,
      "path": "./events/instana-events.syslog"
    }
  }
}
```

---

# Local Run

```bash
go run main.go
```

Health endpoint:

```text
GET /health
```

Webhook endpoint:

```text
POST /webhook/instana
```

---

# Docker

## Build

```bash
docker build -t instana-event-bridge .
```

## Run

```bash
touch instana-events.syslog

docker run -d \
-p 8080:8080 \
-v ./events/instana-events.syslog:/app/events/instana-events.syslog \
--name instana-event-bridge \
instana-event-bridge
```

---

# Example Test Request

```bash
curl -X POST http://localhost:8080/webhook/instana \
-H "Content-Type: application/json" \
-d @samples/open-issue.json
```

---

# Example Generated Event

```text
<134>May 14 14:47:27 instana-event-bridge source="instana" event_type="issue" state="OPEN" severity="CRITICAL" raw_severity="5" event_id="53650436-8e35-49a3-a610-56b442ae7620"
```

---

# systemd Service

Example service file:

```text
systemd/instana-event-bridge.service
```

Install:

```bash
sudo cp systemd/instana-event-bridge.service /etc/systemd/system/

sudo systemctl daemon-reload

sudo systemctl enable instana-event-bridge

sudo systemctl start instana-event-bridge
```

---

# Sample Payloads

Sample Instana webhook payloads are available under:

```text
samples/
```

These samples can be used for:

- Local testing
- Validation
- Integration testing
- Load testing
- Development

---

# Notes

- Missing fields are normalized as `not available`
- Arrays are converted into comma-separated values
- Custom payloads are safely serialized
- Syslog-compatible formatting is preserved
- Docker timezone support is enabled via `tzdata`

---

# License

MIT