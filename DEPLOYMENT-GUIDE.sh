#!/bin/bash

# Quick Deployment Guide
# This script demonstrates the deployment process

echo "=== GitHub Actions Deployment Setup ==="
echo ""
echo "1. SETUP GITHUB SECRETS"
echo "   Go to: Repository → Settings → Secrets → Actions"
echo "   Add these secrets:"
echo "   - PROD_HOST: 103.157.116.91"
echo "   - PROD_USERNAME: root"
echo "   - PROD_PASSWORD: g!+D7^PCoz"
echo ""

echo "2. COMMIT YOUR CHANGES"
echo "   git add ."
echo "   git commit -m 'Your changes'"
echo ""

echo "3. CREATE AND PUSH A RELEASE TAG"
echo "   git tag v1.0.0"
echo "   git push origin v1.0.0"
echo ""

echo "4. WATCH THE DEPLOYMENT"
echo "   Go to: Repository → Actions tab"
echo "   Watch the 'Build and Deploy to Production' workflow"
echo ""

echo "5. VERIFY DEPLOYMENT"
echo "   ./scripts/test-production.sh"
echo ""

echo "=== What Happens Automatically ==="
echo "✓ Builds controller, worker, and agent Docker images"
echo "✓ Pushes images to GitHub Container Registry (ghcr.io)"
echo "✓ Uploads docker-compose.production.yml to server"
echo "✓ Uploads nginx config to server"
echo "✓ Pulls new images on production server"
echo "✓ Restarts all services with new images"
echo ""

echo "=== Manual Deployment (if needed) ==="
echo "ssh root@103.157.116.91"
echo "cd /root/config-management"
echo "export IMAGE_TAG=v1.0.0"
echo "docker compose -f docker-compose.production.yml pull"
echo "docker compose -f docker-compose.production.yml up -d"
echo ""

echo "For full documentation, see: GITHUB-ACTIONS-DEPLOYMENT.md"
