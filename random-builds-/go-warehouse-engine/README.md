# Warehouse Inventory Engine

**Stack:** Go `net/http` · PostgreSQL · `pgx/v5` · No ORM · Raw SQL

## Project Structure

```
warehouse-engine/
├── cmd/server/main.go                   # entry point, wires everything together
├── api/handler/inventory.go             # HTTP handlers (net/http ServeMux)
├── internal/
│   ├── config/config.go                 # env-based config
│   ├── db/postgres.go                   # pgxpool setup
│   ├── domain/                          # pure Go structs, no DB tags
│   │   ├── product.go
│   │   └── reservation.go
│   ├── repository/                      # raw SQL queries via pgx
│   │   ├── product.go
│   │   └── reservation.go
│   ├── service/inventory.go             # business logic, retry loop, channels
│   └── worker/
│       ├── ttl_watcher.go               # background goroutine: releases expired holds
│       └── reorder_worker.go            # channel listener: logs low-stock alerts
├── migrations/001_init.sql
├── docker-compose.yml
├── Dockerfile
└── README.md
```

## Core Concepts in This Codebase

| Concept | File | What to Study |
|---|---|---|
| Optimistic locking | `repository/product.go` | `ReserveStock` — version column in WHERE |
| Retry loop on conflict | `service/inventory.go` | `Reserve` — 5 retries with re-read |
| Compensating transaction | `service/inventory.go` | `Reserve` — ReleaseStock if Create fails |
| TTL background goroutine | `worker/ttl_watcher.go` | `time.Ticker` + `ctx.Done()` select |
| Reorder alert channel | `worker/reorder_worker.go` | buffered channel fan-out |
| Graceful shutdown | `cmd/server/main.go` | signal.Notify + cancel() + srv.Shutdown |
| Raw SQL (no ORM) | `repository/*.go` | all queries written by hand |

## Quick Start

### Option A — Docker Compose (easiest)

```bash
docker compose up --build
```

Both PostgreSQL and the Go app start together. Migrations run automatically.

### Option B — Run locally

```bash
# 1. Start PostgreSQL
docker compose up postgres -d

# 2. Apply schema + seed data
psql postgres://postgres:postgres@localhost:5432/warehouse -f migrations/001_init.sql

# 3. Download dependencies
go mod tidy

# 4. Run
go run ./cmd/server
```

## API Reference

### Health check

```bash
curl http://localhost:8080/health
# {"status":"ok"}
```

### Get seeded product ID

```bash
psql postgres://postgres:postgres@localhost:5432/warehouse \
  -c "SELECT id, sku, total_stock, reserved FROM products;"
```

### Reserve stock

```bash
curl -X POST http://localhost:8080/reserve \
  -H "Content-Type: application/json" \
  -d '{
    "product_id": "<UUID from above>",
    "order_id":   "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
    "quantity":   3
  }'
# Returns: { "id": "<reservation_id>", "expires_at": "...", "status": "pending" }
```

### Confirm (payment done — permanently deducts stock)

```bash
curl -X POST http://localhost:8080/confirm \
  -H "Content-Type: application/json" \
  -d '{"reservation_id": "<reservation_id>"}'
```

### Cancel (release held stock back to available)

```bash
curl -X POST http://localhost:8080/cancel \
  -H "Content-Type: application/json" \
  -d '{"reservation_id": "<reservation_id>"}'
```

## How the Optimistic Lock Works

```
1. GET product WHERE id = $1
   → returns { total_stock:100, reserved:5, version:7 }

2. available = 100 - 5 = 95  ✓ enough for qty=3

3. UPDATE products
   SET reserved = reserved + 3, version = version + 1
   WHERE id = $1 AND version = 7          ← guard
     AND (total_stock - reserved) >= 3    ← safety check

4a. rows_affected = 1  → success, insert reservation row
4b. rows_affected = 0  → another request changed version since step 1
                         → retry from step 1 (up to 5 times)
```

No DB-level row locks. No deadlocks possible.

## Environment Variables

| Variable | Default |
|---|---|
| `PORT` | `8080` |
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/warehouse?sslmode=disable` |
