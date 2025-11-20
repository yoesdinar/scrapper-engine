# Docker Multi-Agent/Worker Testing Guide

## Architecture

This test setup creates a distributed system with:
- **1 Controller** (port 8080) - Central configuration management
- **3 Workers** (ports 8082, 8083, 8084) - Execute configured tasks
- **3 Agents** (no exposed ports) - Poll controller and forward config to workers
  - Agent1 → Worker1
  - Agent2 → Worker2
  - Agent3 → Worker3

## Prerequisites

- Docker Desktop installed and running on Mac
- At least 2GB free RAM
- Ports 8080, 8082-8084 available

## Quick Start

### 1. Build and Start All Services

```bash
# Start all services (controller + 3 agents + 3 workers)
docker-compose -f docker-compose.test.yml up -d

# Check all services are running
docker-compose -f docker-compose.test.yml ps

# View logs for all services
docker-compose -f docker-compose.test.yml logs -f
```

### 2. Run Complete Test Suite

```bash
# Automated test script (recommended)
./scripts/test-docker-multi.sh
```

This script will:
- ✓ Start all services
- ✓ Wait for health checks
- ✓ Set initial configuration
- ✓ Simulate 3 users hitting each worker (9 total requests)
- ✓ Verify agent registrations
- ✓ Update configuration
- ✓ Test workers with new configuration

### 3. Manual Testing

#### Set Configuration

```bash
# Set worker configuration to hit httpbin.org
curl -X POST http://localhost:8080/api/v1/config \
  -u admin:admin123 \
  -H "Content-Type: application/json" \
  -d '{"url": "https://httpbin.org/uuid"}'
```

#### Wait for Config Propagation
```bash
# Wait 5-10 seconds for agents to poll and forward config to workers
sleep 5
```

#### Test Each Worker

```bash
# Hit Worker 1 (port 8082)
curl http://localhost:8082/hit

# Hit Worker 2 (port 8083)
curl http://localhost:8083/hit

# Hit Worker 3 (port 8084)
curl http://localhost:8084/hit
```

#### Simulate Multiple Users

```bash
# 3 users hit Worker 1
for i in 1 2 3; do curl http://localhost:8082/hit & done; wait

# 3 users hit Worker 2
for i in 1 2 3; do curl http://localhost:8083/hit & done; wait

# 3 users hit Worker 3
for i in 1 2 3; do curl http://localhost:8084/hit & done; wait
```

### 4. Concurrent Load Testing

```bash
# Send 10 requests to each worker simultaneously (30 total)
./scripts/test-concurrent-load.sh 10

# Send 50 requests to each worker simultaneously (150 total)
./scripts/test-concurrent-load.sh 50
```

## Monitoring

### Check Agent Status

```bash
# View all registered agents
curl -s http://localhost:8080/api/v1/agents \
  -u admin:admin123 | jq '.'
```

### View Service Logs

```bash
# All services
docker-compose -f docker-compose.test.yml logs -f

# Specific service
docker-compose -f docker-compose.test.yml logs -f controller
docker-compose -f docker-compose.test.yml logs -f agent1
docker-compose -f docker-compose.test.yml logs -f worker1

# Last 50 lines
docker-compose -f docker-compose.test.yml logs --tail=50 agent2
```

### Check Service Health

```bash
# Controller
curl http://localhost:8080/health

# Worker 1
curl http://localhost:8082/health

# Worker 2
curl http://localhost:8083/health

# Worker 3
curl http://localhost:8084/health
```

## Configuration Updates

### Change Target URL

```bash
# Update to different endpoint
curl -X POST http://localhost:8080/api/v1/config \
  -u admin:admin123 \
  -H "Content-Type: application/json" \
  -d '{"url": "https://api.ipify.org?format=json"}'

# Wait for propagation
sleep 5

# Test workers with new config
curl http://localhost:8082/hit
curl http://localhost:8083/hit
curl http://localhost:8084/hit
```

### Adjust Poll Interval

```bash
# Set poll interval to 10 seconds
curl -X POST http://localhost:8080/api/v1/config?poll_interval=10 \
  -u admin:admin123 \
  -H "Content-Type: application/json" \
  -d '{"url": "https://httpbin.org/headers"}'
```

## Test Scenarios

### Scenario 1: Configuration Propagation
1. Start all services
2. Set configuration via controller
3. Verify all 3 workers receive and execute the same config

### Scenario 2: Concurrent Access
1. Set configuration
2. Launch 9 users (3 per worker) simultaneously
3. Verify all requests succeed

### Scenario 3: Dynamic Reconfiguration
1. Set initial configuration
2. Test all workers
3. Update configuration
4. Verify all workers switch to new config

### Scenario 4: Agent Resilience
1. Stop one agent: `docker stop config-agent2`
2. Update configuration
3. Verify agent1 and agent3 still work
4. Restart agent: `docker start config-agent2`
5. Verify agent2 catches up

## Cleanup

```bash
# Stop all services
docker-compose -f docker-compose.test.yml down

# Stop and remove volumes (clean slate)
docker-compose -f docker-compose.test.yml down -v

# Remove images
docker-compose -f docker-compose.test.yml down --rmi all
```

## Troubleshooting

### Services not starting
```bash
# Check Docker Desktop is running
docker ps

# View startup logs
docker-compose -f docker-compose.test.yml logs

# Rebuild images
docker-compose -f docker-compose.test.yml build --no-cache
```

### Port conflicts
```bash
# Check what's using the ports
lsof -i :8080
lsof -i :8082
lsof -i :8083
lsof -i :8084

# Kill conflicting processes or change ports in docker-compose.test.yml
```

### Workers not receiving config
```bash
# Check agent logs
docker-compose -f docker-compose.test.yml logs agent1
docker-compose -f docker-compose.test.yml logs agent2
docker-compose -f docker-compose.test.yml logs agent3

# Verify controller has the config
curl http://localhost:8080/api/v1/config -u agent:agent123
```

### High CPU/Memory usage
```bash
# Check resource usage
docker stats

# Limit resources in docker-compose.test.yml:
# deploy:
#   resources:
#     limits:
#       cpus: '0.5'
#       memory: 256M
```

## API Endpoints

### Controller (port 8080)
- `GET /health` - Health check
- `POST /api/v1/register` - Register agent (requires agent auth)
- `GET /api/v1/config` - Get current config (requires agent auth)
- `POST /api/v1/config` - Update config (requires admin auth)
- `GET /api/v1/agents` - List agents (requires admin auth)
- `GET /swagger/index.html` - API documentation

### Workers (ports 8082, 8083, 8084)
- `GET /health` - Health check
- `POST /config` - Receive config from agent
- `GET /hit` - Execute configured task
- `GET /swagger/index.html` - API documentation

## Performance Notes

On a typical Mac with Docker Desktop:
- Each service uses ~20-50MB RAM
- Total RAM usage: ~300-500MB
- Can handle 100+ concurrent requests
- Config propagation: 1-5 seconds
