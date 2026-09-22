# Order Delivery System Documentation

This folder documents the current implementation of the order-delivery system as it exists in code today.

## Documents

1. `system-overview.md`
   Full architecture, request flow, folder structure, business rules, assignment logic, state transitions, data model, frontend behavior, and realtime flow.

2. `api-reference.md`
   HTTP endpoints, request/response examples, and endpoint-level implementation notes.

3. `known-issues.md`
   Current bugs, mismatches, technical risks, and recommended change areas found while reading the system.

## Scope

This documentation reflects the current codebase, not an idealized design. Where the implementation has gaps or mismatches, those are called out explicitly in `known-issues.md`.

## Source Map

- Entry point: `cmd/server/main.go`
- Routing: `internal/router/router.go`
- Handlers: `internal/handlers/`
- Services: `internal/service/`
- Repositories: `internal/repository/`
- Models: `internal/models/models.go`
- Realtime hub: `internal/websocket/hub.go`
- Schema: `migrations/001_init.sql`
- Seed data: `scripts/seed.sql`
- Frontend: `public/index.html`
