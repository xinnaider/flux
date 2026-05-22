#!/bin/sh
set -e

echo ""
echo "=== 1/4 Lint ==="
golangci-lint run --timeout=5m
echo "Lint passed."

echo ""
echo "=== 2/4 Go Tests ==="
go test -v -race -count=1 ./...
echo "Tests passed."

echo ""
echo "=== 3/4 Go Build ==="
go build -o /dev/null ./cmd/server
echo "Build passed."

echo ""
echo "=== 4/4 Astro Build ==="
cd www
npm ci
npm run build
cd ..
echo "Astro build passed."

echo ""
echo "=============================="
echo " All checks passed!          "
echo "=============================="
