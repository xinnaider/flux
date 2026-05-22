#!/usr/bin/env bash
set -euo pipefail

REQUESTS="${1:-500}"
CONCURRENCY="${2:-50}"
DIR="$(cd "$(dirname "$0")" && pwd)"

echo "==========================================="
echo "  FLUX LOAD TEST"
echo "==========================================="
echo ""

echo "-> Cleaning previous containers..."
docker compose -f "$DIR/docker-compose.test.yml" down --remove-orphans 2>/dev/null || true

echo "-> Building images..."
docker compose -f "$DIR/docker-compose.test.yml" build

echo "-> Running load test (${REQUESTS} requests, ${CONCURRENCY} concurrent)..."
NUM_REQUESTS="$REQUESTS" CONCURRENCY="$CONCURRENCY" \
  docker compose -f "$DIR/docker-compose.test.yml" up \
    --abort-on-container-exit --exit-code-from loadtester

EXIT_CODE=$?

echo ""
if [ "$EXIT_CODE" -eq 0 ]; then
  echo "SUCCESS: Load test passed!"
else
  echo "FAILURE: Load test reported errors (exit=$EXIT_CODE)"
fi

echo "-> Cleaning up..."
docker compose -f "$DIR/docker-compose.test.yml" down --remove-orphans

exit $EXIT_CODE
