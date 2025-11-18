#!/bin/bash

# End-to-end test script for the distributed configuration management system

set -e

BASE_URL_CONTROLLER="http://localhost:8080"
BASE_URL_WORKER="http://localhost:8082"
ADMIN_CREDS="admin:admin123"

echo "======================================"
echo "E2E Test: Configuration Management"
echo "======================================"
echo ""

echo "Prerequisites: Ensure all services are running:"
echo "  Terminal 1: cd controller && ./controller"
echo "  Terminal 2: cd worker && ./worker"
echo "  Terminal 3: cd agent && ./agent"
echo ""
read -p "Press Enter when all services are running..."

echo ""
echo "1. Checking Controller health..."
curl -s ${BASE_URL_CONTROLLER}/health | jq .
echo ""

echo "2. Checking Worker health..."
curl -s ${BASE_URL_WORKER}/health | jq .
echo ""

URLS=(
  "https://ip.me"
  "https://api.github.com"
  "https://jsonplaceholder.typicode.com/todos/1"
  "https://dog.ceo/api/breeds/image/random"
  "https://httpbin.org/get"
  "https://shopee.co.id/Baju-Gamis-Wanita-Muslim-Terbaru-Sandira-Dress-cantik-Murah-kekinian-GMS01-i.464688262.9270692704"
)

for idx in "${!URLS[@]}"; do
  url="${URLS[$idx]}"
  echo "Test $((idx+3)). Updating configuration to $url..."
  curl -s -X POST ${BASE_URL_CONTROLLER}/api/v1/config \
    -u ${ADMIN_CREDS} \
    -H "Content-Type: application/json" \
    -d "{\"url\":\"$url\"}" | jq .
  echo ""

  echo "Waiting 35 seconds for agent to poll and update worker..."
  for i in {35..1}; do
    echo -ne "   Countdown: $i seconds\r"
    sleep 1
  done
  echo ""

  echo "Testing worker /hit endpoint (should proxy $url response)..."
  # Use jq for JSON, else print raw
  resp=$(curl -s ${BASE_URL_WORKER}/hit)
  if [[ "$resp" =~ ^\{ ]]; then
    echo "$resp" | jq . | head -20
  else
    echo "$resp"
  fi
  echo ""
done

echo "Listing all registered agents..."
curl -s ${BASE_URL_CONTROLLER}/api/v1/agents -u ${ADMIN_CREDS} | jq .
echo ""

echo "======================================"
echo "E2E Test Complete!"
echo "======================================"
echo ""
echo "View Swagger documentation at:"
echo "  Controller: ${BASE_URL_CONTROLLER}/swagger/index.html"
echo "  Worker: ${BASE_URL_WORKER}/swagger/index.html"
