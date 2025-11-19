#!/bin/bash

set -e

CONTROLLER_URL="http://localhost:8080"
ADMIN_USER="admin"
ADMIN_PASS="admin123"

WORKER1_URL="http://localhost:8082"
WORKER2_URL="http://localhost:8083"
WORKER3_URL="http://localhost:8084"

echo "=== Docker Multi-Agent/Worker Test ==="
echo ""
echo "This script tests the distributed configuration management system with:"
echo "  - 1 Controller (port 8080)"
echo "  - 3 Workers (ports 8082, 8083, 8084)"
echo "  - 3 Agents (one connected to each worker)"
echo "  - 3 Users hitting each worker simultaneously"
echo ""

echo "Step 1: Starting all services..."
docker compose -f docker-compose.test.yml up -d

echo ""
echo "Step 2: Waiting for services to be healthy..."
sleep 10

check_health() {
    local url=$1
    local name=$2
    local max_attempts=30
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if curl -f -s "${url}/health" > /dev/null 2>&1; then
            echo "✓ ${name} is healthy"
            return 0
        fi
        echo "  Waiting for ${name}... (attempt ${attempt}/${max_attempts})"
        sleep 2
        attempt=$((attempt + 1))
    done
    
    echo "✗ ${name} failed to become healthy"
    return 1
}

check_health "$CONTROLLER_URL" "Controller"
check_health "$WORKER1_URL" "Worker 1"
check_health "$WORKER2_URL" "Worker 2"
check_health "$WORKER3_URL" "Worker 3"

echo ""
echo "Step 3: Setting configuration via controller..."
curl -s -X POST "${CONTROLLER_URL}/api/v1/config" \
    -u "${ADMIN_USER}:${ADMIN_PASS}" \
    -H "Content-Type: application/json" \
    -d '{"url": "https://httpbin.org/uuid"}' | jq '.'

echo ""
echo "Step 4: Waiting for agents to propagate config to workers..."
sleep 5

echo ""
echo "Step 5: Simulating 3 users hitting each worker (9 total requests)..."
echo ""

simulate_users() {
    local worker_url=$1
    local worker_name=$2
    local user_id=$3
    
    echo "  [User ${user_id} -> ${worker_name}] Starting request..."
    response=$(curl -s -w "\n%{http_code}" "${worker_url}/hit")
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" = "200" ]; then
        echo "  [User ${user_id} -> ${worker_name}] ✓ Success (HTTP ${http_code})"
        echo "  [User ${user_id} -> ${worker_name}] Response: ${body}"
    else
        echo "  [User ${user_id} -> ${worker_name}] ✗ Failed (HTTP ${http_code})"
        echo "  [User ${user_id} -> ${worker_name}] Response: ${body}"
    fi
}

echo "--- Hitting Worker 1 (port 8082) ---"
for i in 1 2 3; do
    simulate_users "$WORKER1_URL" "Worker1" "$i" &
done
wait

echo ""
echo "--- Hitting Worker 2 (port 8083) ---"
for i in 4 5 6; do
    simulate_users "$WORKER2_URL" "Worker2" "$i" &
done
wait

echo ""
echo "--- Hitting Worker 3 (port 8084) ---"
for i in 7 8 9; do
    simulate_users "$WORKER3_URL" "Worker3" "$i" &
done
wait

echo ""
echo "Step 6: Getting agent status from controller..."
curl -s -X GET "${CONTROLLER_URL}/api/v1/agents" \
    -u "${ADMIN_USER}:${ADMIN_PASS}" | jq '.'

echo ""
echo "Step 7: Testing configuration update..."
curl -s -X POST "${CONTROLLER_URL}/api/v1/config" \
    -u "${ADMIN_USER}:${ADMIN_PASS}" \
    -H "Content-Type: application/json" \
    -d '{"url": "https://api.ipify.org?format=json"}' | jq '.'

echo ""
echo "Waiting for config propagation..."
sleep 5

echo ""
echo "Step 8: Testing workers with new configuration..."
echo ""

echo "--- Testing Worker 1 with new config ---"
curl -s "${WORKER1_URL}/hit"
echo ""

echo "--- Testing Worker 2 with new config ---"
curl -s "${WORKER2_URL}/hit"
echo ""

echo "--- Testing Worker 3 with new config ---"
curl -s "${WORKER3_URL}/hit"
echo ""

echo ""
echo "=== Test Complete ==="
echo ""
echo "To view logs:"
echo "  docker compose -f docker-compose.test.yml logs -f [controller|agent1|agent2|agent3|worker1|worker2|worker3]"
echo ""
echo "To stop services:"
echo "  docker compose -f docker-compose.test.yml down"
echo ""
echo "To stop and remove volumes:"
echo "  docker compose -f docker-compose.test.yml down -v"
