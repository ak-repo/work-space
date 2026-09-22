# Order Delivery System Overview

## Purpose

This system manages delivery drivers and customer orders, automatically assigns orders to the best available driver, tracks order status changes, and pushes live updates to connected clients using Server-Sent Events (SSE).

The backend is written in Go using `net/http` and PostgreSQL with raw SQL through `lib/pq`. There is no ORM.

## High-Level Architecture

```text
Browser UI (public/index.html)
        |
        v
HTTP Router (internal/router/router.go)
        |
        v
Handlers (internal/handlers)
        |
        v
Services (internal/service)
        |
        +--> Assignment logic, state rules, SSE broadcasting
        |
        v
Repositories (internal/repository)
        |
        v
PostgreSQL

Realtime side path:
Services -> Hub (internal/websocket/hub.go) -> SSE endpoint `/events` -> Browser EventSource
```

## Folder Structure

```text
cmd/
  server/
    main.go                  # application bootstrap

internal/
  config/
    config.go               # config loading + env overrides
    config.yaml             # default runtime config
  handlers/
    driver_handler.go       # HTTP handlers for drivers
    order_handler.go        # HTTP handlers for orders
    sse_handler.go          # SSE streaming endpoint
    helpers.go              # JSON response helpers
  models/
    models.go               # domain models and API payload structs
  repository/
    db.go                   # PostgreSQL connection and pool config
    driver_repo.go          # driver SQL queries
    order_repo.go           # order SQL queries
  router/
    router.go               # route registration
    middleware.go           # logging + CORS middleware
  service/
    order_service.go        # order lifecycle and validation
    assignment_service.go   # driver assignment and batch routing
    priority_queue.go       # max-heap driver ranking
    haversine.go            # geographic distance calculation
  websocket/
    hub.go                  # SSE client registry and broadcaster

migrations/
  001_init.sql              # schema definition

scripts/
  seed.sql                  # seed drivers

public/
  index.html                # single-page dispatch dashboard
```

## Startup Flow

Application bootstrapping happens in `cmd/server/main.go`.

### Flow

1. Load config from `internal/config/config.yaml`.
2. Apply supported environment variable overrides.
3. Connect to PostgreSQL and configure connection pooling.
4. Create repositories.
5. Create the realtime hub.
6. Create services.
7. Create HTTP handlers.
8. Register routes.
9. Start the HTTP server.
10. Wait for `SIGINT` or `SIGTERM` and shut down gracefully.

### Example

```go
db, err := repository.Connect(cfg.Database)
driverRepo := repository.NewDriverRepository(db)
orderRepo := repository.NewOrderRepository(db)

hub := websocket.NewHub()

assignSvc := service.NewAssignmentService(driverRepo, orderRepo, hub, cfg.Assignment)
orderSvc := service.NewOrderService(orderRepo, driverRepo, assignSvc, hub, cfg.Defaults)

driverHandler := handlers.NewDriverHandler(driverRepo, orderRepo, cfg.Defaults)
orderHandler := handlers.NewOrderHandler(orderSvc, orderRepo, assignSvc)
sseHandler := handlers.NewSSEHandler(hub, cfg.SSE)

r := router.New(driverHandler, orderHandler, sseHandler)
```

## Configuration

Default config lives in `internal/config/config.yaml`.

### Current config groups

- `server`
- `database`
- `assignment`
- `sse`
- `defaults`

### Key runtime settings

```yaml
server:
  port: "8081"

assignment:
  max_search_radius_km: 50
  distance_weight: 0.7
  capacity_weight: 0.3
  distance_smoothing: 0.1

defaults:
  driver_capacity: 3
  order_priority: 1
```

### Supported environment overrides

From `internal/config/config.go`:

- `CONFIG_PATH`
- `HOST`
- `PORT`
- `DB_HOST`
- `DB_PORT`
- `DB_USER`
- `DB_PASSWORD`
- `DB_NAME`
- `DB_SSLMODE`

Only these fields are currently overridden via environment variables.

## Data Model

The core models are defined in `internal/models/models.go`.

### Driver

- `id`
- `name`
- `phone`
- `status`: `available | busy | unavailable`
- `current_lat`, `current_lng`
- `capacity`
- `active_orders`

### Order

