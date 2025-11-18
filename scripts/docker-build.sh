#!/bin/bash

set -e

echo "======================================"
echo "Building Docker Images"
echo "======================================"
echo ""

# Build context needs to be at root to access pkg/
cd "$(dirname "$0")/.."

echo "Building Controller image..."
docker build -f controller/Dockerfile -t config-controller:latest .

echo "Building Worker image..."
docker build -f worker/Dockerfile -t config-worker:latest .

echo "Building Agent image..."
docker build -f agent/Dockerfile -t config-agent:latest .

echo ""
echo "======================================"
echo "Docker Images Built Successfully!"
echo "======================================"
echo ""
echo "Available images:"
docker images | grep config-

echo ""
echo "To run the services:"
echo "  1. Start Controller:"
echo "     docker-compose -f docker-compose.controller.yml up -d"
echo ""
echo "  2. Start Agent + Worker:"
echo "     docker-compose -f docker-compose.agent-worker.yml up -d"
