---
layout: "@layouts/DocLayout.astro"
title: GET /{service}
description: Discover and route to the least-loaded instance.
---

## Request

Route a client to the least-loaded healthy instance for a service.

### Redirect mode (default)

```bash
curl -v http://localhost:8080/ms.auth/login
# HTTP/1.1 302 Found
# Location: http://10.0.0.5:3001/login
```

### Proxy mode (`PROXY_MODE=true`)

```bash
curl -v http://localhost:8080/ms.auth/login
# HTTP/1.1 200 OK
# <response from backend>
```

### Path

| Segment | Description |
|---------|-------------|
| `ms.auth` | Service name to discover |
| `/login` | Path forwarded to the backend |

## Response

| Mode | Status | Behaviour |
|------|--------|-----------|
| Redirect (default) | `302 Found` | Returns `Location` header pointing to the best instance |
| Proxy (`PROXY_MODE=true`) | `200 OK` (from backend) | Proxies the request and returns the backend response directly |

In proxy mode, flux adds `X-Forwarded-For`, `X-Forwarded-Host`, `X-Forwarded-Proto`, and `X-Real-IP` headers to the proxied request.

## Selection Algorithm

flux selects the instance with the lowest `active_connections` among healthy instances (those whose TTL hasn't expired).
