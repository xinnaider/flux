# flux — Claude Project Config

Leia `AGENTS.md` para contexto completo do projeto.

## Regras Cruciais

1. **Testes antes de commit**: sempre rodar `go test -v -race ./...` (precisa Redis em localhost:6379) antes de qualquer commit
2. **Backward compat**: `PROXY_MODE=false` (redirect) é o default. Nao quebrar esse comportamento
3. **Nao hardcodar URLs**: flux expoe `http://` — HTTPS é responsabilidade do Nginx na frente
4. **Versao Go**: 1.22+. `httputil.ReverseProxy` exige `Director` explicito (nao deixar nil)
5. **Nova feature = teste**: toda nova funcionalidade precisa de teste no `internal/` correspondente

## Comandos Rapidos

```powershell
# Testes (requer Redis)
go test -v -race ./...

# Build
go build -o bin/flux ./cmd/server

# Run local (precisa Redis)
go run ./cmd/server

# Load test
cd test
$env:NUM_REQUESTS="5000"; $env:CONCURRENCY="300"
docker compose -f docker-compose.test.yml up --abort-on-container-exit --exit-code-from loadtester-1
```
