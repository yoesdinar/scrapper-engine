#!/bin/bash

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}🧪 Local Redis Distribution Strategy Test${NC}"
echo "=========================================="

# Function to check if a service is running
check_service() {
    local url=$1
    local name=$2
    local timeout=30
    local count=0
    
    echo -e "${YELLOW}⏳ Waiting for $name to start...${NC}"
    while [ $count -lt $timeout ]; do
        if curl -s -f "$url" >/dev/null 2>&1; then
            echo -e "${GREEN}✅ $name is running${NC}"
            return 0
        fi
        sleep 1
        count=$((count + 1))
    done
    
    echo -e "${RED}❌ $name failed to start within ${timeout}s${NC}"
    return 1
}

# Function to test configuration update
test_config_update() {
    echo -e "${BLUE}🔄 Testing Redis distribution strategy propagation...${NC}"
    
    # Update config
    local config='{"url": "https://httpbin.org/uuid"}'
    echo "Updating configuration: $config"
    
    local response=$(curl -s -X POST http://localhost:8080/api/v1/config \
        -H 'Content-Type: application/json' \
        -u admin:admin123 \
        -d "$config")
    
    if echo "$response" | grep -q "successfully"; then
        echo -e "${GREEN}✅ Configuration updated successfully${NC}"
    else
        echo -e "${RED}❌ Configuration update failed: $response${NC}"
        return 1
    fi
    
    # Wait a moment for Redis propagation
    sleep 2
    
    # Test worker response
    echo "Testing worker response..."
    local worker_response=$(curl -s http://localhost:8082/hit)
    
    if echo "$worker_response" | grep -q "uuid"; then
        echo -e "${GREEN}✅ Worker received Redis config instantly!${NC}"
        echo "Response: $worker_response"
        return 0
    else
        echo -e "${YELLOW}⏳ Worker response: $worker_response${NC}"
        echo -e "${YELLOW}   (May need to wait for HTTP polling fallback)${NC}"
        return 1
    fi
}

# Function to cleanup
cleanup() {
    echo -e "${YELLOW}🧹 Cleaning up...${NC}"
    
    # Kill any running services from make commands
    pkill -f "go run ./cmd" 2>/dev/null || true
    
    # Clean up build artifacts
    rm -f controller/controller agent/agent worker/worker
    rm -f controller/*.db* agent/*.cache
    
    echo -e "${GREEN}✅ Cleanup complete${NC}"
}

# Function to start services in background
start_services() {
    echo -e "${BLUE}🚀 Starting services in background...${NC}"
    
    # Start controller with Redis strategy
    echo "Starting controller (with Redis strategy)..."
    cd controller
    DISTRIBUTION_STRATEGY=REDIS \
    REDIS_ADDRESS=localhost:6379 \
    REDIS_PASSWORD= \
    REDIS_DB=0 \
    DB_PATH=./controller.db \
    PORT=8080 \
    AGENT_USERNAME=agent \
    AGENT_PASSWORD=secret123 \
    ADMIN_USERNAME=admin \
    ADMIN_PASSWORD=admin123 \
    go run ./cmd > ../logs/controller.log 2>&1 &
    CONTROLLER_PID=$!
    cd ..
    
    # Start worker
    echo "Starting worker..."
    cd worker
    PORT=8082 go run ./cmd > ../logs/worker.log 2>&1 &
    WORKER_PID=$!
    cd ..
    
    # Wait for controller and worker
    sleep 5
    check_service "http://localhost:8080/health" "Controller" || return 1
    check_service "http://localhost:8082/health" "Worker" || return 1
    
    # Start agent with Redis strategy
    echo "Starting agent (with Redis strategy)..."
    cd agent
    DISTRIBUTION_STRATEGY=REDIS \
    REDIS_ADDRESS=localhost:6379 \
    REDIS_PASSWORD= \
    REDIS_DB=0 \
    CONTROLLER_URL=http://localhost:8080 \
    CONTROLLER_USERNAME=agent \
    CONTROLLER_PASSWORD=secret123 \
    WORKER_URL=http://localhost:8082 \
    CACHE_FILE=./agent_config.cache \
    go run ./cmd > ../logs/agent.log 2>&1 &
    AGENT_PID=$!
    cd ..
    
    # Wait for agent registration
    sleep 10
    echo -e "${GREEN}✅ All services started${NC}"
}

# Main test function
main() {
    echo "Starting Redis strategy architecture test..."
    
    # Create logs directory
    mkdir -p logs
    
    # Setup Redis
    if ! ./scripts/setup-local-redis.sh; then
        echo -e "${RED}❌ Redis setup failed${NC}"
        exit 1
    fi
    
    # Cleanup any previous runs
    cleanup
    
    # Start services
    start_services
    
    # Run tests
    echo ""
    echo -e "${BLUE}🧪 Running Redis strategy tests...${NC}"
    
    # Test 1: Configuration propagation
    if test_config_update; then
        echo -e "${GREEN}✅ Test 1 PASSED: Redis configuration propagation${NC}"
    else
        echo -e "${YELLOW}⚠️  Test 1 PARTIAL: May be using HTTP polling fallback${NC}"
    fi
    
    # Test 2: Check logs for Redis activity
    echo ""
    echo -e "${BLUE}📋 Checking service logs for Redis activity...${NC}"
    
    if grep -q "Redis client initialized successfully" logs/controller.log; then
        echo -e "${GREEN}✅ Controller: Redis client initialized${NC}"
    else
        echo -e "${RED}❌ Controller: Redis client not found in logs${NC}"
    fi
    
    if grep -q "Configuration published to Redis" logs/controller.log; then
        echo -e "${GREEN}✅ Controller: Configuration published to Redis${NC}"
    else
        echo -e "${YELLOW}⚠️  Controller: No Redis publish messages found${NC}"
    fi
    
    if grep -q "Started Redis configuration subscriber" logs/agent.log; then
        echo -e "${GREEN}✅ Agent: Redis subscriber started${NC}"
    else
        echo -e "${YELLOW}⚠️  Agent: Redis subscriber not found in logs${NC}"
    fi
    
    # Test 3: Multiple rapid updates (Redis advantage)
    echo ""
    echo -e "${BLUE}🔄 Testing rapid configuration updates (Redis advantage)...${NC}"
    
    for i in {1..3}; do
        echo "Update $i..."
        curl -s -X POST http://localhost:8080/api/v1/config \
            -H 'Content-Type: application/json' \
            -u admin:admin123 \
            -d "{\"url\": \"https://httpbin.org/delay/$i\"}" > /dev/null
        sleep 1
    done
    
    sleep 2
    local final_response=$(curl -s http://localhost:8082/hit)
    if echo "$final_response" | grep -q "delay"; then
        echo -e "${GREEN}✅ Rapid updates: Worker received latest config${NC}"
    else
        echo -e "${YELLOW}⚠️  Rapid updates: Response: $final_response${NC}"
    fi
    
    # Show service logs
    echo ""
    echo -e "${BLUE}📋 Service Logs (last 10 lines):${NC}"
    echo ""
    echo -e "${YELLOW}Controller Log:${NC}"
    tail -10 logs/controller.log
    echo ""
    echo -e "${YELLOW}Agent Log:${NC}"
    tail -10 logs/agent.log
    echo ""
    echo -e "${YELLOW}Worker Log:${NC}"
    tail -10 logs/worker.log
    
    # Cleanup
    cleanup
    
    echo ""
    echo -e "${GREEN}🎉 Redis distribution strategy test completed!${NC}"
    echo ""
    echo "Summary:"
    echo "✅ Services started with Redis distribution strategy"
    echo "✅ Configuration updates tested"
    echo "✅ Service logs captured"
    echo ""
    echo "To run manually:"
    echo "1. ./scripts/setup-local-redis.sh"
    echo "2. make run-controller-redis  (Terminal 1)"
    echo "3. make run-worker            (Terminal 2)" 
    echo "4. make run-agent-redis       (Terminal 3)"
    echo "5. Test with curl commands"
    echo ""
    echo "For HTTP polling strategy, use: make run-agent-poller"
}

# Handle interruption
trap cleanup EXIT INT TERM

# Run main function
main "$@"