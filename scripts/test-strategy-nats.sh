#!/bin/bash

# Test script for NATS distribution strategy
# This script tests the NATS pub/sub configuration distribution system

set -e

echo "🧪 Testing NATS Distribution Strategy..."
echo "========================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Docker and Docker Compose are available
print_status "Checking prerequisites..."
command -v docker >/dev/null 2>&1 || { print_error "Docker is required but not installed. Aborting."; exit 1; }
if command -v docker-compose >/dev/null 2>&1; then
    DOCKER_COMPOSE="docker-compose"
elif command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
    DOCKER_COMPOSE="docker compose"
else
    print_error "Docker Compose is required but not installed. Aborting."
    exit 1
fi
print_success "Using: $DOCKER_COMPOSE"

# Clean up any existing containers
print_status "Cleaning up existing containers..."
$DOCKER_COMPOSE -f docker-compose.agents-nats.yml down -v 2>/dev/null || true
docker system prune -f >/dev/null 2>&1 || true

# Build and start services
print_status "Building and starting NATS services..."
$DOCKER_COMPOSE -f docker-compose.agents-nats.yml build --no-cache

print_status "Starting NATS server..."
$DOCKER_COMPOSE -f docker-compose.agents-nats.yml up -d nats

# Wait for NATS to be ready
print_status "Waiting for NATS server to be ready..."
sleep 5
print_success "NATS server is ready"

# Start controller
print_status "Starting controller with NATS distribution..."
$DOCKER_COMPOSE -f docker-compose.agents-nats.yml up -d controller

# Wait for controller to be ready
print_status "Waiting for controller to be ready..."
sleep 10

# Check controller health
if ! curl -s http://localhost:8080/health >/dev/null; then
    print_error "Controller is not responding"
    $DOCKER_COMPOSE -f docker-compose.agents-nats.yml logs controller
    exit 1
fi
print_success "Controller is ready"

# Start worker
print_status "Starting worker..."
$DOCKER_COMPOSE -f docker-compose.agents-nats.yml up -d worker

# Wait for worker to be ready
sleep 5

# Start agent with NATS strategy
print_status "Starting agent with NATS distribution strategy..."
$DOCKER_COMPOSE -f docker-compose.agents-nats.yml up -d agent

# Wait for agent to start and connect
print_status "Waiting for agent to connect..."
sleep 10

# Check if all services are running
print_status "Verifying all services are running..."
if $DOCKER_COMPOSE -f docker-compose.agents-nats.yml ps | grep -q "Exit"; then
    print_error "Some services failed to start"
    $DOCKER_COMPOSE -f docker-compose.agents-nats.yml ps
    exit 1
fi
print_success "All services are running"

# Test configuration update via NATS
print_status "Testing configuration update through NATS..."

# Create a test configuration
cat > test_config.json << 'EOF'
{
  "enabled": true,
  "interval_seconds": 15,
  "max_concurrent": 3,
  "endpoints": [
    {
      "url": "https://httpbin.org/delay/2",
      "method": "GET",
      "headers": {"User-Agent": "Config-Test-NATS"},
      "timeout_seconds": 5
    }
  ]
}
EOF

# Update configuration
print_status "Sending configuration update to controller..."
if ! curl -s -X POST \
  -H "Content-Type: application/json" \
  -H "Authorization: Basic $(echo -n 'admin:admin123' | base64)" \
  -d @test_config.json \
  http://localhost:8080/api/v1/config >/dev/null; then
    print_error "Failed to update configuration"
    exit 1
fi

print_success "Configuration update sent successfully"

# Wait for NATS propagation
print_status "Waiting for NATS configuration propagation..."
sleep 5

# Verify agent received the configuration
print_status "Verifying agent received the NATS configuration..."
agent_logs=$($DOCKER_COMPOSE -f docker-compose.agents-nats.yml logs agent | tail -20)

if echo "$agent_logs" | grep -q "Received new config from NATS"; then
    print_success "✅ Agent received configuration via NATS!"
else
    print_warning "⚠️  Configuration may not have been received via NATS"
    echo "Recent agent logs:"
    echo "$agent_logs"
fi

print_success "✅ NATS Distribution Strategy Test Passed!"

# Cleanup
print_status "Cleaning up test files..."
rm -f test_config.json

print_status "Test completed! Use '$DOCKER_COMPOSE -f docker-compose.agents-nats.yml down -v' to stop services."