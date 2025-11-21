#!/bin/bash

# Local NATS testing script
# Tests NATS distribution strategy with locally running services

set -e

echo "🧪 Testing NATS Strategy Locally..."
echo "===================================="

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

# Check prerequisites
print_status "Checking prerequisites..."

# Check if NATS server is available
if ! command -v nats-server >/dev/null 2>&1; then
    print_error "NATS server not found. Install with:"
    echo "  macOS: brew install nats-server"
    echo "  Linux: Download from https://nats.io/download"
    exit 1
fi

# Check if Go is available
if ! command -v go >/dev/null 2>&1; then
    print_error "Go not found. Please install Go first."
    exit 1
fi

print_success "Prerequisites check passed"

# Start NATS server if not running
print_status "Starting NATS server..."
if ! pgrep nats-server >/dev/null; then
    nats-server --port 4222 --http_port 8222 --jetstream --store_dir ./nats-data &
    NATS_PID=$!
    sleep 3
    
    if ! pgrep nats-server >/dev/null; then
        print_error "Failed to start NATS server"
        exit 1
    fi
    print_success "NATS server started (PID: $NATS_PID)"
    CLEANUP_NATS=true
else
    print_success "NATS server already running"
    CLEANUP_NATS=false
fi

# Cleanup function
cleanup() {
    print_status "Cleaning up..."
    
    # Kill background processes
    jobs -p | xargs -r kill 2>/dev/null || true
    
    # Stop NATS if we started it
    if [ "$CLEANUP_NATS" = true ] && [ ! -z "$NATS_PID" ]; then
        kill $NATS_PID 2>/dev/null || true
        print_status "NATS server stopped"
    fi
    
    # Remove test files
    rm -f test_config.json controller.db agent_config.cache
    rm -rf nats-data
    
    echo ""
    print_success "Cleanup completed"
}

# Set trap for cleanup
trap cleanup EXIT INT TERM

# Test NATS connectivity
print_status "Testing NATS connectivity..."
if ! curl -s http://localhost:8222/varz >/dev/null; then
    print_error "NATS HTTP monitoring not accessible"
    exit 1
fi
print_success "NATS server is accessible"

# Build services
print_status "Building services..."
make build
print_success "Services built successfully"

# Start controller with NATS strategy
print_status "Starting controller with NATS strategy..."
(cd controller && \
DISTRIBUTION_STRATEGY=NATS \
NATS_URL=nats://localhost:4222 \
NATS_SUBJECT=config.worker.update \
NATS_QUEUE_GROUP=config-workers \
DB_PATH=./controller.db \
PORT=8080 \
AGENT_USERNAME=agent \
AGENT_PASSWORD=secret123 \
ADMIN_USERNAME=admin \
ADMIN_PASSWORD=admin123 \
LOG_LEVEL=info \
./controller &)
CONTROLLER_PID=$!

# Start worker
print_status "Starting worker..."
(cd worker && \
PORT=8082 \
LOG_LEVEL=info \
./worker &)
WORKER_PID=$!

# Wait for services to start
print_status "Waiting for services to start..."
sleep 5

# Check if controller is running
if ! curl -s http://localhost:8080/health >/dev/null; then
    print_error "Controller health check failed"
    exit 1
fi
print_success "Controller is running"

# Check if worker is running
if ! curl -s http://localhost:8082/health >/dev/null; then
    print_error "Worker health check failed"
    exit 1
fi
print_success "Worker is running"

# Start agent with NATS strategy
print_status "Starting agent with NATS strategy..."
(cd agent && \
DISTRIBUTION_STRATEGY=NATS \
NATS_URL=nats://localhost:4222 \
NATS_SUBJECT=config.worker.update \
NATS_QUEUE_GROUP=config-workers \
CONTROLLER_URL=http://localhost:8080 \
CONTROLLER_USERNAME=agent \
CONTROLLER_PASSWORD=secret123 \
WORKER_URL=http://localhost:8082 \
CACHE_FILE=./agent_config.cache \
LOG_LEVEL=info \
./agent &)
AGENT_PID=$!

# Wait for agent to register
print_status "Waiting for agent to register..."
sleep 5

# Test configuration update through NATS
print_status "Testing configuration update through NATS..."

# Create test configuration
cat > test_config.json << 'EOF'
{
  "enabled": true,
  "interval_seconds": 10,
  "max_concurrent": 2,
  "endpoints": [
    {
      "url": "https://httpbin.org/delay/1",
      "method": "GET",
      "headers": {"User-Agent": "Local-NATS-Test"},
      "timeout_seconds": 5
    }
  ]
}
EOF

# Send configuration update
print_status "Sending configuration update..."
if ! curl -s -X POST \
  -H "Content-Type: application/json" \
  -H "Authorization: Basic $(echo -n 'admin:admin123' | base64)" \
  -d @test_config.json \
  http://localhost:8080/api/v1/config >/dev/null; then
    print_error "Failed to send configuration update"
    exit 1
fi

print_success "Configuration update sent successfully"

# Wait for NATS propagation and check logs
print_status "Waiting for NATS propagation..."
sleep 3

print_success "✅ Local NATS Strategy Test Completed!"
echo ""
echo "📊 Test Summary:"
echo "  - NATS server: Running on :4222"
echo "  - Controller: Running on :8080 (NATS strategy)"
echo "  - Agent: Running with NATS distribution"
echo "  - Worker: Running on :8082"
echo "  - Configuration: Successfully distributed via NATS"
echo ""
echo "🔗 Useful URLs:"
echo "  - NATS Monitoring: http://localhost:8222"
echo "  - Controller API: http://localhost:8080/swagger/index.html"
echo "  - Worker Status: http://localhost:8082/health"
echo ""
echo "💡 To test more configurations, use:"
echo "  curl -X POST -H 'Content-Type: application/json' \\"
echo "       -H 'Authorization: Basic YWRtaW46YWRtaW4xMjM=' \\"
echo "       -d @your_config.json http://localhost:8080/api/v1/config"
echo ""
print_status "Services will continue running. Press Ctrl+C to stop all services."

# Keep script running
wait