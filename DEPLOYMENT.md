# Production Deployment Guide

## GitHub Secrets Configuration

Before deploying to production, configure the following secrets in your GitHub repository:

### Server Access Secrets
```
PROD_HOST      - Production server IP/hostname
PROD_USERNAME  - SSH username for production server  
PROD_PASSWORD  - SSH password for production server
```

### Application Secrets (Optional - defaults will be used)
```
AGENT_PASSWORD  - Agent authentication password (default: prod-secret-123)
ADMIN_PASSWORD  - Admin authentication password (default: admin-prod-456)
```

## Production Architecture

The production deployment uses external load balancing:

```
External Clients → NGINX Load Balancer (port 8085) → Worker Pool (ports 8082-8084)
                                                   
Agents → Direct Worker Connections (no load balancer)
```

### Service Ports
- **Controller**: 8080
- **External Load Balancer**: 8085 (for client traffic)
- **Worker1**: 8082 (direct access)
- **Worker2**: 8083 (direct access) 
- **Worker3**: 8084 (direct access)
- **NATS**: 4222
- **Agents**: 8090-8092 (internal)

## Deployment Process

### Automatic Deployment
Push to `main` branch to trigger automatic deployment:

```bash
git add .
git commit -m "Deploy to production"
git push origin main
```

### Manual Deployment
Trigger deployment manually via GitHub Actions:

1. Go to Actions tab in GitHub repository
2. Select "Deploy to Production" workflow
3. Click "Run workflow"

## Testing Production Deployment

### Health Checks
```bash
# Controller health
curl http://your-server:8080/health

# External load balancer health  
curl http://your-server:8085/health

# Individual worker health
curl http://your-server:8082/health
curl http://your-server:8083/health
curl http://your-server:8084/health
```

### API Testing
```bash
# Create config via load balancer (simulating external client)
curl -X POST http://your-server:8085/worker/configs \
  -H "Content-Type: application/json" \
  -d '{"key": "test-key", "value": "test-value"}'

# Check agent received config (direct worker access)
curl http://your-server:8082/configs/test-key
```

## Architecture Benefits

### External Load Balancing
- **Client Traffic**: Distributed across worker pool via NGINX
- **High Availability**: Failed workers automatically removed from rotation
- **Scalability**: Easy to add more workers to the pool

### Direct Agent Connections
- **Performance**: Agents connect directly to assigned workers
- **Predictable Routing**: 1:1 agent-to-worker mapping
- **Reduced Latency**: No additional load balancer hop for internal traffic

### NATS Integration
- **Real-time Updates**: Config changes pushed immediately via NATS
- **Load Balancing**: Queue groups distribute updates efficiently
- **Resilience**: Automatic reconnection and message persistence

## Monitoring

### Service Status
```bash
ssh user@your-server
cd /root/config-management
docker compose ps
```

### Logs
```bash
# All services
docker compose logs

# Specific service
docker compose logs controller
docker compose logs worker1
docker compose logs agent1
docker compose logs external-lb
```

### Resource Usage
```bash
docker stats
```

## Troubleshooting

### Common Issues

1. **Services not starting**: Check logs and ensure proper image versions
2. **Load balancer not routing**: Verify NGINX config and worker health
3. **NATS connection issues**: Check NATS server status and connectivity
4. **Agent not receiving updates**: Verify NATS queue group subscriptions

### Recovery Commands
```bash
# Restart all services
docker compose down && docker compose up -d

# Restart specific service
docker compose restart controller

# View service logs
docker compose logs -f worker1
```