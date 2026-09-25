# Server Monitor Dashboard

A small learning project showing how a monitoring/observability dashboard works using Go + rendered HTML.

## Features

- Live CPU usage and logical CPU count
- Memory usage
- Root disk usage
- 1/5/15 minute load averages
- Hostname, OS, platform, kernel and uptime
- Network RX/TX counters
- Go runtime stats: goroutines, heap, system memory, GC count and last GC pause
- PostgreSQL health check and ping latency
- `database/sql` pool stats: open, in-use, idle, wait count and close counters
- Liveness/readiness endpoints
- Self-contained embedded HTML dashboard
- Browser-side history charts without external JS libraries
- Graceful shutdown

## Requirements

- Go 1.24+
- Linux/macOS/Windows supported by `gopsutil`; dashboard is most useful on Linux
- PostgreSQL is optional

## Run without PostgreSQL

```bash
go mod download
go run ./cmd/server
```

Open:

```text
http://localhost:8080
```

The database card will show `NOT CONFIGURED`.

## Run with PostgreSQL

```bash
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable'
go run ./cmd/server
```

Optional tuning:

```bash
export DB_MAX_OPEN_CONNS=20
export DB_MAX_IDLE_CONNS=10
export DB_CONN_MAX_LIFETIME=30m
export DB_PING_TIMEOUT=2s
export SYSTEM_METRIC_INTERVAL=2s
```

## API

| Endpoint | Purpose |
| --- | --- |
| `GET /` | HTML dashboard |
| `GET /api/snapshot` | Combined system + DB snapshot |
| `GET /api/system` | System/runtime metrics |
| `GET /api/database` | DB status and pool metrics |
| `GET /api/health` | Overall health |
| `GET /health/live` | Process liveness |
| `GET /health/ready` | Readiness including configured DB |

## Example architecture

```text
Browser dashboard
      |
      | GET /api/snapshot every 2s
      v
Go HTTP server
      |
      +--> SystemCollector goroutine --> CPU / RAM / disk / load / runtime
      |
      +--> DatabaseMonitor -----------> PostgreSQL ping + sql.DB stats
```

The `SystemCollector` intentionally runs in a background goroutine. It periodically refreshes a cached snapshot protected by `sync.RWMutex`, so HTTP requests do not perform all server metric work themselves.

## Test

```bash
make test
make vet
```

## Build

```bash
make build
./bin/server-monitor
```

## Learning path

1. Inspect `internal/monitor/system.go` for metric collection.
2. Inspect `internal/monitor/database.go` for dependency health and connection-pool statistics.
3. Inspect `internal/httpapi` for health/API endpoints.
4. Inspect `internal/webui/static/index.html` for polling and visualization.
5. Next, replace in-memory/browser history with Prometheus and Grafana.
6. Later, add OpenTelemetry traces and centralized logging.
