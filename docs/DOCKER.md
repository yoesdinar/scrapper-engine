# Docker Deployment Guide

## Overview

This project includes Docker support for all three services:
- **Controller**: Standalone service with SQLite database
- **Agent + Worker**: Combined deployment where agent manages worker

## Prerequisites

- Docker installed (version 20.10+)
- Docker Compose installed (version 1.29+)
- At least 2GB available RAM
- Ports 8080 (controller) and 8082 (worker) available

## Quick Start

### Option 1: Build and Run All Services

```bash
# Build all Docker images
./scripts/docker-build.sh

# Start Controller
docker-compose -f docker-compose.controller.yml up -d

# Start Agent + Worker
docker-compose -f docker-compose.agent-worker.yml up -d

# Check logs
docker-compose -f docker-compose.controller.yml logs -f
docker-compose -f docker-compose.agent-worker.yml logs -f
```

### Option 2: Step-by-Step Deployment

#### 1. Build Images

```bash
# From project root
cd /path/to/coding-test

# Build controller
docker build -f controller/Dockerfile -t config-controller:latest .

# Build worker
docker build -f worker/Dockerfile -t config-worker:latest .

# Build agent
docker build -f agent/Dockerfile -t config-agent:latest .
```

#### 2. Start Controller

```bash
docker-compose -f docker-compose.controller.yml up -d
```

Wait ~5 seconds for controller to initialize the database.

#### 3. Start Agent + Worker

```bash
docker-compose -f docker-compose.agent-worker.yml up -d
```

## Configuration

### Controller Environment Variables

```yaml
CONTROLLER_PORT: 8080                # HTTP server port
CONTROLLER_DB_PATH: /app/data/controller.db  # SQLite database path
CONTROLLER_AGENT_USER: agent         # Agent authentication username
CONTROLLER_AGENT_PASS: agent123      # Agent authentication password
CONTROLLER_ADMIN_USER: admin         # Admin authentication username
CONTROLLER_ADMIN_PASS: admin123      # Admin authentication password
```

### Agent Environment Variables

```yaml
CONTROLLER_URL: http://host.docker.internal:8080  # Controller URL
CONTROLLER_USER: agent               # Authentication username
CONTROLLER_PASSWORD: agent123        # Authentication password
WORKER_URL: http://worker:8082       # Worker URL (internal network)
CACHE_FILE: /app/cache/agent_config.cache  # Config cache file path
```

### Worker Environment Variables

```yaml
WORKER_PORT: 8082                    # HTTP server port
```

## Docker Compose Files

### controller (docker-compose.controller.yml)

- **Service**: controller
- **Ports**: 8080:8080
- **Volumes**: controller-data (persists SQLite database)
- **Health Check**: `wget http://localhost:8080/health`
- **Restart Policy**: unless-stopped

### Agent + Worker (docker-compose.agent-worker.yml)

- **Services**:
  - **worker**: HTTP service on port 8082
  - **agent**: Polls controller, manages worker
- **Network**: Both services on same Docker network
- **Volumes**: agent-cache (persists configuration cache)
- **Dependencies**: Agent waits for worker to start
- **Special**: Uses `host.docker.internal` to reach controller on host

## Networking

### Internal Communication
- Agent → Worker: `http://worker:8082` (Docker internal DNS)
- Worker is NOT exposed directly (only through agent)

### External Communication
- Agent → Controller: `http://host.docker.internal:8080` (host machine)
- User → Worker: `http://localhost:8082` (port forwarding)
- User → Controller: `http://localhost:8080` (port forwarding)

## Data Persistence

### Controller Volume
```bash
# View database
docker exec config-controller ls -la /app/data/

# Backup database
docker cp config-controller:/app/data/controller.db ./controller-backup.db

# Restore database
docker cp ./controller-backup.db config-controller:/app/data/controller.db
docker-compose -f docker-compose.controller.yml restart
```

### Agent Cache Volume
```bash
# View cache
docker exec config-agent ls -la /app/cache/

# Clear cache (agent will re-fetch from controller)
docker exec config-agent rm /app/cache/agent_config.cache
docker-compose -f docker-compose.agent-worker.yml restart agent
```

## Testing Dockerized Services

```bash
# Test controller health
curl http://localhost:8080/health

# Update configuration
curl -X POST http://localhost:8080/api/v1/config \
  -u admin:admin123 \
  -H "Content-Type: application/json" \
  -d '{"url":"https://ip.me"}'

# Wait 35 seconds for agent to poll

# Test worker
curl http://localhost:8082/hit
```