- `id`
- `customer_name`, `customer_phone`
- `pickup_lat`, `pickup_lng`, `pickup_address`
- `delivery_lat`, `delivery_lng`, `delivery_address`
- `status`: `pending | assigned | picked_up | delivered | cancelled`
- `driver_id`
- `priority`
- `notes`
- `assigned_at`, `delivered_at`

### AssignmentResult

Returned by auto-assignment. It contains:

- the updated order
- the selected driver
- assignment distance
- assignment score

## Database Schema

Schema is created in `migrations/001_init.sql`.

### Tables

1. `drivers`
2. `orders`

### Relationship

`orders.driver_id` references `drivers.id`.

### Indexes

- `idx_orders_status`
- `idx_orders_driver_id`
- `idx_drivers_status`

## Routing

Routes are registered in `internal/router/router.go`.

### Current endpoints

- `GET /health`
- `GET /events`
- `GET /api/drivers`
- `POST /api/drivers`
- `GET /api/drivers/{id}`
- `PUT /api/drivers/{id}/location`
- `GET /api/drivers/{id}/orders`
- `GET /api/orders`
- `POST /api/orders`
- `GET /api/orders/{id}`
- `PATCH /api/orders/{id}/status`
- `POST /api/orders/batch-assign`

### Example route wiring

```go
mux.HandleFunc("/api/orders/", func(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	switch {
	case strings.HasSuffix(path, "/status") && r.Method == http.MethodPatch:
		oh.UpdateStatus(w, r)
	case r.Method == http.MethodGet:
		oh.GetByID(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
})
```

## Request Handling Model

The system follows a simple layered split.

### Handlers

Handlers decode JSON, extract path/query data, call services or repositories, and write API responses.

Examples:

- `OrderHandler.Create`
- `OrderHandler.UpdateStatus`
- `DriverHandler.Create`
- `DriverHandler.UpdateLocation`

### Services

Services contain business rules.

- `OrderService`: order creation, default priority handling, transition validation
- `AssignmentService`: best-driver selection, batch assignment, SSE events

### Repositories

Repositories contain all SQL statements.

- `OrderRepository`
- `DriverRepository`

## Order Creation Flow

Order creation is implemented in:

- `internal/handlers/order_handler.go`
- `internal/service/order_service.go`
- `internal/service/assignment_service.go`

### Actual flow

1. Client sends `POST /api/orders`.
2. Handler decodes `CreateOrderRequest`.
3. `OrderService.CreateAndAssign()` validates minimum required input.
4. Default priority is applied if request priority is `0`.
5. Order is inserted into the database with status `pending`.
6. Assignment service tries to find the best driver.
7. If assignment succeeds:
   the order becomes `assigned`
   the driver's `active_orders` increases
   an SSE update is broadcast
8. If assignment fails:
   the order still remains created as `pending`
   API still returns success with an order-only result

### Example

```go
order := &models.Order{
	ID:           uuid.New().String(),
	CustomerName: req.CustomerName,
	PickupLat:    req.PickupLat,
	PickupLng:    req.PickupLng,
	DeliveryLat:  req.DeliveryLat,
	DeliveryLng:  req.DeliveryLng,
	Status:       models.OrderPending,
	Priority:     req.Priority,
}

if err := s.orderRepo.Create(ctx, order); err != nil {
	return nil, fmt.Errorf("creating order: %w", err)
}

result, err := s.assignment.AssignDriver(ctx, order)
if err != nil {
	return &models.AssignmentResult{Order: *order}, nil
}
```

## Driver Assignment Logic

Driver assignment is implemented in:

- `internal/service/assignment_service.go`
- `internal/service/priority_queue.go`
- `internal/service/haversine.go`
- `internal/repository/driver_repo.go`

### Candidate selection

Only drivers matching both conditions are considered:

- `status = 'available'`
- `active_orders < capacity`

SQL source:

```sql
SELECT id, name, phone, status, current_lat, current_lng, capacity, active_orders, created_at, updated_at
FROM drivers
WHERE status = 'available' AND active_orders < capacity
ORDER BY active_orders ASC
```

### Scoring model

The system computes a score per driver using:

- distance to pickup location
- remaining capacity

Drivers outside `max_search_radius_km` are ignored.

### Score formula

```go
distScore := 1.0 / (dist + smoothing)
capacityScore := float64(d.Capacity-d.ActiveOrders) / float64(d.Capacity)
totalScore := (distScore * distanceWeight) + (capacityScore * capacityWeight)
```

### Selection structure

The ranking structure is a max-heap using `container/heap`.

