# Production Deployment - Configuration Management System

## Deployment Information

**Production VM:** 103.157.116.91  
**Deployment Date:** November 19, 2025  
**Status:** ✅ All systems operational (11/11 tests passed)

## Architecture

The production system consists of:
- **1 Nginx Server** (port 80) - API Gateway & Load Balancer
- **1 Controller** (internal port 8080) - Central configuration management
- **3 Workers** (internal port 8082 each) - Configuration execution
- **3 Agents** - Configuration fetchers and executors

All services run in Docker containers on the same VM with internal networking.

## Access URLs

### Public URLs (via Nginx)

- **Homepage:** http://103.157.116.91/
- **Admin Panel (Swagger):** http://103.157.116.91/admin/swagger/index.html
  - Credentials: `admin` / `admin123`
- **Worker 1 API:** http://103.157.116.91/worker1/swagger/index.html
- **Worker 2 API:** http://103.157.116.91/worker2/swagger/index.html
- **Worker 3 API:** http://103.157.116.91/worker3/swagger/index.html

### Health Check URLs

- Nginx: http://103.157.116.91/nginx-health
- Controller: http://103.157.116.91/admin/health
- Worker 1: http://103.157.116.91/worker1/health
- Worker 2: http://103.157.116.91/worker2/health
- Worker 3: http://103.157.116.91/worker3/health

## Credentials

### Admin Access
- Username: `admin`
- Password: `admin123`

### Agent Credentials
- Agent 1: `agent1` / `agent1pass`
- Agent 2: `agent2` / `agent2pass`
- Agent 3: `agent3` / `agent3pass`

## Management Commands

### SSH Access
```bash
ssh root@103.157.116.91
cd ~/config-management
```

### View Service Status
```bash
docker compose -f docker-compose.production.yml ps
```

### View Logs
```bash
# All services
docker compose -f docker-compose.production.yml logs -f

# Specific service
docker compose -f docker-compose.production.yml logs -f controller
docker compose -f docker-compose.production.yml logs -f worker1
docker compose -f docker-compose.production.yml logs -f agent1
docker compose -f docker-compose.production.yml logs -f nginx
```

### Restart Services
```bash
# Restart all
docker compose -f docker-compose.production.yml restart

# Restart specific service
docker compose -f docker-compose.production.yml restart controller
docker compose -f docker-compose.production.yml restart worker1
```

### Stop All Services
```bash
docker compose -f docker-compose.production.yml down
```

### Start All Services
```bash
docker compose -f docker-compose.production.yml up -d
```

### Rebuild and Restart
```bash
docker compose -f docker-compose.production.yml build --no-cache
docker compose -f docker-compose.production.yml up -d
```

## Testing

Run tests from your local machine:
```bash
./scripts/test-production.sh
```

This will test:
1. Homepage accessibility
2. Nginx health
3. Controller health & API
4. All 3 workers health & APIs
5. End-to-end configuration creation

## Deployment Script

To redeploy or update:
```bash
# 1. Update code locally
# 2. Archive project
tar czf project.tar.gz --exclude='*.db' --exclude='.git' .

# 3. Upload to VM
scp project.tar.gz root@103.157.116.91:~/

# 4. SSH to VM and deploy
ssh root@103.157.116.91
cd ~/config-management
tar xzf ~/project.tar.gz
./scripts/deploy-production.sh
```

## Network Configuration

### Nginx Routing
- `/` → Homepage (static HTML)
- `/admin/*` → Controller (port 8080)
- `/worker1/*` → Worker 1 (port 8082)
- `/worker2/*` → Worker 2 (port 8082)
- `/worker3/*` → Worker 3 (port 8082)
- `/nginx-health` → Nginx health check

### Internal Docker Network
All services communicate via `config-network` bridge network:
- `controller:8080` - Controller service
- `worker1:8082` - Worker 1 service
- `worker2:8082` - Worker 2 service
- `worker3:8082` - Worker 3 service

## Troubleshooting

### Agents Restarting
If agents are restarting, they may not be registered yet. Agents automatically register themselves when they successfully connect to the controller.

### Check Agent Registration
```bash
# Via API (from local machine)
curl -u admin:admin123 http://103.157.116.91/admin/api/v1/agents

# Via logs (from VM)
docker compose -f docker-compose.production.yml logs agent1
```

### Controller Not Responding
```bash
# Check controller logs
docker compose -f docker-compose.production.yml logs controller

# Restart controller
docker compose -f docker-compose.production.yml restart controller
```

### Worker Issues
```bash
# Check worker logs
docker compose -f docker-compose.production.yml logs worker1

# Restart worker
docker compose -f docker-compose.production.yml restart worker1
```

### Database Issues
The SQLite database is stored in a Docker volume:
```bash
# View database location
docker volume inspect config-management_controller-data

# Backup database
docker compose -f docker-compose.production.yml exec controller cat /data/config.db > backup.db
```

## Files Overview

- `docker-compose.production.yml` - Production Docker Compose configuration
- `nginx/nginx.conf` - Nginx reverse proxy configuration
- `scripts/deploy-production.sh` - Automated deployment script (run on VM)
- `scripts/test-production.sh` - Test script (run from local machine)

## Security Notes

⚠️ **Important for Production:**
1. Change default admin password (`admin123`)
2. Change agent passwords
3. Consider adding SSL/TLS (HTTPS)
4. Implement firewall rules (only allow ports 22, 80, 443)
5. Regular backups of controller database
6. Monitor logs for suspicious activity

## Support

For issues or questions:
1. Check service logs: `docker compose -f docker-compose.production.yml logs [service]`
2. Verify service health: `docker compose -f docker-compose.production.yml ps`
3. Run test suite: `./scripts/test-production.sh`
4. Check this documentation for common troubleshooting steps
