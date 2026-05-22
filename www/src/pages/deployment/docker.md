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

## GHCR Pull (private image)

The flux image is hosted on GitHub Container Registry and requires authentication:

```bash
echo $PAT | docker login ghcr.io -u xinnaider --password-stdin
docker pull ghcr.io/xinnaider/flux
docker run -e REDIS_ADDR=host.docker.internal:6379 -p 8080:8080 ghcr.io/xinnaider/flux
```

## Docker Build

Build the image locally from source:

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
