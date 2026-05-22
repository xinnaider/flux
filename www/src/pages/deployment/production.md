---
layout: "@layouts/DocLayout.astro"
title: Production
description: Production deployment considerations.
---

## Architecture

### Redirect mode (internal services)

Run multiple flux instances behind a TCP/HTTP load balancer. All instances share the same Redis backend.

```
        ┌──────────────┐
        │  Load        │
        │  Balancer    │
        └──┬────┬────┬──┘
           │    │    │
     ┌─────┘    │    └─────┐
     ▼          ▼          ▼
  ┌────────┐┌────────┐┌────────┐
  │  flux  ││  flux  ││  flux  │
  │  :8080 ││  :8080 ││  :8080 │
  └───┬────┘└───┬────┘└───┬────┘
      │         │         │
      └─────────┼─────────┘
                ▼
          ┌──────────┐
          │  Redis   │
          │  Cluster │
          └──────────┘
```

### Proxy mode (public APIs with HTTPS)

Place Nginx in front of flux for TLS termination and reverse proxy to backends.

```
[Client] --HTTPS :443--> [Nginx] --HTTP :8080--> [flux (proxy mode)] --HTTP--> [Backend]
```

Nginx handles SSL, flux handles service discovery and load balancing.

#### Docker Compose

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

#### Nginx Configuration

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

        proxy_set_header Host                   $host;
        proxy_set_header X-Real-IP              $remote_addr;
        proxy_set_header X-Forwarded-For        $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto      $scheme;
        proxy_set_header X-Forwarded-Host       $host;
        proxy_set_header Connection             "";

        proxy_connect_timeout   10s;
        proxy_send_timeout      30s;
        proxy_read_timeout      30s;

        proxy_buffering         on;
        proxy_buffer_size       4k;
        proxy_buffers           8 4k;
    }
}
```

## Health Checks

Use the `/health` endpoint for load balancer health probes. flux is stateless — if it's running and can reach Redis, it's healthy.

## Scaling

- **Horizontal**: Add more flux instances behind your load balancer or Nginx upstream
- **Redis**: Use Redis Cluster or Sentinel for high availability
- **No sessions**: flux doesn't store client state; no affinity needed
- **Proxy mode throughput**: ~7.5k req/s per flux instance (redirect mode: ~10k req/s)

## Resource Requirements

flux is lightweight. A single instance handles thousands of requests per second with minimal CPU/memory.

| Resource | Estimate |
|----------|----------|
| Memory | ~20MB base (redirect) / ~50-100MB (proxy, buffering) |
| CPU | Minimal in redirect mode; moderate in proxy mode (byte copying) |
| Network | Redirect: small (JSON + 302). Proxy: full request/response bandwidth |

## Monitoring

- Expose `/health` to your monitoring system
- Monitor Redis connection
- Set up alerts for eviction rates and registration failures
- In proxy mode, monitor backend response times via X-Forwarded-* headers
