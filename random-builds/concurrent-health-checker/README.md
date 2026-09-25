# Concurrent Service Health Checker

A compact standard-library Go service that checks HTTP endpoints concurrently while preserving input order.

## Architecture

```text
Client
  |
  v
HTTP Handler -> Metrics
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
      Shared HTTP Client
             |
             v
       Remote Services
```

The request context flows from the handler through the service and worker pool into every outbound request. A per-request deadline limits the entire batch. Client disconnects, deadlines, and forced server shutdown therefore stop in-flight HTTP work.

The producer owns and closes the jobs channel. Workers only receive jobs and send results. A dedicated goroutine waits for every worker with `sync.WaitGroup` before closing the results channel. Only the collector writes the ordered output slice.

The process creates one `http.Client` and one configured `http.Transport`; workers reuse both, allowing connection pooling across checks. Atomic counters track request activity while an `RWMutex` protects aggregate latency state.

## Run

```bash
make run
```

```bash
curl http://localhost:8080/health

curl -X POST http://localhost:8080/api/v1/check \
  -H 'Content-Type: application/json' \
  -d '{
    "urls": ["https://example.com", "https://github.com"],
    "workers": 2,
    "timeout_ms": 3000
  }'

curl http://localhost:8080/metrics
```

`workers` defaults to 10 and is limited to 50. `timeout_ms` defaults to 3000 and must be between 50 and 30000. Requests are limited to 500 URLs and 64 KiB.

## Verify

```bash
make test
make race
make vet
make bench
```

The benchmark runs real local HTTP checks with 1, 5, 10, 25, and 50 workers.

## Security Note

URL validation permits only absolute `http` and `https` URLs. This learning project does not prevent SSRF; a production deployment must resolve and block loopback, link-local, private, and other restricted destinations according to its network policy.
