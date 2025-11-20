#!/bin/bash

# Production Deployment Script for Configuration Management System
# Target VM: 103.157.116.91
# Usage: ./scripts/deploy-production.sh

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
VM_IP="103.157.116.91"
VM_USER="root"
COMPOSE_FILE="docker-compose.production.yml"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Configuration Management System${NC}"
echo -e "${BLUE}Production Deployment Script${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Check if we're running on the VM or locally
if [ "$(hostname -I | grep -c "$VM_IP")" -eq 0 ]; then
    echo -e "${YELLOW}⚠️  This script is meant to run on the production VM${NC}"
    echo -e "${YELLOW}Please run it on $VM_IP or copy files there first${NC}"
    echo ""
    echo -e "${BLUE}To deploy:${NC}"
    echo -e "1. Copy project to VM: ${GREEN}scp -r . $VM_USER@$VM_IP:~/config-management/${NC}"
    echo -e "2. SSH to VM: ${GREEN}ssh $VM_USER@$VM_IP${NC}"
    echo -e "3. Run deployment: ${GREEN}cd ~/config-management && ./scripts/deploy-production.sh${NC}"
    exit 0
fi

echo -e "${GREEN}✓ Running on production VM${NC}"
echo ""

# Step 1: Check prerequisites
echo -e "${BLUE}Step 1: Checking prerequisites...${NC}"

# Check Docker
if ! command -v docker &> /dev/null; then
    echo -e "${RED}✗ Docker not found${NC}"
    echo -e "${YELLOW}Installing Docker...${NC}"
    curl -fsSL https://get.docker.com -o get-docker.sh
    sh get-docker.sh
    rm get-docker.sh
    systemctl enable docker
    systemctl start docker
    echo -e "${GREEN}✓ Docker installed${NC}"
else
    echo -e "${GREEN}✓ Docker found: $(docker --version)${NC}"
fi

# Check Docker Compose
if ! docker compose version &> /dev/null; then
    echo -e "${RED}✗ Docker Compose not found${NC}"
    echo -e "${YELLOW}Please install Docker Compose v2${NC}"
    exit 1
else
    echo -e "${GREEN}✓ Docker Compose found: $(docker compose version)${NC}"
fi

# Check compose file
if [ ! -f "$COMPOSE_FILE" ]; then
    echo -e "${RED}✗ $COMPOSE_FILE not found${NC}"
    exit 1
else
    echo -e "${GREEN}✓ $COMPOSE_FILE found${NC}"
fi

# Check nginx config
if [ ! -f "nginx/nginx.conf" ]; then
    echo -e "${RED}✗ nginx/nginx.conf not found${NC}"
    exit 1
else
    echo -e "${GREEN}✓ nginx/nginx.conf found${NC}"
fi

echo ""

# Step 2: Stop existing containers (if any)
echo -e "${BLUE}Step 2: Stopping existing containers...${NC}"
if docker compose -f "$COMPOSE_FILE" ps -q 2>/dev/null | grep -q .; then
    docker compose -f "$COMPOSE_FILE" down
    echo -e "${GREEN}✓ Stopped existing containers${NC}"
else
    echo -e "${YELLOW}⚠️  No existing containers found${NC}"
fi
echo ""

# Step 3: Build images
echo -e "${BLUE}Step 3: Building Docker images...${NC}"
docker compose -f "$COMPOSE_FILE" build --no-cache
echo -e "${GREEN}✓ Images built successfully${NC}"
echo ""

# Step 4: Start services
echo -e "${BLUE}Step 4: Starting services...${NC}"
docker compose -f "$COMPOSE_FILE" up -d
echo -e "${GREEN}✓ Services started${NC}"
echo ""

# Step 5: Wait for services to be healthy
echo -e "${BLUE}Step 5: Waiting for services to be healthy...${NC}"
echo -e "${YELLOW}This may take 30-60 seconds...${NC}"

MAX_WAIT=120
ELAPSED=0
INTERVAL=5

