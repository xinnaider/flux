---
layout: "@layouts/DocLayout.astro"
title: POST /register
description: Register a service instance.
---

Register a new service instance with the registry.

## Request

```bash
curl -X POST http://localhost:8080/register \
  -H "Content-Type: application/json" \
  -d '{"name":"ms.auth","host":"10.0.0.5","port":3001,"health_url":"/health"}'
```

### Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | yes | Service name (used in URL path) |
| `host` | string | yes | Instance hostname or IP |
| `port` | int | yes | Instance port |
| `health_url` | string | no | Path for health checks (e.g. `/health`) |

## Response

```json
{"instance_id":"10.0.0.5:3001","ttl_seconds":15}
```

The instance is now registered with a 15-second TTL. Send heartbeats to keep it alive.
