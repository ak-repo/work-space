# Operations Guide

## Normal local workflow

```bash
# terminal 1
cp .env.example .env
set -a; source .env; set +a
go run ./cmd/server

# terminal 2
go run ./cmd/simulator -count 20
```

Open `http://localhost:8080`.

## Switch from simulated Jev to live Jev

```bash
export JEV_MODE=live
export TYPESAFE_API_KEY='...'
export JEV_MODEL=jev-latest
go run ./cmd/server
```

Do not put the key in `web/app.js`, HTML, screenshots, Git, or frontend environment variables.

## What to inspect during a demo

1. Send a clear infrastructure incident such as an outage.
2. Inspect the returned `category`, `severity`, `needs_escalation`, and `next_action` answers.
3. Show that the Go policy may choose a final action independently of `next_action`.
4. Send an ambiguous event; the confidence/margin gate should send it to human review.
5. Send a security event; the policy routes it to `security_review` after confidence gates.
6. Run the simulator and watch SSE update the dashboard in real time.

## Reset local data

```bash
rm -rf data
```

The local state file is recreated automatically when new events arrive.

## Troubleshooting

### Live Jev returns 401

Verify `TYPESAFE_API_KEY`. Authentication/validation failures should not be blindly retried.

### Live Jev returns 429 or overload status

The current sample exposes the provider error rather than hiding it. For production, add bounded exponential backoff and a circuit breaker around retryable rate-limit/overload responses.

### Dashboard is open but does not update

Check `/health/ready`, then open browser network tools and verify `/api/v1/stream` remains connected as `text/event-stream`.

## Production hardening still required

This repository is a learning/demo project. Before real operations add:

- authentication and authorization
- outbound state redaction/PII policy
- bounded retries + circuit breaker
- OpenTelemetry/Prometheus metrics
- proper migration runner/versioning
- human review workflow
- LLM fallback adapter if required
- action-specific permissions and approval gates
- integration tests against a staging Jev account
