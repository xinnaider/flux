# flux

**Service Registry + Redirect Load Balancer.**

flux stores service instances in Redis, receives heartbeat load from each instance, and returns HTTP `302` redirects to the least-loaded healthy target. It is **not** a reverse proxy — application payload goes directly from client to instance.

---

## Quick Start

### With Docker Compose (recommended for development)

```bash
git clone https://github.com/xinnaider/flux
cd flux
docker compose up -d

# Check it's alive
curl http://localhost:8080/health
# {"status":"ok"}
```

Register a service instance:

```bash
curl -X POST http://localhost:8080/register \
  -H "Content-Type: application/json" \
  -d '{"name":"ms.auth","host":"10.0.0.5","port":3001,"health_url":"/health"}'

# {"instance_id":"10.0.0.5:3001","ttl_seconds":15}
```

Send heartbeats (every ~10s) with real load:

```bash
curl -X POST http://localhost:8080/heartbeat \
  -H "Content-Type: application/json" \
  -d '{"name":"ms.auth","instance_id":"10.0.0.5:3001","active_connections":3}'

# {"ok":true,"ttl_seconds":15}
```

Redirect a client to the least-loaded instance:

```bash
curl -v http://localhost:8080/ms.auth/login
# HTTP/1.1 302 Found
# Location: http://10.0.0.5:3001/login
```

---

## API

### `POST /register` — add an instance

| Field | Type | Required |
|-------|------|----------|
| `name` | string | yes |
| `host` | string | yes |
| `port` | int | yes |
| `health_url` | string | no |

Returns `{"instance_id":"<host>:<port>","ttl_seconds":15}`.

### `POST /heartbeat` — refresh load + TTL

| Field | Type | Required |
|-------|------|----------|
| `name` | string | yes |
| `instance_id` | string | yes |
| `active_connections` | int | yes |

The registry resets the 15s TTL. Missing 2-3 heartbeats = instance expires out.

### `GET /{service}/*` — redirect to least-loaded

Picks the instance with the fewest `active_connections` and returns `302 Found` with `Location: http://<host>:<port>/<path>`.

### `POST /release` — decrement load

```bash
curl -X POST http://localhost:8080/release \
  -H "Content-Type: application/json" \
  -d '{"name":"ms.auth","instance_id":"10.0.0.5:3001"}'
```

### `POST /unregister` — remove instance

```bash
curl -X POST http://localhost:8080/unregister \
  -H "Content-Type: application/json" \
  -d '{"name":"ms.auth","instance_id":"10.0.0.5:3001"}'
```

### `GET /health` — registry health

```bash
curl http://localhost:8080/health
# {"status":"ok"}
```

---

## Configuration

| Env | Default | Description |
|-----|---------|-------------|
| `PORT` | `8080` | HTTP listen port |
| `REDIS_ADDR` | `localhost:6379` | Redis address |
| `REDIS_PASSWORD` | — | Redis password |
| `REDIS_DB` | `0` | Redis DB number |
| `HEARTBEAT_TTL` | `15s` | Instance TTL (Go duration) |
| `CLEANUP_INTERVAL` | `5s` | Stale instance cleanup interval |
| `REQUEST_TIMEOUT` | `30s` | HTTP read/write timeout |

---

## Redis State

flux uses a small Redis key surface:

```
service:ms.auth:instances   SET     — instance IDs for the service
instance:ms.auth:10.0.0.5:3001  HASH  — host, port, health_url, connections
```

Instances are **hash entries** with a TTL. If the heartbeat stops, the hash expires and `Cleanup` removes the stale reference from the service set.

---

## Running Locally

```bash
# With Go (requires Redis)
go run ./cmd/server

# Or build first
go build -o bin/flux ./cmd/server
./bin/flux

# With Docker (build local)
docker build -t flux .
docker run -p 8080:8080 --network host flux
```

## GHCR Pull (private image)

The flux image is hosted on GitHub Container Registry. Authenticate with a GitHub PAT:

```bash
echo $PAT | docker login ghcr.io -u xinnaider --password-stdin
docker pull ghcr.io/xinnaider/flux
docker run -e REDIS_ADDR=host.docker.internal:6379 -p 8080:8080 ghcr.io/xinnaider/flux
```

---

## Tests

Requires Redis on `localhost:6379` (or `REDIS_TEST_ADDR`):

```bash
go test -v -race ./...
```

---

## How It Works

1. Service instances **register** with host/port.
2. Every ~10s they send a **heartbeat** with current `active_connections`.
3. When a client hits `GET /{service}/*`, flux queries Redis for all live instances, picks the one with the least connections, and replies with a **302 redirect**.
4. flux also **increments** the chosen instance's counter as a fallback between heartbeats.
5. Instances that stop heartbeating are dropped after TTL + cleanup.

The redirect means traffic flows client → instance directly. flux never touches the payload.

---

## Stack

| Layer | Technology |
|-------|-----------|
| Runtime | Go |
| State | Redis 7 |
| HTTP | `net/http` |
| Infra | Docker Compose |

---

## License

MIT
