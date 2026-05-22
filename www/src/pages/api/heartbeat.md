---
layout: "@layouts/DocLayout.astro"
title: POST /heartbeat
description: Refresh instance TTL and report load.
---

Send periodic heartbeats to keep an instance alive and report its current load.

## Request

```bash
curl -X POST http://localhost:8080/heartbeat \
  -H "Content-Type: application/json" \
  -d '{"name":"ms.auth","instance_id":"10.0.0.5:3001","active_connections":3}'
```

### Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | yes | Service name |
| `instance_id` | string | yes | `host:port` of the instance |
| `active_connections` | int | yes | Current connection count |

## Response

```json
{"ok":true,"ttl_seconds":15}
```

The instance TTL is refreshed. Continue sending heartbeats before the TTL expires.
