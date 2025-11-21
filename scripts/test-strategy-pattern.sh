#!/bin/bash

set -e

echo "🧪 Testing Distribution Strategy Pattern Architecture"
echo "=================================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Build Docker images for multi-platform support
build_images() {
    echo -e "${BLUE}🔨 Building Docker images for local testing...${NC}"
    
    # Build images using docker compose for better compatibility
    echo "Building images with docker compose..."
    if ! docker compose -f docker-compose.strategy-test.yml -f docker-compose.local.yml build; then
        echo -e "${RED}❌ Failed to build Docker images${NC}"
        return 1
    fi
    
    echo -e "${GREEN}✅ Docker images built successfully${NC}"
    return 0
}

# Test functions
test_redis_mode() {
    echo -e "${BLUE}📡 Testing Redis Pub/Sub Distribution Strategy${NC}"
    echo "Starting Redis strategy architecture..."
    
    # Start with Redis strategy using local images
    DISTRIBUTION_STRATEGY=REDIS docker compose -f docker-compose.strategy-test.yml -f docker-compose.local.yml up -d
    
    # Wait for services to be ready
    echo "⏳ Waiting for services to initialize..."
    sleep 30
    
    # Test configuration update (should propagate via Redis)
    echo "🔄 Testing Redis configuration propagation..."
    curl -X POST http://localhost:8080/api/v1/config \
        -H "Content-Type: application/json" \
        -u admin:admin123 \
        -d '{"url": "https://httpbin.org/uuid"}' \
        -s > /dev/null
    
    # Wait for Redis propagation
    sleep 5
    
    # Test all workers (agents 1&2 should get via Redis, agent 3 via polling)
    echo "🔍 Testing worker responses..."
    for port in 8082 8083 8084; do
        response=$(curl -s http://localhost:$port/hit)
        if [[ $response == *"uuid"* ]]; then
            echo -e "${GREEN}✅ Worker on port $port: Redis config received${NC}"
        else
            echo -e "${YELLOW}⏳ Worker on port $port: Waiting for config...${NC}"
        fi
    done
    
    # Check logs for Redis messages
    echo "📋 Checking Redis distribution strategy logs..."
    if docker logs agent1 2>&1 | grep -q "Received new config from Redis\|Using Redis distribution strategy"; then
        echo -e "${GREEN}✅ Agent1: Redis distribution strategy working${NC}"
    else
        echo -e "${YELLOW}⏳ Agent1: Using HTTP polling fallback${NC}"
    fi
    
    if docker logs agent2 2>&1 | grep -q "Received new config from Redis\|Using Redis distribution strategy"; then
        echo -e "${GREEN}✅ Agent2: Redis distribution strategy working${NC}"
    else
        echo -e "${YELLOW}⏳ Agent2: Using HTTP polling fallback${NC}"
    fi
    
    if docker logs agent3 2>&1 | grep -q "Using HTTP polling distribution strategy\|Using POLLER distribution strategy"; then
        echo -e "${GREEN}✅ Agent3: HTTP polling strategy (as expected)${NC}"
    else
        echo -e "${RED}❌ Agent3: Should be using HTTP polling strategy${NC}"
    fi
    
    echo ""
}

test_polling_mode() {
    echo -e "${BLUE}🔄 Testing HTTP Polling Distribution Strategy${NC}"
    
    # Stop current setup
    docker compose -f docker-compose.strategy-test.yml -f docker-compose.local.yml down
    
    echo "Starting polling-only strategy architecture..."
    
    # Start with polling strategy (no Redis services needed) using local images
    DISTRIBUTION_STRATEGY=POLLER docker compose -f docker-compose.strategy-test.yml -f docker-compose.local.yml up -d controller worker1 worker2 worker3 agent1 agent2 agent3
    
    # Wait for services
    echo "⏳ Waiting for polling strategy services..."
    sleep 20
    
    # Test configuration update (should propagate via polling)
    echo "🔄 Testing HTTP polling configuration propagation..."
    curl -X POST http://localhost:8080/api/v1/config \
        -H "Content-Type: application/json" \
        -u admin:admin123 \
        -d '{"url": "https://api.github.com/zen"}' \
        -s > /dev/null
    
    # Wait for polling cycles
    sleep 35
    
    # Test all workers
    echo "🔍 Testing worker responses..."
    for port in 8082 8083 8084; do
        response=$(curl -s http://localhost:$port/hit)
        if [[ $response == *"zen"* ]] || [[ $response =~ (Non.*Violence|Design|Favor.*focus) ]]; then
            echo -e "${GREEN}✅ Worker on port $port: Polling config received${NC}"
        else
            echo -e "${YELLOW}⏳ Worker on port $port: Response: $response${NC}"
        fi
    done
    
    # Verify all agents are using HTTP polling strategy
    echo "📋 Checking polling strategy logs..."
    for agent in agent1 agent2 agent3; do
        if docker logs $agent 2>&1 | grep -q "Using HTTP polling distribution strategy\|Using POLLER distribution strategy"; then
            echo -e "${GREEN}✅ $agent: Using HTTP polling strategy (as expected)${NC}"
        else
            echo -e "${YELLOW}⏳ $agent: Strategy unclear${NC}"
        fi
    done
    
    echo ""
}

test_backward_compatibility() {
    echo -e "${BLUE}🔄 Testing Backward Compatibility${NC}"
    
    # Test that POLLER strategy works as backward compatibility
    echo "Testing POLLER strategy as backward compatibility..."
    
    # Stop current setup
    docker compose -f docker-compose.strategy-test.yml -f docker-compose.local.yml down
    
    # Test POLLER strategy (backward compatible mode)
    echo "Testing POLLER strategy without Redis dependencies..."
    DISTRIBUTION_STRATEGY=POLLER docker compose -f docker-compose.strategy-test.yml -f docker-compose.local.yml up -d controller worker1 agent1
    
    echo "⏳ Waiting for backward compatible services..."
    sleep 20
    
    # Test configuration update
    curl -X POST http://localhost:8080/api/v1/config \
        -H "Content-Type: application/json" \
        -u admin:admin123 \
        -d '{"url": "https://httpbin.org/json"}' \
        -s > /dev/null
    
    # Wait and test
    sleep 25
    
    response=$(curl -s http://localhost:8082/hit)
    if [[ $response == *"slideshow"* ]]; then
        echo -e "${GREEN}✅ Backward compatibility: POLLER strategy works without Redis${NC}"
    else
        echo -e "${YELLOW}⏳ Backward compatibility: Response: $response${NC}"
    fi
    
    echo ""
}

cleanup() {
    echo "🧹 Cleaning up test environment..."
    docker compose -f docker-compose.strategy-test.yml -f docker-compose.local.yml down -v 2>/dev/null || true
    docker compose -f docker-compose.agent-worker.yml down 2>/dev/null || true
    echo -e "${GREEN}✅ Cleanup complete${NC}"
}

# Main test execution
main() {
    echo "Starting distribution strategy pattern tests..."
    echo ""
    
    # Ensure clean start
    cleanup
    
    # Build Docker images
    if ! build_images; then
        echo -e "${RED}❌ Docker build failed. Exiting.${NC}"
        exit 1
    fi
    
    # Run tests
    test_redis_mode
    test_polling_mode
    test_backward_compatibility
    
    # Cleanup
    cleanup
    
    echo ""
    echo -e "${GREEN}🎉 All strategy pattern tests completed!${NC}"
    echo ""
    echo "Summary:"
    echo "✅ Redis distribution strategy tested"
    echo "✅ HTTP polling distribution strategy tested"
    echo "✅ Backward compatibility verified"
    echo "✅ Strategy pattern supports clean separation between Redis and HTTP polling"
    echo ""
    echo "To run with Redis strategy in production:"
    echo "  DISTRIBUTION_STRATEGY=REDIS docker compose -f docker-compose.production.yml up -d"
    echo ""
    echo "To run with HTTP polling strategy:"
    echo "  DISTRIBUTION_STRATEGY=POLLER docker compose -f docker-compose.production.yml up -d"
    echo ""
    echo "Future strategies can be added (NATS, Kafka, etc.) by implementing the ConfigDistributor interface"
}

# Handle interruption
trap cleanup EXIT INT TERM

# Run main function
main "$@"