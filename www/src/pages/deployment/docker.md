---
layout: "@layouts/DocLayout.astro"
title: Docker
description: Run flux with Docker.
---

## Docker Compose (Recommended)

The project includes a `docker-compose.yml` that starts flux + Redis:

```bash
docker compose up -d
```

This starts:
- **flux** on port `8080`
- **Redis 7** on port `6379`

## Docker Build

Build the image locally:

```bash
docker build -t flux .
docker run -p 8080:8080 --network host flux
```

## Docker Compose Customization

```yaml
services:
  flux:
    build: .
    ports:
      - "8080:8080"
    environment:
      - REDIS_ADDR=redis:6379
      - HEARTBEAT_TTL=30s
    depends_on:
      - redis

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis-data:/data

volumes:
  redis-data:
```
