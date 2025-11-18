# Distributed Configuration Management System

A distributed configuration management system implemented in Go, featuring a Controller service for centralized config management, Agent services that poll for updates, and Worker services that execute tasks based on received configuration.

## Architecture

```
┌─────────────────────────────────────────┐
│         Controller (Port 8080)          │
│  ┌──────────────────────────────────┐   │
│  │   REST API + SQLite Database     │   │
│  └──────────────────────────────────┘   │
└────────────────┬────────────────────────┘
                 │ Polling (HTTP)
         ┌───────┴────────┬────────────┐
         │                │            │
    ┌────▼─────┐    ┌────▼─────┐  ┌───▼──────┐
    │  Agent 1 │    │  Agent 2 │  │  Agent N │
    └────┬─────┘    └────┬─────┘  └───┬──────┘
         │               │            │
    ┌────▼─────┐    ┌────▼─────┐  ┌───▼──────┐
    │ Worker 1 │    │ Worker 2 │  │ Worker N │
    │(Port 8082│    │          │  │          │
    └──────────┘    └──────────┘  └──────────┘
```

### Components

- **Controller**: Central configuration management service with REST API and SQLite database
- **Agent**: Bridge service that polls Controller for config changes and forwards to Worker
- **Worker**: HTTP service that executes tasks (proxies HTTP requests) based on received configuration

## Features

- ✅ **Centralized Configuration Management** - Single source of truth for all agents
- ✅ **Version-based Change Detection** - ETag headers for efficient polling
- ✅ **Exponential Backoff** - Resilient agent polling with automatic retry
- ✅ **Configuration Persistence** - SQLite database with history tracking
- ✅ **Configuration Caching** - Agents cache config locally for offline operation
- ✅ **Dynamic Poll Interval** - Controller can adjust agent polling frequency
- ✅ **Basic Authentication** - Separate credentials for agents and admins
- ✅ **Swagger Documentation** - Auto-generated API docs
- ✅ **Graceful Shutdown** - Proper cleanup on SIGTERM/SIGINT
- ✅ **Structured Logging** - Comprehensive logging with logrus

## Project Structure

```
coding-test/
├── controller/          # Controller service
│   ├── cmd/            # Entry point
│   ├── internal/       # Private code
│   │   ├── api/        # REST handlers & router
│   │   ├── database/   # SQLite operations
│   │   ├── models/     # Data structures
│   │   └── service/    # Business logic
│   ├── docs/           # Swagger generated docs
│   └── test/           # Tests
├── agent/              # Agent service
│   ├── cmd/            # Entry point
│   ├── internal/       # Private code
│   │   ├── config/     # Configuration loader
│   │   ├── poller/     # Controller polling logic
│   │   ├── backoff/    # Exponential backoff
│   │   └── worker/     # Worker manager
│   └── test/           # Tests
├── worker/             # Worker service
│   ├── cmd/            # Entry point
│   ├── internal/       # Private code
│   │   ├── api/        # REST handlers & router
│   │   ├── config/     # Thread-safe config storage
│   │   └── proxy/      # HTTP proxy logic
│   ├── docs/           # Swagger generated docs
│   └── test/           # Tests
├── pkg/                # Shared code
│   ├── models/         # Common data structures
│   ├── auth/           # Authentication utilities
│   └── logger/         # Logging utilities
├── docker/             # Docker configurations
├── docs/               # Documentation
└── scripts/            # Utility scripts
```

## Prerequisites

- Go 1.21 or higher
- Make (optional, for using Makefile)
- Docker & Docker Compose (for containerized deployment)
- swag CLI (for generating Swagger docs): `go install github.com/swaggo/swag/cmd/swag@latest`

## Quick Start

### 1. Install Dependencies

```bash
# Install dependencies for all services
make deps

# Or manually:
cd pkg && go mod tidy
cd ../controller && go mod tidy
cd ../agent && go mod tidy
cd ../worker && go mod tidy
```

### 2. Generate Swagger Documentation

```bash
make swagger

# Or manually:
cd controller && swag init -g cmd/main.go -o docs
cd ../worker && swag init -g cmd/main.go -o docs
```

### 3. Build Services

```bash
make build

# Or manually:
cd controller && go build -o controller ./cmd
cd ../agent && go build -o agent ./cmd
cd ../worker && go build -o worker ./cmd
```

### 4. Run Services

**Terminal 1 - Controller:**
```bash
cd controller
export DB_PATH=./controller.db
export AGENT_USERNAME=agent
export AGENT_PASSWORD=secret123
export ADMIN_USERNAME=admin
export ADMIN_PASSWORD=admin123
export PORT=8080
./controller
```

**Terminal 2 - Worker:**
```bash
cd worker
export PORT=8082
./worker
```

**Terminal 3 - Agent:**
```bash
cd agent
export CONTROLLER_URL=http://localhost:8080
export CONTROLLER_USERNAME=agent
export CONTROLLER_PASSWORD=secret123
export WORKER_URL=http://localhost:8082
./agent
```

## API Documentation

### Controller API

Base URL: `http://localhost:8080`

#### POST /api/v1/register
Register a new agent.

**Authentication:** Basic Auth (agent credentials)

**Request:**
```json
{
  "hostname": "agent-1",
  "metadata": "region=us-west"
}
```

