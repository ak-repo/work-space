# Order Delivery Routing Engine

A production-grade delivery assignment system in Go using `net/http` + PostgreSQL (`lib/pq`).

## Architecture

```
cmd/server/main.go          → entry point, wires everything together
internal/
  models/                   → shared structs (Order, Driver, etc.)
  repository/               → raw SQL queries via lib/pq (no ORM)
  service/
    assignment_service.go   → priority queue driver assignment (sync.Mutex)
    order_service.go        → order lifecycle + state machine
    priority_queue.go       → container/heap implementation (max-heap)
    haversine.go            → GPS distance calculation
  handlers/                 → net/http request handlers
  router/                   → mux + logging + CORS middleware
  websocket/hub.go          → SSE broadcast hub (sync.RWMutex)
migrations/                 → SQL schema
scripts/                    → seed data
```

## Quick Start

```bash
# 1. Start DB + server
docker-compose up --build

# 2. Seed drivers
psql -h localhost -U postgres -d order_delivery -f scripts/seed.sql

# 3. Register a driver
curl -X POST http://localhost:8080/api/drivers \
  -H "Content-Type: application/json" \
  -d '{"name":"Ali","phone":"9000000001","current_lat":8.524,"current_lng":76.936,"capacity":3}'

# 4. Create an order (auto-assigns nearest driver)
curl -X POST http://localhost:8080/api/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customer_name": "Ananda",
    "pickup_lat": 8.5241, "pickup_lng": 76.9366,
    "pickup_address": "MG Road, Trivandrum",
    "delivery_lat": 8.5300, "delivery_lng": 76.9400,
    "delivery_address": "Pattom, Trivandrum",
    "priority": 2
  }'

# 5. Watch real-time SSE updates
curl -N http://localhost:8080/events

# 6. Advance order status
curl -X PATCH http://localhost:8080/api/orders/<id>/status \
  -H "Content-Type: application/json" \
  -d '{"status":"picked_up"}'

# 7. Mark delivered
curl -X PATCH http://localhost:8080/api/orders/<id>/status \
  -H "Content-Type: application/json" \
  -d '{"status":"delivered"}'
```

## API Endpoints

| Method  | Endpoint                        | Description                        |
|---------|---------------------------------|------------------------------------|
| GET     | /health                         | Health check                       |
| GET     | /events                         | SSE real-time stream               |
| POST    | /api/drivers                    | Register driver                    |
| GET     | /api/drivers                    | List all drivers                   |
| GET     | /api/drivers/:id                | Get driver by ID                   |
| PUT     | /api/drivers/:id/location       | Update driver GPS                  |
| GET     | /api/drivers/:id/orders         | Driver's active orders             |
| POST    | /api/orders                     | Create order + auto-assign driver  |
| GET     | /api/orders                     | List orders (?status=pending)      |
| GET     | /api/orders/:id                 | Get order by ID                    |
| PATCH   | /api/orders/:id/status          | Update order status                |
| POST    | /api/orders/batch-assign        | Batch assign orders to one driver  |

## Order Status Flow

pending → assigned → picked_up → delivered
pending → cancelled
assigned → cancelled

## Key Concepts Used

- **Priority Queue (container/heap)**: Max-heap scores drivers by proximity + capacity
- **sync.Mutex**: Prevents two orders from claiming the same driver concurrently
- **sync.RWMutex**: SSE hub allows concurrent reads, exclusive writes
- **Haversine formula**: GPS distance calculation between coordinates
- **State machine**: isValidTransition() enforces order lifecycle rules
- **SSE (Server-Sent Events)**: Real-time push updates to connected clients
- **lib/pq**: Raw PostgreSQL driver, no ORM