```go
func (pq DriverPriorityQueue) Less(i, j int) bool {
	return pq[i].Score > pq[j].Score
}
```

### Concurrency guard

`AssignmentService` uses a `sync.Mutex` to avoid concurrent assignment attempts selecting the same driver in memory at the same time.

```go
s.mu.Lock()
defer s.mu.Unlock()
```

## Batch Assignment Flow

Batch assignment is handled by `AssignmentService.RouteBatch()`.

### Behavior

1. Fetch target driver.
2. Loop through requested order IDs.
3. Skip orders not found or not `pending`.
4. Stop when driver capacity is full.
5. Assign each eligible order.
6. Increment driver active order count.
7. Broadcast an order update event.

This endpoint does not optimize route sequence. It only assigns multiple pending orders to one selected driver.

## Order Status Lifecycle

Status rules are enforced in `internal/service/order_service.go`.

### Allowed transitions

```text
pending   -> assigned
pending   -> cancelled
assigned  -> picked_up
assigned  -> cancelled
picked_up -> delivered
```

### Validation code

```go
allowed := map[models.OrderStatus][]models.OrderStatus{
	models.OrderPending:   {models.OrderAssigned, models.OrderCancelled},
	models.OrderAssigned:  {models.OrderPickedUp, models.OrderCancelled},
	models.OrderPickedUp:  {models.OrderDelivered},
	models.OrderDelivered: {},
	models.OrderCancelled: {},
}
```

### Update side effects

When an order becomes `delivered` or `cancelled` and has a driver:

- driver `active_orders` is decremented
- order update SSE is broadcast

## Realtime Flow

Realtime support is implemented using SSE, not WebSocket, even though the hub package is named `websocket`.

### Components

- SSE endpoint: `internal/handlers/sse_handler.go`
- client hub: `internal/websocket/hub.go`
- browser subscription: `public/index.html`

### Server flow

1. Client connects to `GET /events`.
2. Server registers a buffered channel for that client.
3. Hub stores the client in a map protected by `sync.RWMutex`.
4. Services broadcast `models.StatusUpdate` payloads.
5. Each connected client receives the event stream.

### SSE connect example

```go
client := &websocket.Client{
	ID:      uuid.New().String(),
	Channel: make(chan []byte, h.clientChannelBuffer),
}
h.hub.Register(client)
defer h.hub.Unregister(client.ID)
```

### Broadcast example

```go
for _, c := range h.clients {
	select {
	case c.Channel <- data:
	default:
		log.Printf("WARN: SSE client %s channel full, skipping", c.ID)
	}
}
```

## Frontend Behavior

The UI is a single static HTML page in `public/index.html`.

### Current frontend responsibilities

- fetch all orders
- fetch all drivers
- create drivers
- create orders
- update driver location
- advance order status
- batch assign orders to one driver
- open an EventSource connection to `/events`
- render KPIs, tables, detail dialogs, and toast messages

### Frontend fetch examples

```js
const r = await fetch('/api/orders' + (f ? '?status=' + f : ''));
const d = await r.json();
```

```js
const r = await fetch(`/api/orders/${id}/status`, {
  method: 'PATCH',
  headers: {'Content-Type': 'application/json'},
  body: JSON.stringify({status})
});
```

## Middleware

The router wraps the mux with:

- logging middleware
- CORS middleware

### Logging

Requests are logged with method, path, status, and duration.

### CORS

Current CORS behavior allows all origins and common methods/headers.

## Deployment Notes

The local development stack is defined by `docker-compose.yml`.

### Services

1. `postgres`
2. `server`

### Seed data

Sample drivers are inserted by `scripts/seed.sql`.

## Current Implementation Boundaries

What the system currently does not implement explicitly:

- authentication or authorization
- DB transactions for multi-step assignment updates
- route optimization for batch assignments
- validation of all coordinate and payload edge cases
- DB-level status transition constraints
- automated tests in the current codebase

## Documentation Reading Order

For a new developer, the best reading order is:

1. `cmd/server/main.go`
2. `internal/router/router.go`
3. `internal/handlers/order_handler.go`
4. `internal/service/order_service.go`
5. `internal/service/assignment_service.go`
6. `internal/repository/order_repo.go`
7. `internal/repository/driver_repo.go`
8. `public/index.html`

For API details, continue with `api-reference.md`.
For bugs and recommended changes, continue with `known-issues.md`.
