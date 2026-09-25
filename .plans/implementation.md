You are implementing a compact but production-style Go project for learning advanced Go concepts.

Do not only create a plan or markdown documentation. Implement the complete working project, tests, benchmarks, and verification.

# Goal

Build a high-performance **Concurrent Service Health Checker**.

The system accepts many HTTP endpoints, checks them concurrently, aggregates results, exposes metrics, supports cancellation/timeouts, limits concurrency, and shuts down gracefully.

The purpose is not CRUD functionality. The purpose is to exercise advanced and expert-level Go concepts in a small codebase.

Target completion scope: roughly 1 hour of focused implementation.

Use only the Go standard library unless an external dependency provides significant learning value.

---

# Functional API

Implement:

```http
POST /api/v1/check
```

Request:

```json
{
  "urls": [
    "https://google.com",
    "https://github.com",
    "https://example.com"
  ],
  "workers": 10,
  "timeout_ms": 3000
}
```

Response example:

```json
{
  "request_id": "abc123",
  "total": 3,
  "healthy": 2,
  "failed": 1,
  "duration_ms": 420,
  "results": [
    {
      "url": "https://google.com",
      "status_code": 200,
      "latency_ms": 120,
      "healthy": true
    }
  ]
}
```

Also implement:

```http
GET /health
GET /metrics
```

`/metrics` does not need Prometheus. A simple JSON endpoint is sufficient.

Example:

```json
{
  "requests_total": 100,
  "checks_total": 3000,
  "checks_failed": 42,
  "active_requests": 3,
  "average_latency_ms": 140
}
```

---

# Architecture

Keep the codebase intentionally small but properly separated.

Use approximately:

```text
cmd/server/main.go

internal/checker/
    checker.go
    worker_pool.go

internal/service/
    service.go

internal/httpapi/
    handler.go
    middleware.go

internal/metrics/
    metrics.go
```

Tests should live beside their packages.

Do not introduce unnecessary repository/database abstractions because this project does not require persistence.

---

# Advanced Go Concepts That MUST Be Used

## 1. Context propagation

The incoming HTTP request context must propagate through:

```text
HTTP handler
    -> service
        -> worker pool
            -> HTTP request
```

Use:

```go
context.Context
```

Every outbound HTTP request must use:

```go
http.NewRequestWithContext
```

Support:

* client cancellation
* server-side timeout
* worker cancellation
* shutdown cancellation

Never replace request contexts with `context.Background()` inside business logic.

---

## 2. Worker pool

Do not create unlimited goroutines.

Implement a bounded worker pool using channels.

Conceptually:

```text
jobs chan Job

worker 1 ─┐
worker 2 ─┼──> results chan Result
worker N ─┘
```

Workers must stop when:

* jobs channel closes
* context is cancelled

The worker count must be configurable per request but constrained by a server-defined maximum.

Example:

```go
const MaxWorkers = 50
```

Reject or clamp invalid values consistently.

---

## 3. Generics

Use generics somewhere meaningful rather than artificially.

For example, implement a reusable concurrent processing helper:

```go
type Processor[I any, O any] interface {
    Process(context.Context, I) O
}
```

or:

```go
func RunPool[I any, O any](
    ctx context.Context,
    workers int,
    jobs []I,
    fn func(context.Context, I) O,
) []O
```

Use it for URL checks.

The abstraction must remain simple and understandable.

---

## 4. Channels

Demonstrate correct use of:

```go
chan T
<-chan T
chan<- T
```

Use directional channels where appropriate.

Avoid:

* deadlocks
* goroutine leaks
* sending after close
* multiple owners closing the same channel

Clearly define which goroutine owns channel closure.

---

## 5. sync.WaitGroup

Use `sync.WaitGroup` for worker lifecycle management.

The result channel should only be closed after all workers finish.

Implement the ownership correctly.

---

## 6. Atomic metrics

Use:

```go
sync/atomic
```

for counters such as:

```text
requests_total
checks_total
checks_failed
active_requests
```

