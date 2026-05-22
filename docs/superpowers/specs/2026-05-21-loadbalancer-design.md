# Load Balancer por Redirect 302 com Redis

**Data:** 2026-05-21
**Status:** Aprovado

## Visão Geral

Service Registry + Load Balancer para médio tráfego. Serviços se registram com nome lógico (ex: `ms.autenticacao`), enviam heartbeats periódicos com a carga real, e o frontend é redirecionado via HTTP 302 para instância com menos conexões ativas.

## Arquitetura

```
                    ┌──────────────┐
                    │  Load Balancer│  (HAProxy / nginx / DNS RR)
                    │  :8080        │
                    └──────┬───────┘
                   ┌───────┼───────┐
                   │       │       │
              ┌────▼──┐ ┌──▼──┐ ┌──▼────┐
              │Reg 1  │ │Reg 2│ │Reg N  │  ← Go binary, stateless
              └───┬───┘ └──┬──┘ └───┬───┘
                  │        │        │
                  └────────┼────────┘
                           │
                     ┌─────▼──────┐
                     │   Redis     │  ← Fonte única de verdade
                     └────────────┘
```

**Registry é stateless** — todo estado fica no Redis. Múltiplas instâncias do registry podem rodar atrás de um LB simples (HAProxy, nginx, ou até DNS round-robin).

## Fluxo

```
1. POST /register        ──► Redis: cria HASH + adiciona ao SET
2. POST /heartbeat       ──► Redis: atualiza TTL + active_connections
3. GET /{service}/*      ──► Redis: busca HASH com menor connections → 302
4. POST /release         ──► Redis: decrementa connections (opcional)
5. Goroutine cleanup     ──► Redis: remove instâncias expiradas
```

## Estratégia de Balanceamento

**Híbrida (Opção C):**

1. **Instância reporta carga real** no heartbeat:
   ```json
   POST /heartbeat
   { "name": "ms.autenticacao", "instance_id": "uuid", "active_connections": 42 }
   ```
   O registry armazena `active_connections` no HASH do Redis. Esse valor é a verdade — a instância sabe exatamente quantas conexões está servindo.

2. **Registry incrementa contador ao redirecionar** — como fallback entre heartbeats:
   - Ao fazer o redirect, registry incrementa `connections` no Redis
   - Isso evita que múltiplos redirects caiam na mesma instância entre heartbeats

3. **Próximo heartbeat sobrescreve** com o valor real da instância:
   - A instância envia `active_connections` real (ex: o servidor web já conta)
   - Registry grava por cima do valor incrementado

**Resultado:** precisão da instância + correção entre heartbeats. Simples e eficaz.

## Estrutura no Redis

| Chave | Tipo | Descrição | TTL |
|-------|------|-----------|-----|
| `service:{name}:instances` | SET | Instance IDs do serviço | - |
| `instance:{name}:{id}` | HASH | `{host, port, health_url, connections}` | 15s (renovado no heartbeat) |

### Operações

**Registro:**
- `HMSET instance:ms.auth:uuid host 10.0.0.5 port 3001 health_url /health connections 0`
- `EXPIRE instance:ms.auth:uuid 15`
- `SADD service:ms.auth:instances uuid`

**Heartbeat (agora com active_connections):**
- `HMSET instance:ms.auth:uuid connections {active_connections}`
- `EXPIRE instance:ms.auth:uuid 15`

**Redirect (least-connections via SMEMBERS + menor score):**
- `SMEMBERS service:ms.auth:instances` → lista de instance_ids
- `HMGET instance:ms.auth:A connections, instance:ms.auth:B connections...`
- → escolhe instância com **menor connections**
- `HINCRBY instance:ms.auth:{id} connections 1` → incrementa contador

**Release (opcional — instância libera conexão ao fechar):**
- `POST /release {name, instance_id}`
- `HINCRBY instance:ms.auth:{id} connections -1`

**Cleanup (goroutine a cada 5s):**
- `SCAN 0 MATCH service:*:instances` → descobre todos os serviços
- `SMEMBERS service:{name}:instances` → itera instâncias
- `EXISTS instance:{name}:{id}` = 0 → instância expirou
- `SREM service:{name}:instances {id}` → remove do set

## API

| Método | Rota | Body | Resposta |
|--------|------|------|----------|
| `POST` | `/register` | `{name, host, port, health_url}` | `201 {instance_id, ttl}` |
| `POST` | `/unregister` | `{name, instance_id}` | `200` |
| `POST` | `/heartbeat` | `{name, instance_id, active_connections}` | `200 {ttl}` |
| `POST` | `/release` | `{name, instance_id}` | `200` |
| `GET` | `/health` | - | `200 {ok}` |
| `GET` | `/{name}/*` | - | `302 Location: http://host:port/path` |

## Capacidade

| Métrica | Estimativa |
|---------|-----------|
| Redirects/s por instância | ~50k-100k |
| Latência p95 | < 2ms (incluindo Redis) |
| Instâncias do registry | N (horizontal, stateless) |
| Redis | 1 instância (ou cluster para HA) |

Dependência externa: apenas **Redis**. Sem banco SQL, sem fila, sem cache extra.

## Estrutura do Projeto Go

```
.
├── cmd/
│   └── server/
│       └── main.go              # Entry point, wiring
├── internal/
│   ├── api/
│   │   └── handler.go           # HTTP handlers
│   ├── registry/
│   │   ├── registry.go          # Interface + Redis implementation
│   │   └── registry_test.go
│   ├── balancer/
│   │   ├── balancer.go          # Least-connections: SMEMBERS + HGET + HINCRBY
│   │   └── balancer_test.go
│   ├── health/
│   │   └── checker.go           # Cleanup goroutine
│   └── config/
│       └── config.go            # Env-based config
├── go.mod
├── Makefile
└── Dockerfile
```

## Dependências

- `github.com/redis/go-redis/v9` — cliente Redis
- `github.com/kelseyhightower/envconfig` — config por env vars (opcional, pode ser std lib)
- `net/http` — std lib (sem framework externo)
