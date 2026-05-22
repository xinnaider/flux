---
layout: "@layouts/DocLayout.astro"
title: How It Works
description: Understand the architecture and request flow.
---

flux is a lightweight **service registry** and **HTTP redirect load balancer**. It stores ephemeral service instance data in Redis and routes client traffic via HTTP 302 redirects.

## Architecture

```
+-------------------+
|     Client        |
+--------+----------+
         |
         | GET /ms.auth/login
         v
+-------------------+     +---------+
|      flux         |---->|  Redis  |
|     :8080         |     |  :6379  |
+--------+----------+     +---------+
         |
         | 302 -> http://10.0.0.5:3001/login
         v
+-------------------+
|    ms.auth        |
|    10.0.0.5       |
|    :3001           |
+-------------------+
```

## Request Flow

1. **Register** — Service instances POST to `/register` with their name, host, and port.
2. **Heartbeat** — Instances POST to `/heartbeat` every few seconds with their active connection count.
3. **Discover** — Clients GET `/{service}/{path}`. flux looks up the least-loaded instance and returns a **302 redirect** to `http://{instance}/{path}`.
4. **Eviction** — Instances that stop heartbeating are automatically evicted after their TTL expires.

## Why Redirect?

- **No proxy** — traffic hits the target directly. No bandwidth costs, no proxy bottlenecks.
- **No sticky sessions** — the registry is stateless. Add more flux instances behind a round-robin.
- **Client-resilient** — clients follow the redirect. If an instance dies, the next request gets a healthy one.

## Data Model

| Concept | Representation |
|---------|---------------|
| Service | A named group (e.g. `ms.auth`) |
| Instance | `host:port` within a service |
| Load | Active connection count reported via heartbeat |
| TTL | Seconds since last heartbeat before eviction |
