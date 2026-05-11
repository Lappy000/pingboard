<h1 align="center">
  <br>
  <img src="https://img.shields.io/badge/ping-board-00d4aa?style=for-the-badge&logo=go&logoColor=white" alt="PingBoard">
  <br>
  PingBoard
  <br>
</h1>

<p align="center">
  <strong>Real-time terminal dashboard for HTTP endpoint monitoring</strong>
</p>

<p align="center">
  <a href="https://github.com/Lappy000/pingboard/releases"><img src="https://img.shields.io/github/v/release/Lappy000/pingboard?style=flat-square&color=00d4aa" alt="Release"></a>
  <a href="https://github.com/Lappy000/pingboard/actions"><img src="https://img.shields.io/github/actions/workflow/status/Lappy000/pingboard/build.yml?style=flat-square" alt="Build"></a>
  <a href="https://goreportcard.com/report/github.com/Lappy000/pingboard"><img src="https://goreportcard.com/badge/github.com/Lappy000/pingboard?style=flat-square" alt="Go Report"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue?style=flat-square" alt="License"></a>
  <a href="https://github.com/Lappy000/pingboard/stargazers"><img src="https://img.shields.io/github/stars/Lappy000/pingboard?style=flat-square&color=yellow" alt="Stars"></a>
</p>

<p align="center">
  <a href="#features">Features</a> •
  <a href="#installation">Installation</a> •
  <a href="#usage">Usage</a> •
  <a href="#configuration">Configuration</a> •
  <a href="#alerts">Alerts</a>
</p>

---

## Preview