Do not use a mutex for simple counters.

Use `atomic.Int64` / appropriate typed atomics if supported by the selected Go version.

---

## 7. Mutex-protected state

Use a mutex for state that cannot be represented safely as one atomic counter.

For example, maintain rolling latency statistics:

```go
type Stats struct {
    mu sync.RWMutex

    totalLatency time.Duration
    samples      int64
}
```

Demonstrate correct use of:

```go
sync.Mutex
sync.RWMutex
```

where meaningful.

Do not add locks purely for demonstration.

---

## 8. HTTP connection pooling

Create ONE reusable:

```go
http.Client
```

with a configured:

```go
http.Transport
```

Configure useful values such as:

```text
MaxIdleConns
MaxIdleConnsPerHost
IdleConnTimeout
TLSHandshakeTimeout
ResponseHeaderTimeout
```

Do not create a new `http.Client` for every URL.

The checker should reuse TCP connections where possible.

---

## 9. Timeout layering

Demonstrate the difference between:

```text
request context timeout
HTTP transport timeout
server read/write/idle timeouts
```

Implement them properly.

Example:

```go
ctx, cancel := context.WithTimeout(...)
defer cancel()
```

Avoid timeout configuration that causes obvious conflicting behavior.

---

## 10. Graceful shutdown

Handle:

```text
SIGINT
SIGTERM
```

using:

```go
signal.NotifyContext
```

or the appropriate modern Go pattern.

When shutting down:

1. stop accepting new requests
2. allow current requests to finish
3. use a bounded shutdown timeout
4. cancel remaining work
5. close idle HTTP client connections

Use:

```go
server.Shutdown(ctx)
```

Do not terminate directly with `os.Exit` during normal shutdown.

---

# Error Handling

Create meaningful sentinel/custom errors where useful.

Use:

```go
errors.Is
errors.As
fmt.Errorf("...: %w", err)
```

Do not compare error strings.

Differentiate at least:

```text
invalid URL
timeout
context cancellation
DNS/network failure
HTTP failure response
```

An HTTP 500 response from a checked service is a completed request but should be considered unhealthy.

---

# URL Validation / Security

Only support:

```text
http
https
```

Reject unsupported schemes.

Since this is a learning project, keep SSRF prevention simple but document that production systems must restrict internal/private destinations.

Do not overbuild security infrastructure.

---

# Result Ordering

Concurrency should not produce nondeterministic API output.

Results must appear in the same order as the input URLs.

A worker can therefore process something like:

```go
type Job struct {
    Index int
    URL   string
}
```

and write results into their original positions.

Implement this without race conditions.

---

# Middleware

Implement lightweight middleware for:

```text
request ID
access logging
panic recovery
request timing
```

Request IDs can be generated without pulling a UUID package if unnecessary.

Log with:

```go
log/slog
```

Use structured logging.

Example fields:

```text
request_id
method
path
status
duration
```

---

# HTTP Server Configuration

Do not use:

```go
http.ListenAndServe(":8080", handler)
```

directly.

Create:

```go
http.Server
```

and configure appropriate:

```text
ReadHeaderTimeout
ReadTimeout
WriteTimeout
IdleTimeout
```

---

# Input Limits

Protect the server from unreasonable requests.

Examples:

```text
maximum 500 URLs/request
maximum request body size
worker count maximum
timeout minimum/maximum
```

Use:

```go
http.MaxBytesReader
```

where appropriate.

Return useful HTTP status codes.

---

# JSON Handling

Use:

```go
json.Decoder
```

and reject unknown request fields:

```go
decoder.DisallowUnknownFields()
```

Reject malformed JSON and multiple JSON values in one request.

---

# Concurrency Correctness

Pay special attention to:

```text
goroutine lifecycle
channel ownership
context cancellation
concurrent metrics access
result writes
shutdown behavior
```

The implementation must pass the Go race detector.

---

# Tests

Write meaningful tests, not placeholder tests.

