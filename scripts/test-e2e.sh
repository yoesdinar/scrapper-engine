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

echo "3. Updating configuration to https://ip.me..."
curl -s -X POST ${BASE_URL_CONTROLLER}/api/v1/config \
  -u ${ADMIN_CREDS} \
  -H "Content-Type: application/json" \
  -d '{"url":"https://ip.me"}' | jq .
echo ""

echo "4. Waiting 35 seconds for agent to poll and update worker..."
for i in {35..1}; do
  echo -ne "   Countdown: $i seconds\r"
  sleep 1
done
echo ""

echo "5. Testing worker /hit endpoint (should return your IP)..."
curl -s ${BASE_URL_WORKER}/hit
echo ""
echo ""

echo "6. Changing configuration to https://api.github.com..."
curl -s -X POST ${BASE_URL_CONTROLLER}/api/v1/config \
  -u ${ADMIN_CREDS} \
  -H "Content-Type: application/json" \
  -d '{"url":"https://api.github.com"}' | jq .
echo ""

echo "7. Waiting 35 seconds for agent to poll and update worker..."
for i in {35..1}; do
  echo -ne "   Countdown: $i seconds\r"
  sleep 1
done
echo ""

echo "8. Testing worker /hit endpoint again (should return GitHub API response)..."
curl -s ${BASE_URL_WORKER}/hit | jq . | head -20
echo ""

echo "9. Listing all registered agents..."
curl -s ${BASE_URL_CONTROLLER}/api/v1/agents -u ${ADMIN_CREDS} | jq .
echo ""

echo "======================================"
echo "E2E Test Complete!"
echo "======================================"
echo ""
echo "View Swagger documentation at:"
echo "  Controller: ${BASE_URL_CONTROLLER}/swagger/index.html"
echo "  Worker: ${BASE_URL_WORKER}/swagger/index.html"