## Monitoring

### View Logs

```bash
# Controller logs
docker-compose -f docker-compose.controller.yml logs -f

# Agent logs
docker-compose -f docker-compose.agent-worker.yml logs -f agent

# Worker logs
docker-compose -f docker-compose.agent-worker.yml logs -f worker

# All services
docker-compose -f docker-compose.controller.yml logs -f &
docker-compose -f docker-compose.agent-worker.yml logs -f
```

### Check Status

```bash
# Controller
docker-compose -f docker-compose.controller.yml ps

# Agent + Worker
docker-compose -f docker-compose.agent-worker.yml ps

# Resource usage
docker stats
```

## Troubleshooting

### Controller Won't Start

```bash
# Check logs
docker-compose -f docker-compose.controller.yml logs

# Common issues:
# - Port 8080 already in use
# - Database permissions
# - Insufficient disk space

# Solution: Check port availability
lsof -i :8080
```

### Agent Can't Connect to Controller

```bash
# Check if controller is reachable from agent
docker exec config-agent ping host.docker.internal

# Check controller URL
docker exec config-agent env | grep CONTROLLER_URL

# Manual test from agent container
docker exec config-agent wget -O- http://host.docker.internal:8080/health
```

### Worker Not Receiving Config

```bash
# Check agent logs
docker-compose -f docker-compose.agent-worker.yml logs agent

# Check if agent registered
curl http://localhost:8080/api/v1/agents -u admin:admin123

# Force agent to poll immediately
docker-compose -f docker-compose.agent-worker.yml restart agent
```

### Database Corruption

```bash
# Stop controller
docker-compose -f docker-compose.controller.yml down

# Remove volume (CAUTION: loses all data)
docker volume rm coding-test_controller-data

# Restart controller (creates fresh database)
docker-compose -f docker-compose.controller.yml up -d
```

## Cleanup

### Stop Services

```bash
# Stop controller
docker-compose -f docker-compose.controller.yml down

# Stop agent + worker
docker-compose -f docker-compose.agent-worker.yml down
```

### Remove Everything

```bash
# Stop and remove containers, networks, volumes
docker-compose -f docker-compose.controller.yml down -v
docker-compose -f docker-compose.agent-worker.yml down -v

# Remove images
docker rmi config-controller config-worker config-agent

# Remove all unused Docker resources
docker system prune -a
```

## Production Considerations

### Security

1. **Change default credentials** in environment variables
2. **Use secrets management** (Docker Secrets, HashiCorp Vault)
3. **Enable TLS/HTTPS** with reverse proxy (nginx, traefik)
4. **Network isolation** with custom Docker networks
5. **Read-only containers** where possible

### Scalability

1. **Multiple agents**: Start multiple agent-worker pairs
```bash
docker-compose -f docker-compose.agent-worker.yml up -d --scale agent=3 --scale worker=3
```

2. **Load balancer**: Add nginx/traefik in front of workers
3. **External database**: Replace SQLite with PostgreSQL/MySQL
4. **Redis/NATS**: Implement pub/sub for better scaling (Phase 2)

### High Availability

1. **Controller HA**: Run multiple controller instances with shared database
2. **Database replication**: Use PostgreSQL with streaming replication
3. **Health checks**: Configure proper health check intervals
4. **Auto-restart**: Use `restart: always` policy
5. **Monitoring**: Add Prometheus + Grafana

### Resource Limits

Add to docker-compose files:

```yaml
services:
  controller:
    deploy:
      resources:
        limits:
          cpus: '0.5'
          memory: 512M
        reservations:
          cpus: '0.25'
          memory: 256M
```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Build and Push Docker Images

on:
  push:
    branches: [ main ]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      
      - name: Build images
        run: |
          docker build -f controller/Dockerfile -t myregistry/config-controller:${{ github.sha }} .
          docker build -f worker/Dockerfile -t myregistry/config-worker:${{ github.sha }} .
          docker build -f agent/Dockerfile -t myregistry/config-agent:${{ github.sha }} .
      
      - name: Push images
        run: |
          docker push myregistry/config-controller:${{ github.sha }}
          docker push myregistry/config-worker:${{ github.sha }}
          docker push myregistry/config-agent:${{ github.sha }}
```

## Next Steps

1. **Test the deployment**: Follow the Quick Start section
2. **Customize configuration**: Update environment variables for your needs
3. **Set up monitoring**: Add logging and metrics collection
4. **Plan for production**: Implement security and HA features
5. **Scale horizontally**: Add more agent-worker pairs as needed
