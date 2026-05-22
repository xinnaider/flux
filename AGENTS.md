# flux — Agent Context

Agentes: leia `CLAUDE.md` na raiz do projeto primeiro.

## Projeto

**flux** — Service Registry + Load Balancer em Go + Redis.

Dois modos de operacao:

| Modo | Descricao |
|------|-----------|
| **Redirect** (`PROXY_MODE=false`, default) | 302 redirect para o backend menos carregado. Cliente vai direto. Ideal para redes internas. |
| **Proxy** (`PROXY_MODE=true`) | Reverse proxy com `httputil.ReverseProxy`. Fluxo: cliente → flux → backend → flux → cliente. Ideal atras de Nginx com HTTPS. |

## Stack

| Camada | Tecnologia |
|--------|-----------|
| Runtime | Go 1.22+ |
| Estado | Redis 7 (hash + set, TTL-based) |
| Proxy | `net/http/httputil.ReverseProxy` (stdlib) |
| Frontend | Astro 6 (www/) |
| Infra | Docker Compose, Nginx |

## Estrutura

```
flux/
├── cmd/server/main.go          ← Entrypoint
├── internal/
│   ├── api/
│   │   ├── handler.go          ← HTTP routes + redirect ou proxy
│   │   └── handler_test.go
│   ├── balancer/
│   │   └── proxy.go            ← Reverse proxy (Director + Transport pool)
│   ├── config/
│   │   └── config.go           ← Env vars
│   ├── health/
│   │   └── checker.go          ← Cleanup goroutine
│   └── registry/
│       ├── registry.go         ← RedisRegistry (Register, Heartbeat, GetInstance)
│       └── registry_test.go
├── test/                       ← Load test framework
│   ├── docker-compose.test.yml
│   ├── fake-backend/           ← App fake que registra + heartbeat
│   └── loadtester/             ← Testador concorrente c/ report
├── www/                        ← Landing page (Astro)
└── README.md
```

## Regras para Codigo

### Proxy Mode (`internal/balancer/proxy.go`)

- Usa `httputil.ReverseProxy` com `Director` setado (Go 1.22+ exige)
- `Transport` compartilhado (connection pool), `ReverseProxy` criado por request
- Headers forwarded: `X-Forwarded-For`, `X-Forwarded-Host`, `X-Forwarded-Proto`, `X-Real-IP`
- Instancia via `registry.GetInstance()` — least-connections
- 503 se service name nao encontrado, 502 se backend caiu

### Handler (`internal/api/handler.go`)

- `SetProxy()` ativa proxy mode. Se nil, usa redirect (302)
- Rotas fixas: `/register`, `/unregister`, `/heartbeat`, `/release`, `/health`
- Catch-all `/` decide entre proxy ou redirect

### Config (`internal/config/config.go`)

Env vars cruciais:

| Env | Default | Info |
|-----|---------|------|
| `PROXY_MODE` | `false` | `true` ativa reverse proxy |
| `PROXY_IDLE_CONNS` | `100` | Pool de conexoes idle |
| `PROXY_IDLE_PER_HOST` | `10` | Por backend |
| `HEARTBEAT_TTL` | `15s` | TTL da instancia no Redis |
| `CLEANUP_INTERVAL` | `5s` | Limpeza de instancias expiradas |

### Testes

- Requer Redis em `localhost:6379` (ou `REDIS_TEST_ADDR`)
- Usa DB separado por pacote (1 = registry, 2 = api, 3 = proxy)
- `go test -v -race ./...` antes de commit
- Testes skipam se Redis nao disponivel

### Load Test (`test/`)

```powershell
cd test
$env:NUM_REQUESTS="5000"; $env:CONCURRENCY="300"
docker compose -f docker-compose.test.yml up --abort-on-container-exit --exit-code-from loadtester-1
```

4 loadtesters paralelos: `loadtester-1` a `loadtester-4`.

## Producao com Nginx

```
[Cliente] --HTTPS--> [Nginx :443] --HTTP--> [flux :8080 (proxy)] --HTTP--> [Backend]
```

Nginx faz SSL termination, flux faz service discovery + least-connections.

## Performance Esperada

| Modo | Throughput (3 backends) |
|------|-----------------------|
| Redirect | ~10k req/s |
| Proxy | ~7.5k req/s |

Gargalo tipico sao os backends, nao flux. Escala horizontal atras do Nginx.
