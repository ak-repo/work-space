

# High-Performance Go Trade Engine

A distributed, event-driven cryptocurrency exchange simulation built with **Golang**, **Apache Kafka**, and **Redis**. This engine is designed to handle ultra-high throughput with price-time priority matching logic.

## 🚀 Performance Summary

* **Core Engine Throughput:** ~8.8 Million orders/sec (In-memory)
* **Latency:** ~113.5 ns/op
* **Target Load:** 200,000 TPS (Achieved via asynchronous Kafka pipelining)

## 🏗 Architecture

The system follows the **Single-Writer / LMAX Disruptor** pattern to eliminate mutex contention and maximize CPU cache efficiency.

1. **API Ingestion:** A Gin-based REST API validates orders and publishes them to the `orders` Kafka topic.
2. **Sequencer (Kafka):** Ensures all orders are processed in the exact order they were received.
3. **Matching Engine:** A single-threaded worker consumes orders, maintains an in-memory **B-Tree Order Book**, and generates trades.
4. **Event Distribution:** Trade events are published to a `trades` topic.
5. **Real-time Analytics:** A consumer processes trade events to aggregate **OHLCV (Candlesticks)** and snapshots the order book into **Redis**.

## 🛠 Tech Stack

* **Language:** Go 1.20+
* **Data Structure:** Google B-Tree (for  matching)
* **Message Broker:** Apache Kafka
* **Cache/Store:** Redis (OHLCV & Book Snapshots)
* **Framework:** Gin Gonic (API)

## 📂 Project Structure

```text
.
├── cmd/
│   ├── api/             # REST API Entry point
│   ├── engine/          # Matching Engine Worker
│   └── load_tester/     # High-speed load generation script
├── internal/
│   ├── engine/          # Core Logic: B-Tree Orderbook & Matching
│   ├── kafka/           # Producer/Consumer logic
│   └── models/          # Order, Trade, and Candle structs
├── tests/               # Unit and Performance Benchmarks
└── docker-compose.yml   # Infrastructure (Kafka, Redis)

```

## 🚦 Getting Started

### 1. Prerequisites

* Docker & Docker Compose
* Go 1.20 or higher

### 2. Start Infrastructure

```bash
docker-compose up -d

```

### 3. Run the Engine

```bash
go run cmd/engine/main.go

```

### 4. Run the API

```bash
go run cmd/api/main.go

```

## 🧪 Testing & Benchmarks

### Core Engine Benchmark (In-Memory)

To verify the 113ns/op performance:

```bash
cd tests
go test -bench=. -cpu=1

```

### System Load Test (End-to-End)

To simulate 1,000,000 orders moving through Kafka:

```bash
go run cmd/load_tester/main.go

```

## 📊 API Documentation

| Endpoint | Method | Description |
| --- | --- | --- |
| `/order` | `POST` | Place a Limit/Market order |
| `/book/:pair` | `GET` | Get current order book (from Redis) |
| `/candles/:pair` | `GET` | Get latest OHLCV data |

## 🧠 Design Decisions

* **Why B-Tree?** Unlike a standard Map, the B-Tree keeps prices sorted, allowing us to find the "Best Bid" and "Best Ask" without scanning the entire memory.
* **Lock-Free Engine:** By using a single goroutine for the matching logic, we avoid the overhead of `sync.Mutex`, which is the primary reason for the high TPS.
* **Asynchronous OHLCV:** Candle aggregation happens outside the matching loop so that the engine never waits for Redis I/O.

