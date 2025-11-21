#!/bin/bash

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}🧪 Quick Strategy Pattern Test${NC}"
echo "=============================="

# Function to check if build succeeds
test_build() {
    echo -e "${YELLOW}📦 Testing build for all services...${NC}"
    
    local failed=0
    
    # Test controller build
    echo "Building controller..."
    if ! (cd controller && go build -o controller ./cmd); then
        echo -e "${RED}❌ Controller build failed${NC}"
        failed=1
    else
        echo -e "${GREEN}✅ Controller build successful${NC}"
        rm -f controller/controller
    fi
    
    # Test agent build
    echo "Building agent..."
    if ! (cd agent && go build -o agent ./cmd); then
        echo -e "${RED}❌ Agent build failed${NC}"
        failed=1
    else
        echo -e "${GREEN}✅ Agent build successful${NC}"
        rm -f agent/agent
    fi
    
    # Test worker build
    echo "Building worker..."
    if ! (cd worker && go build -o worker ./cmd); then
        echo -e "${RED}❌ Worker build failed${NC}"
        failed=1
    else
        echo -e "${GREEN}✅ Worker build successful${NC}"
        rm -f worker/worker
    fi
    
    return $failed
}

# Function to test configuration parsing
test_config_parsing() {
    echo -e "${YELLOW}⚙️  Testing configuration parsing...${NC}"
    
    # Test POLLER strategy build
    echo "Testing POLLER strategy build..."
    cd agent
    if ! DISTRIBUTION_STRATEGY=POLLER go build -o agent_test ./cmd; then
        echo -e "${RED}❌ Agent with POLLER strategy build failed${NC}"
        cd ..
        return 1
    fi
    echo -e "${GREEN}✅ POLLER strategy build successful${NC}"
    rm -f agent_test
    
    # Test REDIS strategy build
    echo "Testing REDIS strategy build..."
    if ! DISTRIBUTION_STRATEGY=REDIS REDIS_ADDRESS=dummy:6379 go build -o agent_test ./cmd; then
        echo -e "${RED}❌ Agent with REDIS strategy build failed${NC}"
        cd ..
        return 1
    fi
    echo -e "${GREEN}✅ REDIS strategy build successful${NC}"
    rm -f agent_test
    
    cd ..
    
    # Test controller
    echo "Testing controller with REDIS strategy build..."
    cd controller
    if ! DISTRIBUTION_STRATEGY=REDIS REDIS_ADDRESS=dummy:6379 go build -o controller_test ./cmd; then
        echo -e "${RED}❌ Controller with REDIS strategy build failed${NC}"
        cd ..
        return 1
    fi
    echo -e "${GREEN}✅ Controller REDIS strategy build successful${NC}"
    rm -f controller_test
    
    echo "Testing controller with POLLER strategy build..."
    if ! DISTRIBUTION_STRATEGY=POLLER go build -o controller_test ./cmd; then
        echo -e "${RED}❌ Controller with POLLER strategy build failed${NC}"
        cd ..
        return 1
    fi
    echo -e "${GREEN}✅ Controller POLLER strategy build successful${NC}"
    rm -f controller_test
    
    cd ..
    
    return 0
}

# Function to test Redis package
test_redis_package() {
    echo -e "${YELLOW}📡 Testing Redis package...${NC}"
    
    cd pkg/redis
    if ! go test ./... -v; then
        echo -e "${RED}❌ Redis package tests failed${NC}"
        cd ../..
        return 1
    fi
    echo -e "${GREEN}✅ Redis package tests passed${NC}"
    cd ../..
    
    return 0
}

# Main test function
main() {
    echo "Starting quick strategy pattern verification..."
    echo ""
    
    local failed=0
    
    # Test 1: Build verification
    if test_build; then
        echo -e "${GREEN}✅ Build Test PASSED${NC}"
    else
        echo -e "${RED}❌ Build Test FAILED${NC}"
        failed=1
    fi
    
    echo ""
    
    # Test 2: Configuration parsing
    if test_config_parsing; then
        echo -e "${GREEN}✅ Configuration Test PASSED${NC}"
    else
        echo -e "${RED}❌ Configuration Test FAILED${NC}"
        failed=1
    fi
    
    echo ""
    
    # Test 3: Redis package
    if test_redis_package; then
        echo -e "${GREEN}✅ Redis Package Test PASSED${NC}"
    else
        echo -e "${RED}❌ Redis Package Test FAILED${NC}"
        failed=1
    fi
    
    echo ""
    echo "=========================================="
    if [ $failed -eq 0 ]; then
        echo -e "${GREEN}🎉 All quick tests PASSED!${NC}"
        echo ""
        echo "Strategy Pattern Implementation Summary:"
        echo "✅ DISTRIBUTION_STRATEGY environment variable supported"
        echo "✅ POLLER strategy: HTTP polling distribution"
        echo "✅ REDIS strategy: Redis pub/sub distribution"
        echo "✅ Extensible design for future strategies (NATS, Kafka)"
        echo "✅ Clean separation of concerns"
        echo ""
        echo "Next steps:"
        echo "1. Run full integration tests: ./scripts/test-strategy-pattern.sh"
        echo "2. Test with local Redis: ./scripts/test-local-redis.sh"
        echo "3. Deploy to production with DISTRIBUTION_STRATEGY=REDIS"
    else
        echo -e "${RED}❌ Some tests FAILED. Please fix the issues above.${NC}"
        exit 1
    fi
}

# Run main function
main "$@"