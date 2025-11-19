#!/bin/bash

echo "=== Testing Docker Build for Multi-Agent Setup ==="
echo ""

echo "Step 1: Building worker image..."
docker build -t coding-test-worker -f worker/Dockerfile . && echo "✓ Worker built successfully" || echo "✗ Worker build failed"

echo ""
echo "Step 2: Building agent image..."
docker build -t coding-test-agent -f agent/Dockerfile . && echo "✓ Agent built successfully" || echo "✗ Agent build failed"

echo ""
echo "Step 3: Building controller image..."
docker build -t coding-test-controller -f controller/Dockerfile . && echo "✓ Controller built successfully" || echo "✗ Controller build failed"

echo ""
echo "=== Build Test Complete ==="
echo ""
echo "If all builds succeeded, you can now run:"
echo "  ./scripts/test-docker-multi.sh"
