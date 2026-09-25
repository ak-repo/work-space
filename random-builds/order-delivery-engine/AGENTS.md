# AGENTS.md

## Setup Commands
- `docker-compose up --build` to start services
- `psql -h localhost -U postgres -d order_delivery -f scripts/seed.sql` to seed drivers

## Critical Rules
- Order status transitions:
  pending → assigned → picked_up → delivered
  (must incrementally update; no skipping steps)
- Update order status via `PATCH /api/orders/{id}/status` with valid status
- Driver assignment uses Haversine distances (max-heap based)

## System Notes
- No ORM: all SQL in `internal/repository`
- SSE events at `/events` using `sync.RWMutex`
- Assignment logic in `internal/service/assignment_service.go`