---
layout: "@layouts/DocLayout.astro"
title: POST /release
description: Decrement instance connection count.
---

Decrement the active connection count for an instance. Useful when a client finishes a request and you track load at the registry level.

## Request

```bash
curl -X POST http://localhost:8080/release \
  -H "Content-Type: application/json" \
  -d '{"name":"ms.auth","instance_id":"10.0.0.5:3001"}'
```

### Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | yes | Service name |
| `instance_id` | string | yes | `host:port` of the instance |

## Response

```json
{"ok":true}
```

The instance's active connection count is decremented by 1.