```
  ____  _             ____                      _
 |  _ \(_)_ __   __ | __ )  ___   __ _ _ __ __| |
 | |_) | | '_ \ / _`|  _ \ / _ \ / _` | '__/ _` |
 |  __/| | | | | (_| | |_) | (_) | (_| | | | (_| |
 |_|   |_|_| |_|\__, |____/ \___/ \__,_|_|  \__,_|
                |___/

  Real-time HTTP Endpoint Monitor

  ENDPOINTS: 5   UP: 4   DOWN: 1   SESSION: 14m 32s

  ┌──────────────────────────────────────────────────────────────────────────────────────────┐
  │ ENDPOINT              STATUS     LATENCY      AVG          P95          UPTIME  CHECKS   │
  ├──────────────────────────────────────────────────────────────────────────────────────────┤
  │ GitHub API            ● UP         89ms       102ms        145ms        100.0%  28       │
  │ Google                ● UP         23ms        31ms         45ms        100.0%  28       │
  │ Cloudflare DNS        ● UP         12ms        15ms         22ms        100.0%  42       │
  │ Internal API          ● DOWN        ---         ---          ---         85.7%  28       │
  │ httpbin               ● UP        201ms       189ms        312ms         96.4%  28       │
  └──────────────────────────────────────────────────────────────────────────────────────────┘

  Latency Sparklines (last 30 checks):
  GitHub API         ▂▃▂▁▂▃▄▃▂▁▁▂▃▂▁▂▃▂▂▁▂▃▃▂▁▁▂▃▂▁ (max: 145ms)
  Google             ▁▁▂▁▁▁▂▁▁▁▁▂▁▁▁▁▁▂▁▁▁▂▁▁▁▁▂▁▁▁ (max: 45ms)
  Cloudflare DNS     ▁▁▁▁▁▁▁▁▁▁▁▂▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁ (max: 22ms)
  httpbin            ▃▄▃▅▃▄▃▂▃▄▃█▃▄▃▂▃▄▃▅▃▄▃▂▃▄▃▂▃▄ (max: 312ms)

  Press Ctrl+C to exit  |  Powered by pingboard
```

## Features

- **Real-time Monitoring** — Concurrent HTTP checks with configurable intervals
- **Rich Terminal UI** — Color-coded status table with box-drawing characters
- **Latency Sparklines** — Visual latency history for each endpoint
- **Uptime Tracking** — Rolling uptime percentage with P95 latency stats
- **Flexible Alerts** — stdout, webhook (Slack/Discord), or JSON log file
- **One-shot Mode** — Run a single check and exit with status code (great for CI)
- **YAML Configuration** — Simple, declarative endpoint definitions
- **Zero Dependencies** — Single binary, no runtime requirements
- **Cross-Platform** — Works on Linux, macOS, and Windows

## Installation

### From Source (requires Go 1.21+)

```bash
go install github.com/Lappy000/pingboard/cmd/pingboard@latest
```

### Build from Repository

```bash
git clone https://github.com/Lappy000/pingboard.git
cd pingboard
make build
./bin/pingboard -config config.example.yaml
```

### Download Binary

Grab a prebuilt binary from the [Releases](https://github.com/Lappy000/pingboard/releases) page.

## Usage

```bash
# Run with a config file
pingboard -config config.yaml

# Generate an example config
pingboard -example > config.yaml

# One-shot mode (single check, exits with status code)
pingboard -config config.yaml -once

# Compact display mode
pingboard -config config.yaml -compact

# Show version
pingboard -version
```

## Configuration

Create a `config.yaml` file:

```yaml
defaults:
  method: GET
  timeout: 10s
  interval: 30s
  expect_status: 200

endpoints:
  - name: Production API
    url: https://api.example.com/health
    interval: 15s
    headers:
      Authorization: Bearer ${API_TOKEN}

  - name: Frontend
    url: https://www.example.com
    timeout: 5s

  - name: Database Health
    url: http://localhost:8080/db/health
    interval: 10s
    expect_status: 200

alerts:
  - type: stdout
    threshold: 3

  - type: webhook
    webhook_url: https://hooks.slack.com/services/T.../B.../xxx
    threshold: 2

dashboard:
  refresh_rate: 2s
  compact: false
  max_history: 100
```

### Configuration Reference

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `defaults.method` | string | `GET` | HTTP method for all endpoints |
| `defaults.timeout` | duration | `10s` | Request timeout |
| `defaults.interval` | duration | `30s` | Check interval |
| `defaults.expect_status` | int | `200` | Expected HTTP status code |
| `endpoints[].name` | string | required | Display name (must be unique) |
| `endpoints[].url` | string | required | Full URL to check |
| `endpoints[].method` | string | from defaults | Override HTTP method |
| `endpoints[].headers` | map | `{}` | Custom request headers |
| `alerts[].type` | string | required | `stdout`, `webhook`, or `log` |
| `alerts[].threshold` | int | `3` | Consecutive failures before alert |
| `dashboard.refresh_rate` | duration | `2s` | UI update frequency |

## Alerts

PingBoard supports three alert channels:

### stdout (Terminal)
Prints colored alerts directly in the terminal:
```
[14:23:01] ALERT: Production API is DOWN - connection refused
[14:25:31] RECOVERED: Production API is back UP (was down for 2m30s)
```

### Webhook (Slack, Discord, etc.)
Sends JSON payloads to any webhook URL:
```json
{
  "endpoint": "Production API",
  "url": "https://api.example.com/health",
  "status": "DOWN",
  "error": "connection refused",
  "timestamp": "2024-01-15T14:23:01Z",
  "recovered": false
}
```

### Log File
Appends events as JSON Lines to a file for later analysis:
```yaml
alerts:
  - type: log
    log_file: ./alerts.jsonl
    threshold: 3
```

## CI/CD Integration

Use one-shot mode in your CI pipeline:

```bash
# Exit code 0 = all endpoints healthy
# Exit code 1 = one or more endpoints down
pingboard -config ci-checks.yaml -once
```

## Architecture

```
cmd/pingboard/       → Entry point, CLI flags, main loop
internal/config/     → YAML config loading & validation
internal/monitor/    → Concurrent HTTP probing engine
internal/ui/         → ANSI terminal rendering
internal/alerter/    → Alert dispatch (stdout, webhook, log)
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing`)
5. Open a Pull Request

## License

MIT License — see [LICENSE](LICENSE) for details.
