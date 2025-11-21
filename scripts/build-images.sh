#!/bin/bash

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}🔨 Building Multi-Platform Docker Images${NC}"
echo "========================================"

echo -e "${YELLOW}📋 Building for local development and testing...${NC}"

# Function to build with fallback
build_service() {
    local service=$1
    local context=$2
    
    echo -e "${BLUE}Building ${service}...${NC}"
    
    # Try with buildx first for multi-platform support
    if docker buildx build --platform linux/amd64,linux/arm64 \
        -t "local/${service}:latest" \
        --load \
        "${context}" 2>/dev/null; then
        echo -e "${GREEN}✅ ${service} built with multi-platform support${NC}"
    else
        # Fallback to regular build for current platform
        echo -e "${YELLOW}⚠️  Buildx failed, using regular build for current platform${NC}"
        if docker build -t "local/${service}:latest" "${context}"; then
            echo -e "${GREEN}✅ ${service} built for current platform${NC}"
        else
            echo -e "${RED}❌ Failed to build ${service}${NC}"
            return 1
        fi
    fi
    
    return 0
}

# Build all services
echo ""
echo "Building services..."

if ! build_service "controller" "./controller"; then
    exit 1
fi

if ! build_service "agent" "./agent"; then
    exit 1
fi

if ! build_service "worker" "./worker"; then
    exit 1
fi

echo ""
echo -e "${GREEN}🎉 All services built successfully!${NC}"
echo ""
echo "Built images:"
echo "- local/controller:latest"
echo "- local/agent:latest" 
echo "- local/worker:latest"
echo ""
echo "You can now run:"
echo "- ./scripts/test-strategy-pattern.sh"
echo "- ./scripts/test-local-redis.sh"
echo "- docker-compose commands with local images"