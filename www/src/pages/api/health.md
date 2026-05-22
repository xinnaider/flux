---
layout: "@layouts/DocLayout.astro"
title: GET /health
description: Check flux registry health.
---

Simple health check endpoint.

## Request

```bash
curl http://localhost:8080/health
```

## Response

```json
{"status":"ok"}
```

## Used By

- Docker health checks
- Load balancer health probes
- Monitoring systems