while [ $ELAPSED -lt $MAX_WAIT ]; do
    HEALTHY=$(docker compose -f "$COMPOSE_FILE" ps --format json | jq -r '.[] | select(.Health=="healthy") | .Service' 2>/dev/null | wc -l)
    TOTAL=$(docker compose -f "$COMPOSE_FILE" ps --format json | jq -r '.[] | select(.Health) | .Service' 2>/dev/null | wc -l)
    
    if [ "$TOTAL" -eq 0 ]; then
        # Fallback if jq not available
        RUNNING=$(docker compose -f "$COMPOSE_FILE" ps -q 2>/dev/null | wc -l)
        echo -e "${YELLOW}⏳ $RUNNING services running (health checks pending)...${NC}"
    else
        echo -e "${YELLOW}⏳ $HEALTHY/$TOTAL services healthy...${NC}"
        
        if [ "$HEALTHY" -eq "$TOTAL" ] && [ "$TOTAL" -gt 0 ]; then
            echo -e "${GREEN}✓ All services are healthy!${NC}"
            break
        fi
    fi
    
    sleep $INTERVAL
    ELAPSED=$((ELAPSED + INTERVAL))
done

if [ $ELAPSED -ge $MAX_WAIT ]; then
    echo -e "${YELLOW}⚠️  Timeout waiting for services. Checking status...${NC}"
fi
echo ""

# Step 6: Display service status
echo -e "${BLUE}Step 6: Service Status${NC}"
docker compose -f "$COMPOSE_FILE" ps
echo ""

# Step 7: Display logs (last 10 lines per service)
echo -e "${BLUE}Step 7: Recent logs${NC}"
echo -e "${YELLOW}Controller logs:${NC}"
docker compose -f "$COMPOSE_FILE" logs --tail=10 controller
echo ""
echo -e "${YELLOW}Nginx logs:${NC}"
docker compose -f "$COMPOSE_FILE" logs --tail=10 nginx
echo ""

# Step 8: Display access information
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}✓ Deployment Complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "${BLUE}Access URLs:${NC}"
echo -e "  Homepage:          ${GREEN}http://$VM_IP/${NC}"
echo -e "  Admin Panel:       ${GREEN}http://$VM_IP/admin/swagger/index.html${NC}"
echo -e "  Admin Credentials: ${YELLOW}admin / admin123${NC}"
echo ""
echo -e "  Worker 1 API:      ${GREEN}http://$VM_IP/worker1/swagger/index.html${NC}"
echo -e "  Worker 2 API:      ${GREEN}http://$VM_IP/worker2/swagger/index.html${NC}"
echo -e "  Worker 3 API:      ${GREEN}http://$VM_IP/worker3/swagger/index.html${NC}"
echo ""
echo -e "${BLUE}Health Checks:${NC}"
echo -e "  Nginx:       ${GREEN}http://$VM_IP/nginx-health${NC}"
echo -e "  Controller:  ${GREEN}http://$VM_IP/admin/health${NC}"
echo -e "  Worker 1:    ${GREEN}http://$VM_IP/worker1/health${NC}"
echo -e "  Worker 2:    ${GREEN}http://$VM_IP/worker2/health${NC}"
echo -e "  Worker 3:    ${GREEN}http://$VM_IP/worker3/health${NC}"
echo ""
echo -e "${BLUE}Agent Credentials:${NC}"
echo -e "  Agent 1: ${YELLOW}agent1 / agent1pass${NC}"
echo -e "  Agent 2: ${YELLOW}agent2 / agent2pass${NC}"
echo -e "  Agent 3: ${YELLOW}agent3 / agent3pass${NC}"
echo ""
echo -e "${BLUE}Useful Commands:${NC}"
echo -e "  View logs:         ${GREEN}docker compose -f $COMPOSE_FILE logs -f [service]${NC}"
echo -e "  Restart service:   ${GREEN}docker compose -f $COMPOSE_FILE restart [service]${NC}"
echo -e "  Stop all:          ${GREEN}docker compose -f $COMPOSE_FILE down${NC}"
echo -e "  Restart all:       ${GREEN}docker compose -f $COMPOSE_FILE restart${NC}"
echo ""
echo -e "${YELLOW}Next Steps:${NC}"
echo -e "1. Test the deployment: ${GREEN}./scripts/test-production.sh${NC} (from your local machine)"
echo -e "2. Open browser: ${GREEN}http://$VM_IP/${NC}"
echo -e "3. Register agents and create configurations via Admin Panel"
echo ""
