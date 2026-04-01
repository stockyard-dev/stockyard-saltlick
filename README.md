# Stockyard Saltlick

**Feature flag service.** Create, manage, and evaluate feature flags with a single binary. No Redis, no Postgres, no external dependencies.

Part of the [Stockyard](https://stockyard.dev) suite of self-hosted developer tools.

## Quick Start

```bash
# Download and run
curl -sfL https://stockyard.dev/install/saltlick | sh
saltlick

# Or with Docker
docker run -p 8800:8800 -v saltlick-data:/data ghcr.io/stockyard-dev/stockyard-saltlick:latest
```

Dashboard at [http://localhost:8800/ui](http://localhost:8800/ui)

## Usage

```bash
# Create a flag
curl -X POST http://localhost:8800/api/flags \
  -H "Content-Type: application/json" \
  -d '{"name":"new-checkout","description":"New checkout flow","enabled":true}'

# Evaluate a flag (the hot path — just an HTTP GET)
curl http://localhost:8800/api/eval/new-checkout?user_id=user_123
# → {"flag":"new-checkout","enabled":true,"reason":"enabled"}

# Batch evaluate multiple flags
curl -X POST http://localhost:8800/api/eval/batch \
  -H "Content-Type: application/json" \
  -d '{"flags":["new-checkout","dark-mode"],"user_id":"user_123"}'

# Update flag (enable percentage rollout)
curl -X PUT http://localhost:8800/api/flags/new-checkout \
  -H "Content-Type: application/json" \
  -d '{"rollout_percent":25}'

# Get evaluation stats
curl http://localhost:8800/api/flags/new-checkout/stats
```

## API

| Method | Path | Description |
|--------|------|-------------|
| POST | /api/flags | Create flag |
| GET | /api/flags | List all flags |
| GET | /api/flags/{name} | Get flag detail |
| PUT | /api/flags/{name} | Update flag |
| DELETE | /api/flags/{name} | Delete flag |
| GET | /api/eval/{name} | Evaluate flag for user |
| POST | /api/eval/batch | Evaluate multiple flags |
| GET | /api/flags/{name}/stats | Evaluation statistics |
| GET | /api/status | Service status |
| GET | /health | Health check |
| GET | /ui | Web dashboard |

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| PORT | 8800 | HTTP port |
| DATA_DIR | ./data | SQLite data directory |
| RETENTION_DAYS | 30 | Evaluation log retention |
| SALTLICK_LICENSE_KEY | | Pro license key |

## Free vs Pro

| Feature | Free | Pro ($2.99/mo) |
|---------|------|----------------|
| Flags | 10 | Unlimited |
| Global on/off | ✓ | ✓ |
| Percentage rollout | — | ✓ |
| User targeting | — | ✓ |
| Environments | — | ✓ |
| Eval log retention | 7 days | 90 days |
| Webhook on change | — | ✓ |

## License

Apache 2.0 — see [LICENSE](LICENSE).
