# API Reference

This API reference describes the current HTTP behavior implemented in the codebase.

Base routes are registered in `internal/router/router.go`.

## Response Shape

Most JSON responses follow `models.APIResponse`:

```json
{
  "success": true,
  "data": {},
  "error": "",
  "message": ""
}
```

## Health

### `GET /health`

Returns a simple health payload.

### Example response

```json
{"status":"ok"}
```

## Realtime Events

### `GET /events`

Opens an SSE stream.

### Example server event format

```text
event: connected
data: {"client_id":"<uuid>"}

event: update
data: {"type":"order_update","status":"assigned","message":"..."}
```

## Drivers

### `POST /api/drivers`

Creates a driver.

### Request body

```json
{
  "name": "Ali",
  "phone": "9000000001",
  "current_lat": 8.524,
  "current_lng": 76.936,
  "capacity": 3
}
```

### Current rules

- `name` is required
- `phone` is required
- if `capacity == 0`, default capacity from config is used
- status is initialized to `available`
- `active_orders` starts at `0`

### Implementation path

- route: `internal/router/router.go`
- handler: `internal/handlers/driver_handler.go#Create`
- repository: `internal/repository/driver_repo.go#Create`

### `GET /api/drivers`

Returns all drivers ordered by `created_at DESC`.

### Example response data

```json
[
  {
    "id": "d1",
    "name": "Rajan Kumar",
    "phone": "9876543210",
    "status": "available",
    "current_lat": 8.5241,
    "current_lng": 76.9366,
    "capacity": 3,
    "active_orders": 0
  }
]
```

### `GET /api/drivers/{id}`

Returns one driver by ID.

### `PUT /api/drivers/{id}/location`

Updates driver GPS coordinates.

### Request body

```json
{
  "lat": 8.5300,
  "lng": 76.9400
}
```

### Current behavior

- returns success message only
- does not broadcast a driver update SSE event

### `GET /api/drivers/{id}/orders`

Returns active orders for the driver.

### Query behavior

Current SQL excludes:

- `delivered`
- `cancelled`

## Orders

### `POST /api/orders`

Creates an order and attempts automatic driver assignment.

### Request body

```json
{
  "customer_name": "Ananda",
  "customer_phone": "9000000010",
  "pickup_lat": 8.5241,
  "pickup_lng": 76.9366,
  "pickup_address": "MG Road, Trivandrum",
  "delivery_lat": 8.5300,
  "delivery_lng": 76.9400,
  "delivery_address": "Pattom, Trivandrum",
  "priority": 2,
  "notes": "Handle carefully"
}
```

### Current validation

The current service checks:

- `customer_name != ""`
- `pickup_lat != 0`
- `delivery_lat != 0`

Other fields are not strongly validated in the service today.

### Success behavior

Two success outcomes are possible:

1. Order is created and assigned
2. Order is created but not assigned because no suitable driver was found

### Assigned example response shape

```json
{
  "success": true,
  "data": {
    "order": {
      "id": "<order-id>",
      "status": "assigned",
      "driver_id": "<driver-id>"
    },
    "driver": {
      "id": "<driver-id>",
      "name": "Rajan Kumar",
      "distance_km": 1.24,
      "score": 1.93
    },
    "distance_km": 1.24,
    "score": 1.93
  }
}
```

### Unassigned example response shape

```json
{
  "success": true,
  "data": {
    "order": {
      "id": "<order-id>",
      "status": "pending"
    },
    "driver": {
      "id": "",
      "name": ""
    },
    "distance_km": 0,
    "score": 0
  }
}
```

The service currently returns `&models.AssignmentResult{Order: *order}` on assignment failure, so the response keeps the order but not a meaningful assigned driver.

### `GET /api/orders`

Returns all orders.

### Query parameter

Optional:

- `status`

Example:

```text
GET /api/orders?status=pending
```

When filtered, orders are sorted by:

1. `priority DESC`
2. `created_at ASC`

### `GET /api/orders/{id}`

Returns one order by ID.

### `PATCH /api/orders/{id}/status`

Advances or changes order status according to service transition rules.

### Request body

```json
{
  "status": "picked_up"
}
```

### Allowed transitions

```text
pending   -> assigned
pending   -> cancelled
assigned  -> picked_up
assigned  -> cancelled
picked_up -> delivered
```

### Side effects

- `delivered_at` is set when status becomes `delivered`
- driver active order count is decremented on `delivered` and `cancelled`
- order update SSE is broadcast when the order has a driver

### Example invalid transition error

```json
{
  "success": false,
  "error": "invalid transition: pending → delivered"
}
```

### `POST /api/orders/batch-assign`

Assigns multiple pending orders to one selected driver.

### Request body

```json
{
  "driver_id": "d1",
  "order_ids": ["o1", "o2", "o3"]
}
```

### Current behavior

- only pending orders are assigned
- assignment stops when driver capacity is reached
- skipped orders are silently ignored
- response only returns a success message, not assignment details per order

### Example response

```json
{
  "success": true,
  "message": "batch assignment done"
}
```

## Driver Assignment Implementation Notes

The automatic assignment path is:

1. `OrderHandler.Create`
2. `OrderService.CreateAndAssign`
3. `AssignmentService.AssignDriver`
4. `DriverRepository.GetAvailable`
5. `BuildDriverQueue`
6. `OrderRepository.AssignDriver`
7. `DriverRepository.ChangeActiveOrders`
8. `Hub.Broadcast`

## Sample cURL Commands

### Create driver

```bash
curl -X POST http://localhost:8080/api/drivers \
  -H "Content-Type: application/json" \
  -d '{
    "name":"Ali",
    "phone":"9000000001",
    "current_lat":8.524,
    "current_lng":76.936,
    "capacity":3
  }'
```

### Create order

```bash
curl -X POST http://localhost:8080/api/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customer_name":"Ananda",
    "pickup_lat":8.5241,
    "pickup_lng":76.9366,
    "pickup_address":"MG Road, Trivandrum",
    "delivery_lat":8.5300,
    "delivery_lng":76.9400,
    "delivery_address":"Pattom, Trivandrum",
    "priority":2
  }'
```

### Update status

```bash
curl -X PATCH http://localhost:8080/api/orders/<id>/status \
  -H "Content-Type: application/json" \
  -d '{"status":"picked_up"}'
```

### Watch events

```bash
curl -N http://localhost:8080/events
```

For implementation gaps and mismatches, see `known-issues.md`.
