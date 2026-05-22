param(
    [int]$Requests = 500,
    [int]$Concurrency = 50,
    [switch]$NoBuild
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSCommandPath

Write-Host "===========================================" -ForegroundColor Cyan
Write-Host "  FLUX LOAD TEST" -ForegroundColor Cyan
Write-Host "===========================================" -ForegroundColor Cyan
Write-Host ""

# clean up any previous run
Write-Host "-> Cleaning previous containers..." -ForegroundColor Yellow
docker compose -f "$root\docker-compose.test.yml" down --remove-orphans 2>$null

# build images
if (-not $NoBuild) {
    Write-Host "-> Building images..." -ForegroundColor Yellow
    docker compose -f "$root\docker-compose.test.yml" build
    if ($LASTEXITCODE -ne 0) { throw "build failed" }
}

# set env vars for loadtester
$env:NUM_REQUESTS = "$Requests"
$env:CONCURRENCY = "$Concurrency"

# run
Write-Host "-> Starting test stack..." -ForegroundColor Yellow
docker compose -f "$root\docker-compose.test.yml" up --abort-on-container-exit --exit-code-from loadtester
$exitCode = $LASTEXITCODE

# capture logs
Write-Host ""
Write-Host "-> Load tester exit code: $exitCode" -ForegroundColor $(if ($exitCode -eq 0) { "Green" } else { "Red" })

# show any error logs from backends
Write-Host "-> Checking backend logs (errors only)..." -ForegroundColor Yellow
docker compose -f "$root\docker-compose.test.yml" logs --tail=5 fake-app-1 fake-app-2 fake-app-3 2>$null

# clean up
Write-Host "-> Cleaning up..." -ForegroundColor Yellow
docker compose -f "$root\docker-compose.test.yml" down --remove-orphans

if ($exitCode -eq 0) {
    Write-Host "`nSUCCESS: Load test passed!" -ForegroundColor Green
} else {
    Write-Host "`nFAILURE: Load test reported errors." -ForegroundColor Red
}
exit $exitCode
