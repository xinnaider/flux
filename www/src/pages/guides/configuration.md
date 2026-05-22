---
layout: "@layouts/DocLayout.astro"
title: Configuration
description: Environment variables and runtime options.
---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP listen port |
| `REDIS_ADDR` | `localhost:6379` | Redis server address |
| `REDIS_PASSWORD` | (empty) | Redis auth password |
| `REDIS_DB` | `0` | Redis database number |
| `HEARTBEAT_TTL` | `15s` | Time-to-live for registered instances |
| `CLEANUP_INTERVAL` | `5s` | How often to check for expired instances |
| `REQUEST_TIMEOUT` | `30s` | HTTP read/write timeout |
| `PROXY_MODE` | `false` | Enable reverse proxy mode (instead of 302 redirect) |
| `PROXY_TIMEOUT` | `30s` | Backend connection timeout in proxy mode |
| `PROXY_IDLE_CONNS` | `100` | Max idle connections in the transport pool |
| `PROXY_IDLE_PER_HOST` | `10` | Max idle connections per backend host |

## Examples

### Redirect mode (default, internal networks)

```bash
export PORT=9090
export REDIS_ADDR=redis-cluster:6379
export HEARTBEAT_TTL=30s
export PROXY_MODE=false

./bin/flux
```

### Proxy mode (behind Nginx, HTTPS)

```bash
export PORT=8080
export REDIS_ADDR=redis:6379
export PROXY_MODE=true
export PROXY_IDLE_CONNS=200
export PROXY_IDLE_PER_HOST=50

./bin/flux
```

## Default Port

flux listens on `8080` by default. Override with `PORT`:

```bash
PORT=9090 ./bin/flux
```
