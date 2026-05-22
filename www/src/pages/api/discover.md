---
layout: "@layouts/DocLayout.astro"
title: GET /{service}
description: Discover and redirect to the least-loaded instance.
---

Discover the least-loaded healthy instance for a service and redirect the client.

## Request

```bash
curl -v http://localhost:8080/ms.auth/login
# HTTP/1.1 302 Found
# Location: http://10.0.0.5:3001/login
```

### Path

| Segment | Description |
|---------|-------------|
| `ms.auth` | Service name to discover |
| `/login` | Path appended to the redirect URL |

## Response

A **302 Found** redirect to the least-loaded instance. The path after the service name is appended to the target URL.

## Selection Algorithm

flux selects the instance with the lowest `active_connections` among healthy instances (those whose TTL hasn't expired).