At minimum cover:

## Checker tests

Use:

```go
httptest.Server
```

Test:

* HTTP 200
* HTTP 500
* slow server
* request timeout
* context cancellation
* invalid URL

---

## Worker pool tests

Test:

* correct number of results
* result ordering
* context cancellation
* worker count behavior
* no deadlock when jobs fail

---

## Handler tests

Test:

* valid request
* malformed JSON
* unsupported fields
* too many URLs
* invalid workers
* invalid timeout
* context timeout

Use:

```go
httptest.NewRecorder
```

---

## Metrics tests

Verify atomic counters and latency aggregation.

Where practical, test metrics concurrently.

---

# Benchmark

Add at least one benchmark such as:

```go
BenchmarkCheckerSequential
BenchmarkCheckerWorkerPool
```

or:

```go
BenchmarkWorkerPool
```

Compare concurrency levels such as:

```text
1
5
10
25
50 workers
```

Avoid fake benchmarks where the work is too trivial to measure anything useful.

---

# Optional advanced experiment

If implementation remains clean, add a semaphore-based alternative using a buffered channel:

```go
sem := make(chan struct{}, limit)
```

Compare it conceptually with the worker-pool implementation.

Do not replace the worker pool requirement.

---

# Makefile

Add useful commands:

```make
run:
	go run ./cmd/server

test:
	go test ./...

race:
	go test -race ./...

bench:
	go test -bench=. -benchmem ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

verify:
	go test ./...
	go test -race ./...
	go vet ./...
```

---

# README

Keep README concise.

Include:

```text
architecture
important concurrency concepts
how cancellation flows through the system
channel ownership
worker lifecycle
HTTP connection reuse
how to run
curl example
tests
benchmark commands
```

Include a simple architecture diagram:

```text
Client
  |
  v
HTTP Handler
  |
  v
Service
  |
  v
Generic Worker Pool
  |
  +----------+----------+
  |          |          |
Worker 1   Worker 2   Worker N
  |          |          |
  +----------+----------+
             |
             v
        HTTP Client
             |
             v
       Remote Services
```

Do not turn the README into a long tutorial.

---

# Important Implementation Rules

Prefer idiomatic Go.

Avoid unnecessary abstractions.

Do not add:

```text
ORM
database
Docker/Kubernetes
message queue
Redis
frameworks
dependency injection frameworks
large third-party libraries
```

The focus is Go itself.

Avoid global mutable state.

Dependencies should be constructed in `main` and passed explicitly.

Interfaces should exist only where they improve testing or separation.

Do not create interfaces with only speculative future use.

---

# Verification

After implementation run:

```bash
gofmt -w .
go test ./...
go test -race ./...
go vet ./...
go test -bench=. -benchmem ./...
```

Fix every compilation error, test failure, vet issue, or race detector issue.

Also start the server and manually verify:

```bash
curl http://localhost:8080/health
```

and:

```bash
curl -X POST http://localhost:8080/api/v1/check \
  -H 'Content-Type: application/json' \
  -d '{
    "urls": [
      "https://google.com",
      "https://github.com",
      "https://example.com"
    ],
    "workers": 3,
    "timeout_ms": 3000
  }'
```

Then verify:

```bash
curl http://localhost:8080/metrics
```

---

# Final Review

Before finishing, inspect the entire diff.

Specifically review for:

```text
goroutine leaks
data races
incorrect channel closing
WaitGroup misuse
context not propagated
context cancellation ignored
new HTTP client created per request
response bodies not closed
unbounded concurrency
incorrect timeout handling
writes to shared slices/maps without synchronization
shutdown races
test flakiness
unnecessary abstractions
```

Fix discovered issues rather than merely documenting them.

Finally provide:

1. files created/changed
2. architecture summary
3. advanced Go concepts actually used
4. tests implemented
5. benchmark results
6. commands executed for verification
7. any remaining limitations

The task is complete only when the application code itself is implemented and verified. Do not stop after producing a plan or README.
