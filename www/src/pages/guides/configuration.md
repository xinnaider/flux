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
| `HEARTBEAT_TTL` | `15s` | Time-to-live for registered instances |
| `EVICTION_INTERVAL` | `10s` | How often to check for expired instances |

## Example

```bash
export PORT=9090
export REDIS_ADDR=redis-cluster:6379
export REDIS_PASSWORD=secret
export HEARTBEAT_TTL=30s

./bin/flux
```

## Default Port

flux listens on `8080` by default. Override with `PORT`:

```bash
PORT=9090 ./bin/flux
```
