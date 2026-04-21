# Stock Market Simulation Service

A high-availability stock exchange REST API built for the **Remitly recruitment task**. Three Go application instances share state through Redis; Nginx load balances across them. `POST /chaos` kills one instance — the service stays up.

## Quick Start

```bash
./start.sh 8080        # Linux/Mac
start.bat 8080         # Windows

# Verify
curl http://localhost:8080/healthz
```

## Architecture

```
Client
  │
  ▼
Nginx :8080  (least_conn load balancer, proxy_next_upstream retry)
  │
  ├── app1:8080  ──┐
  ├── app2:8080  ──┤── Redis :6379  (single source of truth, RDB persistence)
  └── app3:8080  ──┘
```

**High availability mechanism:**
- All three app instances are stateless — Redis is the single source of truth
- `restart: unless-stopped` in Docker Compose restarts a killed instance within ~1s
- `POST /chaos` kills one instance; the other two immediately absorb all traffic
- Nginx `proxy_next_upstream error timeout` transparently retries on upstream errors
- Result: zero downtime for any request that hits a healthy instance

**Atomicity via Redis Lua scripts:**
All buy/sell operations run as atomic Redis Lua scripts (`EVALSHA`). No transaction can observe partial state, and concurrent buys of a limited stock correctly limit to available quantity — proven by an integration test with 50 goroutines.

## API Reference

| Method | Path | Request | Response | Errors |
|--------|------|---------|----------|--------|
| `GET` | `/stocks` | — | `{"stocks":[{"name":"AAPL","quantity":10}]}` | — |
| `POST` | `/stocks` | `{"stocks":[{"name":"AAPL","quantity":10}]}` | `{}` | 400 negative qty |
| `POST` | `/wallets/{id}/stocks/{name}` | `{"type":"buy"\|"sell"}` | `{}` | 404 not found, 400 out of stock |
| `GET` | `/wallets/{id}` | — | `{"id":"alice","stocks":[...]}` | — |
| `GET` | `/wallets/{id}/stocks/{name}` | — | `42` | — |
| `GET` | `/log` | — | `{"entries":[...]}` | — |
| `POST` | `/chaos` | — | `{"status":"chaos initiated"}` | — |
| `GET` | `/healthz` | — | `{"status":"ok"}` | — |

**Trade rules:**
- `POST /wallets/{id}/stocks/{name}` with `{"type":"buy"}` → decrements bank, increments wallet
- `POST /wallets/{id}/stocks/{name}` with `{"type":"sell"}` → decrements wallet, increments bank
- `404` if stock name was never registered via `POST /stocks`
- `400` if bank has 0 quantity (buy) or wallet has 0 quantity (sell)
- Wallets are created lazily on first trade; `GET /wallets/{id}` returns `{"id":"x","stocks":[]}` for unknown wallets

## Redis Data Model

```
HASH  bank:stocks            field=stock_name  → quantity
SET   bank:stock_names       members=stock_name  (O(1) existence check)
HASH  wallet:{id}:stocks     field=stock_name  → quantity
LIST  audit:log              elements=JSON LogEntry (append-only)
```

## Development

### Run locally without Docker

```bash
# Start Redis
docker run -p 6379:6379 redis:7.2-alpine

# Run app
cd app
REDIS_ADDR=localhost:6379 APP_PORT=8080 go run main.go
```

### Run tests

```bash
# Unit tests
make test-unit

# Integration tests (requires Docker)
make test-integration

# All tests
make test-all

# With coverage report
make cover
```

### Lint

```bash
make lint
```

## Design Decisions

**Why chi?** Zero-dependency, stdlib-compatible router with excellent middleware ecosystem. No reflection overhead, composable middleware. `chi.URLParam` is safe without goroutine-local state.

**Why Redis?** Provides native atomic primitives (Lua scripts, pipelining, `HINCRBY`). RDB snapshots survive container restarts. Horizontal Redis cluster would be the natural next step for further scaling.

**Why Lua scripts for buy/sell?** Redis executes Lua scripts atomically — no other command can interleave during execution. This eliminates all TOCTOU (check-then-act) race conditions without distributed locks. Scripts are pre-loaded via `EVALSHA` for efficiency.

**Why distroless runtime image?** ~10MB image, no shell, no package manager — minimal attack surface. The builder stage (Alpine) has all the tools; the runtime stage has only the binary and TLS certs.

**Why `proxy_next_upstream` without `non_idempotent`?** Safe retry on connection errors/timeouts without risk of double-executing a trade. Idempotent GETs retry freely; POSTs only retry if the upstream never responded.

**Trade-offs:**
- Redis is a single point of failure (mitigated: named volume persistence, automatic reconnect)
- Audit log append after trade success means a chaos event mid-append could lose one log entry (documented, acceptable for this task)
- `GET /wallets/{id}` returns `[]` not `404` for unknown wallets — matches spec intent (lazy creation)
