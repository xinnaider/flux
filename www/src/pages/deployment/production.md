---
layout: "@layouts/DocLayout.astro"
title: Production
description: Production deployment considerations.
---

## Architecture

For production, run multiple flux instances behind a TCP/HTTP load balancer:

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

## Health Checks

Use the `/health` endpoint for load balancer health probes. flux is stateless — if it's running and can reach Redis, it's healthy.

## Scaling

- **Horizontal**: Add more flux instances behind your load balancer
- **Redis**: Use Redis Cluster or Sentinel for high availability
- **No sessions**: flux doesn't store client state; no affinity needed

## Resource Requirements

flux is lightweight. A single instance handles thousands of requests per second with minimal CPU/memory.

| Resource | Estimate |
|----------|----------|
| Memory | ~20MB base + instance data |
| CPU | Minimal (mostly I/O bound) |
| Network | Small (JSON + 302 responses) |

## Monitoring

- Expose `/health` to your monitoring system
- Monitor Redis connection
- Set up alerts for eviction rates and registration failures
