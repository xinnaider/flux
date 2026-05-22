---
layout: "@layouts/DocLayout.astro"
title: How It Works
description: Understand the architecture and request flow.
---

flux is a lightweight **service registry** and **load balancer** with two operation modes:
**redirect mode** (default) and **reverse proxy mode** (`PROXY_MODE=true`).

## Redirect Mode (default)

In redirect mode, flux returns HTTP 302 to the best instance. The client connects directly —
no traffic goes through flux.

```
+-------------------+          +-------------------+
|     Client        |          |    Service A      |
|                   |          |    10.0.0.5:3001  |
+--------+----------+          +---------+---------+
         |                               ^
         | 1. GET /ms.auth/login          |
         |                               |
         v                               |
+-------------------+     +--------+     |
|      flux         |---->|  Redis |     |
|     :8080         |     |  :6379 |     |
+-------------------+     +--------+     |
         |                               |
         | 2. 302 -> http://10.0.0.5:3001/login
         |                               |
         +-------------------------------+
         |
         v
+-------------------+
|     Client follows|
|     redirect      |
+-------------------+
```

## Proxy Mode (`PROXY_MODE=true`)

In proxy mode, flux proxies the entire request to the backend and returns the response.
The client never sees internal addresses.

```
+-------------------+
|     Client        |
+--------+----------+
         |
         | GET /ms.auth/login
         v
+-------------------+     +---------+
|      flux         |---->|  Redis  |
|   (proxy mode)    |     |  :6379  |
+--------+----------+     +---------+
         |
         | proxy_pass to best instance
         v
+-------------------+
|    Service A      |
|    10.0.0.5:3001  |
+-------------------+
         |
         | response flows back through flux
         v
+-------------------+
|     Client gets   |
|   response direct |
+-------------------+
```

This mode is ideal behind Nginx (for HTTPS termination) or in environments where
clients cannot follow 302 redirects.

## Request Flow

1. **Register** — Service instances POST to `/register` with their name, host, and port.
2. **Heartbeat** — Instances POST to `/heartbeat` every few seconds with their active connection count.
3. **Route** — Clients GET `/{service}/{path}`. flux looks up the least-loaded instance:
   - **Redirect mode**: returns `302 Location: http://{instance}/{path}`
   - **Proxy mode**: proxies the request to `http://{instance}/{path}` and returns the response
4. **Eviction** — Instances that stop heartbeating are automatically evicted after their TTL expires.

## Selection Algorithm

flux uses **least connections** — it picks the instance with the lowest `active_connections`
value reported via heartbeat. Between heartbeats, flux also increments a counter on the
chosen instance as a fallback.

## Data Model

| Concept | Representation |
|---------|---------------|
| Service | A named group (e.g. `ms.auth`) |
| Instance | `host:port` within a service |
| Load | Active connection count reported via heartbeat |
| TTL | Seconds since last heartbeat before eviction |
