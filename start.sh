#!/usr/bin/env bash
set -euo pipefail

PORT="${1:?Usage: ./start.sh <port>}"
export PORT

echo "Starting Stock Market Simulation on port $PORT..."

docker compose build --parallel
docker compose up -d

echo ""
echo "Service running at http://localhost:$PORT"
echo "  Health: http://localhost:$PORT/healthz"
echo "  Stocks: http://localhost:$PORT/stocks"
echo "  Log:    http://localhost:$PORT/log"
echo ""
echo "To stop: docker compose down"