**Response:**
```json
{
  "agent_id": "uuid-here",
  "poll_url": "/api/v1/config",
  "poll_interval_seconds": 30
}
```

#### GET /api/v1/config
Get current configuration.

**Authentication:** Basic Auth (agent credentials)

**Response:**
```json
{
  "version": 1,
  "data": {
    "url": "https://ip.me"
  },
  "poll_interval_seconds": 30
}
```

**Headers:**
- `ETag`: Configuration version

#### POST /api/v1/config
Update configuration (admin only).

**Authentication:** Basic Auth (admin credentials)

**Request:**
```json
{
  "url": "https://api.github.com"
}
```

**Query Parameters:**
- `poll_interval` (optional): Poll interval in seconds

**Response:**
```json
{
  "message": "Configuration updated successfully",
  "version": 2
}
```

#### GET /api/v1/agents
List all registered agents (admin only).

**Authentication:** Basic Auth (admin credentials)

**Response:**
```json
[
  {
    "id": "uuid-here",
    "registered_at": "2024-01-01T00:00:00Z",
    "last_poll": "2024-01-01T00:05:00Z",
    "metadata": "region=us-west"
  }
]
```

### Worker API

Base URL: `http://localhost:8082`

#### POST /config
Update worker configuration (called by agent).

**Request:**
```json
{
  "url": "https://ip.me"
}
```

**Response:**
```json
{
  "message": "Configuration updated"
}
```

#### GET /hit
Execute configured task (proxy HTTP GET request).

**Response:** Returns the response from the configured URL.

**Example:**
```bash
curl http://localhost:8082/hit
# Returns: 1.2.3.4 (your public IP if configured URL is https://ip.me)
```

## Testing

### Manual Testing Flow

1. **Start all services** (Controller, Worker, Agent)

2. **Update configuration:**
```bash
curl -X POST http://localhost:8080/api/v1/config \
  -u admin:admin123 \
  -H "Content-Type: application/json" \
  -d '{"url":"https://ip.me"}'
```

3. **Wait for agent to poll** (~30 seconds by default)

4. **Test worker:**
```bash
curl http://localhost:8082/hit
# Should return your public IP
```

5. **Change configuration:**
```bash
curl -X POST http://localhost:8080/api/v1/config \
  -u admin:admin123 \
  -H "Content-Type: application/json" \
  -d '{"url":"https://api.github.com"}'
```

6. **Test again:**
```bash
curl http://localhost:8082/hit
# Should now return GitHub API response
```

### Run Unit Tests

```bash
make test

# Or manually:
cd pkg && go test ./...
cd ../controller && go test ./...
cd ../agent && go test ./...
cd ../worker && go test ./...
```

## Configuration

### Controller Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_PATH` | `./controller.db` | SQLite database file path |
| `PORT` | `8080` | HTTP server port |
| `AGENT_USERNAME` | `agent` | Agent authentication username |
| `AGENT_PASSWORD` | `secret123` | Agent authentication password |
| `ADMIN_USERNAME` | `admin` | Admin authentication username |
| `ADMIN_PASSWORD` | `admin123` | Admin authentication password |
| `DEFAULT_POLL_INTERVAL` | `30` | Default poll interval in seconds |
| `LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |

### Agent Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CONTROLLER_URL` | `http://localhost:8080` | Controller service URL |
| `CONTROLLER_USERNAME` | `agent` | Controller authentication username |
| `CONTROLLER_PASSWORD` | `secret123` | Controller authentication password |
| `WORKER_URL` | `http://localhost:8082` | Worker service URL |
| `CACHE_FILE` | `./agent_config.cache` | Config cache file path |
| `LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |

### Worker Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8082` | HTTP server port |
| `LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |

## Docker Deployment

### Build Docker Images

```bash
make docker-build
```

### Start Services with Docker Compose

```bash
make docker-up
```

### View Logs

```bash
make docker-logs
```

### Stop Services

```bash
make docker-down
```

## Development

### Adding New Features

1. **Create a new branch:**
```bash
git checkout -b feature/your-feature
```

2. **Make changes and test:**
```bash
make test
make build
```

3. **Format code:**
```bash
make fmt
```

4. **Update Swagger docs:**
```bash
make swagger
```

## Architecture Decisions

### Phase 1: Polling Implementation (Current)
- **Simplicity**: Easy to implement and understand
- **Reliability**: Works across all network environments
- **Scalability**: Suitable for up to ~1000 agents

### Future Phases
- **Phase 2**: Redis Pub/Sub for better efficiency (10x less network traffic)
- **Phase 3**: NATS for massive scale (1M+ agents)
- **Phase 4**: Kafka for audit trail and replay capabilities

## Troubleshooting

### Agent Can't Connect to Controller

1. Check controller is running: `curl http://localhost:8080/health`
2. Verify credentials in agent environment variables
3. Check firewall rules if running on different machines

### Worker Not Receiving Config

1. Check agent logs for errors
2. Verify worker is running: `curl http://localhost:8082/health`
3. Ensure `WORKER_URL` in agent matches worker's actual address

### Configuration Not Updating

1. Check if configuration version increased: `curl -u agent:secret123 http://localhost:8080/api/v1/config`
2. Verify agent is polling successfully (check agent logs)
3. Ensure poll interval hasn't been set too high

## License

Apache 2.0

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## Contact

For issues or questions, please open an issue on GitHub.
