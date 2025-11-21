#!/bin/bash

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}🚀 Local Redis Strategy Testing Setup${NC}"
echo "======================================="

# Check if Redis is installed
if ! command -v redis-server >/dev/null 2>&1; then
    echo -e "${RED}❌ Redis not found!${NC}"
    echo ""
    echo "Install Redis:"
    echo "  macOS:   brew install redis"
    echo "  Ubuntu:  sudo apt-get install redis-server"
    echo "  CentOS:  sudo yum install redis"
    exit 1
fi

# Check if Redis is running
if ! redis-cli ping >/dev/null 2>&1; then
    echo -e "${YELLOW}⏳ Starting Redis server...${NC}"
    redis-server --daemonize yes --port 6379
    sleep 2
    
    if redis-cli ping >/dev/null 2>&1; then
        echo -e "${GREEN}✅ Redis started successfully${NC}"
    else
        echo -e "${RED}❌ Failed to start Redis${NC}"
        exit 1
    fi
else
    echo -e "${GREEN}✅ Redis already running${NC}"
fi

echo ""
echo -e "${BLUE}📋 Setup Instructions:${NC}"
echo ""
echo "1. Terminal 1 - Start Controller (with Redis):"
echo -e "${YELLOW}   make run-controller-redis${NC}"
echo ""
echo "2. Terminal 2 - Start Worker:"
echo -e "${YELLOW}   make run-worker-redis${NC}"
echo ""
echo "3. Terminal 3 - Start Agent (with Redis):"
echo -e "${YELLOW}   make run-agent-redis${NC}"
echo ""
echo "4. Terminal 4 - Test the system:"
echo -e "${YELLOW}   # Update config (should propagate via Redis instantly)${NC}"
echo -e "${YELLOW}   curl -X POST http://localhost:8080/api/v1/config \\${NC}"
echo -e "${YELLOW}        -H 'Content-Type: application/json' \\${NC}"
echo -e "${YELLOW}        -u admin:admin123 \\${NC}"
echo -e "${YELLOW}        -d '{\"url\": \"https://httpbin.org/uuid\"}'${NC}"
echo ""
echo -e "${YELLOW}   # Test worker (should respond immediately with new config)${NC}"
echo -e "${YELLOW}   curl http://localhost:8082/hit${NC}"
echo ""
echo -e "${BLUE}🔍 Verify Redis Mode:${NC}"
echo "   Check agent logs for: 'Started Redis configuration subscriber'"
echo "   Check controller logs for: 'Configuration published to Redis successfully'"
echo ""
echo -e "${BLUE}📊 Redis Monitoring:${NC}"
echo -e "${YELLOW}   redis-cli monitor${NC}    # Watch Redis commands in real-time"
echo -e "${YELLOW}   redis-cli info${NC}       # Redis server info"
echo ""
echo -e "${GREEN}🎉 Ready to test Redis strategy mode locally!${NC}"