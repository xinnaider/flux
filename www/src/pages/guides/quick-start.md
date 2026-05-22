---
layout: "@layouts/DocLayout.astro"
title: Quick Start
description: Get flux running in 30 seconds.
---

Start flux and Redis, register a service, discover it.

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/)
- [Go](https://go.dev/dl/) (for local development)

## Use the Public Image (no clone)

```bash
# Start Redis
docker run -d --name redis redis:7-alpine

# Run flux (public image, no login needed)
docker run -d --name flux --link redis -e REDIS_ADDR=redis:6379 -p 8080:8080 ghcr.io/xinnaider/flux

# Check it's alive
curl http://localhost:8080/health
```

Then jump to [Register a Service](#register-a-service).

## Clone the Repository

```bash
git clone https://github.com/xinnaider/flux
cd flux
```

## Start with Docker Compose

```bash
docker compose up -d
```

Check it's alive:

```bash
curl http://localhost:8080/health
# {"status":"ok"}
```

## Register a Service

```bash
curl -X POST http://localhost:8080/register \
  -H "Content-Type: application/json" \
  -d '{"name":"ms.auth","host":"10.0.0.5","port":3001,"health_url":"/health"}'

# {"instance_id":"10.0.0.5:3001","ttl_seconds":15}
```

## Send a Heartbeat

```bash
curl -X POST http://localhost:8080/heartbeat \
  -H "Content-Type: application/json" \
  -d '{"name":"ms.auth","instance_id":"10.0.0.5:3001","active_connections":3}'

# {"ok":true,"ttl_seconds":15}
```

## Discover & Redirect

```bash
curl -v http://localhost:8080/ms.auth/login
# HTTP/1.1 302 Found
# Location: http://10.0.0.5:3001/login
```

## Run Locally (without Docker)

```bash
# Requires Redis on localhost:6379
go run ./cmd/server

# Or build first
go build -o bin/flux ./cmd/server
./bin/flux
```
