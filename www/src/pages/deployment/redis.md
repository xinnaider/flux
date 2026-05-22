---
layout: "@layouts/DocLayout.astro"
title: Redis
description: Redis configuration for flux.
---

## Requirements

- Redis 6+ (Redis 7 recommended)
- No special modules required
- Persistence optional (flux uses ephemeral keys with TTL)

## Connection

flux connects to Redis using standard TCP. Configure via environment variables:

```bash
REDIS_ADDR=redis:6379
REDIS_PASSWORD=optional-password
```

## Key Namespace

All flux keys use the prefix `flux:` by default.

| Key Pattern | Type | Purpose |
|-------------|------|---------|
| `flux:{service}:{instance_id}` | Hash | Instance metadata + load |
| `flux:{service}:instances` | Set | Set of active instance IDs per service |

## TLS

To use TLS for Redis, configure via environment:

```bash
REDIS_ADDR=redis-tls:6380
REDIS_PASSWORD=secret
# flux currently supports plain TCP; wrap with stunnel for TLS
```

## Production Checklist

- Use a dedicated Redis instance or cluster
- Set `maxmemory` and appropriate eviction policy (e.g., `allkeys-lru`)
- Enable Redis persistence if you need to survive restarts
- Use Redis Sentinel or Redis Cluster for HA
