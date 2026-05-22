---
layout: "@layouts/DocLayout.astro"
title: POST /unregister
description: Remove a service instance.
---

Remove a service instance from the registry immediately.

## Request

```bash
curl -X POST http://localhost:8080/unregister \
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

The instance is removed from the registry. Future discover requests for this service will not return this instance.
