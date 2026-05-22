<p align="center">
  <img src="www/public/favicon.svg" width="120" height="120" alt="flux">
</p>

# flux

**Service Registry + Redirect / Reverse Proxy Load Balancer.**

flux stores service instances in Redis, receives heartbeat load from each instance, and routes traffic to the least-loaded healthy target. It supports two modes:

| Mode | Behaviour | Use case |
|------|-----------|----------|
| **Redirect** (default) | Returns HTTP `302` to the best instance. Payload goes client → backend directly. | Internal networks, same-VPC services |
| **Reverse Proxy** (`PROXY_MODE=true`) | Proxies the request to the best instance and returns the response. Payload goes client → flux → backend → flux → client. | Public-facing APIs behind Nginx/Traefik, HTTPS termination |

---

## Quick Start

### With Docker (public image, no clone needed)

```bash
docker run -d --name redis redis:7-alpine
docker run -d --name flux --link redis -e REDIS_ADDR=redis:6379 -p 8080:8080 ghcr.io/xinnaider/flux
curl http://localhost:8080/health
```

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

### `GET /{service}/*` — route to least-loaded

**Redirect mode** (default): returns `302 Found` with `Location: http://<host>:<port>/<path>`.

**Proxy mode** (`PROXY_MODE=true`): proxies the request to the best instance and returns its response directly. No redirect. Supports `X-Forwarded-For`, `X-Forwarded-Host`, `X-Forwarded-Proto`, and `X-Real-IP` headers.

```bash
# Redirect mode
curl -v http://localhost:8080/ms.auth/login
# HTTP/1.1 302 Found
# Location: http://10.0.0.5:3001/login

# Proxy mode
curl -v http://localhost:8080/ms.auth/login
# HTTP/1.1 200 OK
# {"hello":"world"}  ← response from backend
```

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
| `PROXY_MODE` | `false` | Enable reverse proxy mode (instead of 302 redirect) |
| `PROXY_TIMEOUT` | `30s` | Backend connection timeout in proxy mode |
| `PROXY_IDLE_CONNS` | `100` | Max idle connections in the transport pool |
| `PROXY_IDLE_PER_HOST` | `10` | Max idle connections per backend host |

---

## Production with Nginx

For public-facing APIs with HTTPS, place Nginx in front of flux with proxy mode enabled.

```
[Client] --HTTPS--> [Nginx :443] --HTTP--> [flux :8080] --HTTP--> [Backend]
```

### Nginx configuration

```nginx
upstream flux_upstream {
    server flux:8080;
    keepalive 64;
}

server {
    listen 443 ssl http2;
    server_name api.exemplo.com;

    ssl_certificate     /etc/ssl/certs/cert.pem;
    ssl_certificate_key /etc/ssl/certs/key.pem;

    location / {
        proxy_pass http://flux_upstream;
        proxy_http_version 1.1;

        # Headers
        proxy_set_header Host                   $host;
        proxy_set_header X-Real-IP              $remote_addr;
        proxy_set_header X-Forwarded-For        $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto      $scheme;
        proxy_set_header X-Forwarded-Host       $host;
        proxy_set_header Connection             "";

        # Timeouts
        proxy_connect_timeout   10s;
        proxy_send_timeout      30s;
        proxy_read_timeout      30s;

        # Buffering
        proxy_buffering         on;
        proxy_buffer_size       4k;
        proxy_buffers           8 4k;
    }
}
```

### Docker Compose (production-style)

```yaml
services:
  nginx:
    image: nginx:alpine
    ports: ["443:443", "80:80"]
    volumes:
      - ./nginx.conf:/etc/nginx/conf.d/default.conf:ro
      - ./certs:/etc/ssl/certs:ro
    depends_on: [flux]

  flux:
    image: ghcr.io/xinnaider/flux
    environment:
      PORT: "8080"
      REDIS_ADDR: "redis:6379"
      PROXY_MODE: "true"
      PROXY_IDLE_CONNS: "200"
      PROXY_IDLE_PER_HOST: "50"
    depends_on: [redis]

  redis:
    image: redis:7-alpine
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 5
```

### Metrics

In proxy mode, flux proxies ~7.5k req/s per instance (reduced from ~10k in redirect mode due to byte copying). Scale horizontally behind Nginx for more throughput:

```yaml
  flux-1:  # ... same config as above
  flux-2:  # ... same config
  flux-3:  # ... same config
```

Nginx's upstream will round-robin between them.

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
go run ./cmd/server
go build -o bin/flux ./cmd/server
./bin/flux
docker build -t flux .
docker run -p 8080:8080 --network host flux
```

## Public Image

Pull the official image without any authentication:

```bash
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
3. When a client hits `GET /{service}/*`, flux queries Redis for all live instances, picks the one with the least connections.
4. flux also **increments** the chosen instance's counter as a fallback between heartbeats.
5. Instances that stop heartbeating are dropped after TTL + cleanup.

The routing strategy depends on the mode:

- **Redirect mode** (`PROXY_MODE=false`, default): flux returns `302 Location: http://<host>:<port>/<path>`. Traffic flows client → instance directly. flux never touches the payload.
- **Proxy mode** (`PROXY_MODE=true`): flux proxies the entire request to the backend, reads the response, and returns it to the client. Payload goes through flux, but the client never sees internal addresses. Use behind Nginx for HTTPS.

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
