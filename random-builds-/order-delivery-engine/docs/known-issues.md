# Known Issues And Change Areas

This section captures bugs, mismatches, and improvement areas found by reading the current implementation.

## 1. Port mismatch between docs, config, and Docker

### Status

Resolved.

### Current state

- `internal/config/config.yaml` default port is `8080`
- `docker-compose.yml` exposes and configures `8080`
- `README.md` examples use `8080`

### Impact

Running outside Docker now matches the documented examples.

### Files

- `internal/config/config.yaml`
- `docker-compose.yml`
- `README.md`

## 2. SSE event handling likely mismatched in the frontend

### Status

Resolved.

### Current state

Server sends named SSE events:

```text
event: update
data: ...
```

Frontend listens with:

```js
eventSource.addEventListener('update', e => { ... })
```

### Risk

Named events are now consumed with `addEventListener('update', ...)`.

### Files

- `internal/handlers/sse_handler.go`
- `public/index.html`

## 3. Order detail button likely breaks for UUID-based IDs

### Status

Resolved.

### Current state

The frontend renders:

```js
onclick='showOrderDetail(${JSON.stringify(o.id)})'
```

Order IDs are UUID strings, not JavaScript identifiers or numeric literals.

### Risk

The generated HTML now passes UUID strings safely.

### File

- `public/index.html`

## 4. Order validation is incomplete and coordinate-safe validation is missing

### Status

Resolved.

### Current state

`OrderService.CreateAndAssign()` validates:

- `customer_name`
- `pickup_lat`
- `pickup_lng`
- `delivery_lat`
- `delivery_lng`
- address completeness
- priority bounds
- coordinate bounds

It no longer treats `0` as missing, which is coordinate-safe.

### File

- `internal/service/order_service.go`

## 5. Assignment updates are not transactional

### Status

Resolved for assignment and terminal status updates.

### Current state

Assignment currently wraps these updates in one DB transaction:

1. assign driver to order
2. increment driver's active order count

### Risk

The order and driver state are updated atomically.

### Files

- `internal/service/assignment_service.go`
- `internal/repository/order_repo.go`
- `internal/repository/driver_repo.go`

## 6. Batch assignment also lacks transaction protection

### Status

Improved.

### Current state

Batch assignment loops and updates each order in its own transaction with a structured result.

### Risk

Partial assignment is still possible by design, but each order returns `assigned`, `skipped`, or `failed` with a reason.

### Files

- `internal/service/assignment_service.go`
- `internal/handlers/order_handler.go`

## 7. Batch assign route does not explicitly reject wrong methods

### Status

Resolved.

### Current state

`/api/orders/batch-assign` returns `405` on non-`POST` methods.

### Risk

Unexpected methods now return a clear method error.

### File

- `internal/router/router.go`

## 8. Batch assign errors are returned as server errors by default

### Status

Improved.

### Current state

The handler maps validation/not-found errors to `400` and server failures to `500`.

### Risk

Client-side problems are more clearly separated from real server failures.

### File

- `internal/handlers/order_handler.go`

## 9. No driver update SSE events for some driver-changing operations

### Status

Resolved for implemented driver-changing operations.

### Current state

The system broadcasts driver-specific updates when:

- location changes
- active order count changes
- driver creation changes fleet state

### Risk

The frontend refreshes whole datasets after both `order_update` and `driver_update` events.

### Files

- `internal/handlers/driver_handler.go`
- `internal/service/order_service.go`
- `internal/service/assignment_service.go`

## 10. Package naming is misleading for realtime support

### Current state

Realtime implementation is SSE, but the package is named `internal/websocket`.

### Risk

New developers may assume WebSocket behavior or protocols are implemented.

### File

- `internal/websocket/hub.go`

## 11. The create-order success contract is ambiguous when assignment fails

### Status

Improved.

### Current state

When assignment fails, the API still returns success because the order itself was created, and the frontend now shows that the order remains pending/unassigned.

### Risk

The included frontend now distinguishes "created and assigned" from "created but unassigned".

### File

- `internal/service/order_service.go`

## 12. No DB-level constraints for order state transitions

### Status

Partially resolved.

### Current state

Order transitions are enforced in service logic. The schema now also constrains allowed status values and priority bounds.

### Risk

Direct DB updates can no longer write unknown statuses, but full transition-order enforcement still lives in service logic.

### Files

- `internal/service/order_service.go`
- `migrations/001_init.sql`

## 13. No auth or access control

### Current state

All endpoints are open.

### Risk

Any client can create drivers, create orders, update locations, or change order statuses.

### Files

- `internal/router/router.go`
- `internal/handlers/`

## Recommended Next Changes

1. Add tests for status transitions and assignment scoring.
2. Consider renaming `internal/websocket` to `internal/realtime` because the implementation is SSE.
3. Add auth or access control before exposing this service outside a trusted environment.
4. Consider DB triggers if lifecycle transition enforcement must be guaranteed for direct SQL writes.
