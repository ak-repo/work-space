# Jev AI Decision Control Center

Runnable Go demonstration of a **System-One decision pipeline**:

`event -> Jev -> typed decisions -> deterministic Go policy -> audited action -> live dashboard`

The runnable demo uses a durable JSON state store from the Go standard library, SSE for live dashboard updates, and supports two Jev modes. A PostgreSQL migration is included under `migrations/` as the production DB schema reference:

- `mock` — default; no API key/network required.
- `live` — calls TypeSafe's Jev `POST /v1/systemone` endpoint.

## Quick start

### 1. Prerequisites

- Go 1.23+
- Docker + Docker Compose (optional)
- Optional for live mode: TypeSafe/Jev API key

### 2. Run in mock mode

```bash
cp .env.example .env
set -a; source .env; set +a
go run ./cmd/server
```

Open: `http://localhost:8080`

Submit events from the UI or run:

```bash
go run ./cmd/simulator -count 20
```

The default store persists to `data/state.json`, so no external database or Go module download is required.

### 4. Run with real Jev

Create a TypeSafe API key and export it only on the backend:

```bash
export JEV_MODE=live
export TYPESAFE_API_KEY='your-key'
export JEV_MODEL='jev-latest'
go run ./cmd/server
```

The Go adapter sends one state with four typed questions:

1. `category` — Choice
2. `severity` — Score
3. `needs_escalation` — Noul
4. `next_action` — Choice

Jev's output is normalized into `domain.DecisionResult`. No Jev-specific DTO escapes `internal/provider/jev`.

## How processing works

1. `POST /api/v1/events` validates and stores the event.
2. `DecisionProvider.Decide` receives the event state and `incident_triage/v1` questions.
3. Mock or live Jev returns typed answers/probabilities.
4. `policy.Engine` applies deterministic gates:
   - low category confidence -> `human_review`
   - small top-choice margin -> `human_review`
   - security category -> `security_review`
   - escalation probability >= threshold -> `page_engineer`
   - otherwise -> `route_normal`
5. Decision answers and policy result are persisted in one transaction.
6. A safe demo action record is inserted.
7. The result is published over `/api/v1/stream` using SSE.
8. The dashboard refreshes live metrics and recent decisions.

## API

### Submit event

```bash
curl -X POST http://localhost:8080/api/v1/events \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: demo-001' \
  -d '{
    "source":"curl",
    "event_type":"service_alert",
    "title":"Database outage",
    "state":{
      "message":"Primary database unavailable; many users affected",
      "service":"orders"
    }
  }'
```

### Recent events

```bash
curl 'http://localhost:8080/api/v1/events?limit=20'
```

### Metrics

```bash
curl http://localhost:8080/api/v1/metrics/summary
```

### Health

```bash
curl http://localhost:8080/health/live
curl http://localhost:8080/health/ready
```

## Project structure

```text
cmd/server            HTTP server
cmd/simulator         demo event generator
internal/config       environment configuration
internal/domain       provider-independent domain models
internal/decision     decision provider interface + question set
internal/provider/jev real + mock Jev providers
internal/policy       deterministic policy engine
internal/store        standard-library persistent state store
internal/stream       SSE broker
internal/httpapi      REST/SSE handlers
web                   dashboard source
migrations            SQL migration copy for inspection
docs                   operation/design documentation
```

## Important design rule

**Jev makes a judgment; application code makes the final operational decision.**

Do not implement destructive actions directly from model output. Keep confidence thresholds, authorization, side-effect rules, and human-review requirements inside the Go policy/action layer.

## Configuration

| Variable | Default | Purpose |
|---|---|---|
| `HTTP_ADDR` | `:8080` | HTTP listener |
| `DATA_FILE` | `data/state.json` | durable local state file |
| `JEV_MODE` | `mock` | `mock` or `live` |
| `JEV_BASE_URL` | `https://api.typesafe.ai` | Jev base URL |
| `JEV_MODEL` | `jev-latest` | Jev model |
| `JEV_TIMEOUT` | `3s` | provider timeout |
| `TYPESAFE_API_KEY` | empty | live-mode credential |
| `POLICY_MIN_CATEGORY_CONFIDENCE` | `0.75` | auto-route gate |
| `POLICY_MIN_CHOICE_MARGIN` | `0.15` | ambiguity gate |
| `POLICY_ESCALATION_PROBABILITY` | `0.90` | escalation gate |

## Testing

```bash
go test ./...
```

Tests do not require Jev API access. Keep live model tests opt-in because API/network/model behavior can change.

## Current Jev API assumption

The live adapter follows the currently documented contract:

```text
POST https://api.typesafe.ai/v1/systemone
Authorization: Bearer <key>
Content-Type: application/json
```

Request: `model + state + questions`.
Question types: `choice`, `score`, `noul`.

Because Jev is new, keep this contract isolated in `internal/provider/jev/client.go` and verify the provider documentation before production use.
