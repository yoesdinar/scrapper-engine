.PHONY: help build test clean run-controller run-agent run-worker docker-build docker-up docker-down

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build all services
	@echo "Building controller..."
	cd controller && go build -o controller ./cmd
	@echo "Building agent..."
	cd agent && go build -o agent ./cmd
	@echo "Building worker..."
	cd worker && go build -o worker ./cmd

build-images: ## Build Docker images for local development
	./scripts/build-images.sh

test: ## Run tests for all services
	@echo "Testing pkg..."
	cd pkg && go test ./...
	@echo "Testing controller..."
	cd controller && go test ./...
	@echo "Testing agent..."
	cd agent && go test ./...
	@echo "Testing worker..."
	cd worker && go test ./...

clean: ## Clean build artifacts
	rm -f controller/controller
	rm -f agent/agent
	rm -f worker/worker
	rm -f controller/*.db*
	rm -f agent/*.cache

deps: ## Download dependencies
	cd pkg && go mod download
	cd controller && go mod download
	cd agent && go mod download
	cd worker && go mod download

swagger: ## Generate Swagger docs
	swag init -g controller/cmd/main.go -o controller/docs --parseDependency --parseInternal
	swag init -g worker/cmd/main.go -o worker/docs --parseDependency --parseInternal

run-controller: ## Run controller service
	cd controller && go run ./cmd

run-agent: ## Run agent service
	cd agent && go run ./cmd

run-worker: ## Run worker service
	cd worker && go run ./cmd

run-controller-redis: ## Run controller service with Redis distribution strategy
	cd controller && \
	DISTRIBUTION_STRATEGY=REDIS \
	REDIS_ADDRESS=localhost:6379 \
	REDIS_PASSWORD= \
	REDIS_DB=0 \
	DB_PATH=./controller.db \
	PORT=8080 \
	AGENT_USERNAME=agent \
	AGENT_PASSWORD=secret123 \
	ADMIN_USERNAME=admin \
	ADMIN_PASSWORD=admin123 \
	go run ./cmd

run-controller-nats: ## Run controller service with NATS distribution strategy
	cd controller && \
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
	go run ./cmd

run-agent-redis: ## Run agent service with Redis distribution strategy
	cd agent && \
	DISTRIBUTION_STRATEGY=REDIS \
	REDIS_ADDRESS=localhost:6379 \
	REDIS_PASSWORD= \
	REDIS_DB=0 \
	CONTROLLER_URL=http://localhost:8080 \
	CONTROLLER_USERNAME=agent \
	CONTROLLER_PASSWORD=secret123 \
	WORKER_URL=http://localhost:8082 \
	CACHE_FILE=./agent_config.cache \
	go run ./cmd

run-agent-nats: ## Run agent service with NATS distribution strategy
	cd agent && \
	DISTRIBUTION_STRATEGY=NATS \
	NATS_URL=nats://localhost:4222 \
	NATS_SUBJECT=config.worker.update \
	NATS_QUEUE_GROUP=config-workers \
	CONTROLLER_URL=http://localhost:8080 \
	CONTROLLER_USERNAME=agent \
	CONTROLLER_PASSWORD=secret123 \
	WORKER_URL=http://localhost:8082 \
	CACHE_FILE=./agent_config.cache \
	go run ./cmd

run-agent-poller: ## Run agent service with HTTP polling distribution strategy
	cd agent && \
	DISTRIBUTION_STRATEGY=POLLER \
	CONTROLLER_URL=http://localhost:8080 \
	CONTROLLER_USERNAME=agent \
	CONTROLLER_PASSWORD=secret123 \
	WORKER_URL=http://localhost:8082 \
	CACHE_FILE=./agent_config.cache \
	go run ./cmd

run-worker-redis: ## Run worker service (Redis not needed for worker)
	cd worker && \
	PORT=8082 \
	go run ./cmd

start-redis: ## Start Redis server locally (requires Redis installation)
	@echo "Starting Redis server..."
	@if command -v redis-server >/dev/null 2>&1; then \
		redis-server --daemonize yes --port 6379; \
		echo "✅ Redis started on localhost:6379"; \
	else \
		echo "❌ Redis not installed. Install with: brew install redis (macOS) or apt-get install redis-server (Ubuntu)"; \
		exit 1; \
	fi

stop-redis: ## Stop Redis server
	@echo "Stopping Redis server..."
	@redis-cli shutdown || echo "Redis was not running"

start-nats: ## Start NATS server locally (requires NATS installation)
	@echo "Starting NATS server..."
	@if command -v nats-server >/dev/null 2>&1; then \
		nats-server --port 4222 --http_port 8222 --jetstream --store_dir ./nats-data & \
		echo "✅ NATS started on localhost:4222 (HTTP monitoring on :8222)"; \
	else \
		echo "❌ NATS not installed. Install with: brew install nats-server (macOS) or download from https://nats.io/download"; \
		exit 1; \
	fi

stop-nats: ## Stop NATS server
	@echo "Stopping NATS server..."
	@pkill nats-server || echo "NATS was not running"

test-redis-local: ## Test Redis strategy with local services (automated)
	./scripts/test-local-redis.sh

setup-redis-local: ## Setup Redis server for local testing
	./scripts/setup-local-redis.sh

test-nats-local: ## Test NATS strategy with local services (automated)
	./scripts/test-local-nats.sh

test-nats-docker: ## Test NATS strategy with Docker services
	./scripts/test-strategy-nats.sh

setup-nats-local: ## Setup NATS server for local testing
	@echo "Setting up NATS for local testing..."
	@make start-nats
	@echo "NATS server ready for testing"

docker-build: ## Build Docker images
	docker-compose -f docker/docker-compose.controller.yml build
	docker-compose -f docker/docker-compose.agent-worker.yml build

docker-up: ## Start all services with Docker
	docker-compose -f docker/docker-compose.controller.yml up -d
	docker-compose -f docker/docker-compose.agent-worker.yml up -d

docker-up-strategy: ## Start services with strategy pattern testing
	docker-compose -f docker-compose.strategy-test.yml up -d

docker-down: ## Stop all services
	docker-compose -f docker/docker-compose.controller.yml down
	docker-compose -f docker/docker-compose.agent-worker.yml down

docker-down-strategy: ## Stop strategy test services
	docker-compose -f docker-compose.strategy-test.yml down

test-strategy: ## Test strategy pattern architecture
	./scripts/test-strategy-pattern.sh

docker-logs: ## Show Docker logs
	docker-compose -f docker/docker-compose.controller.yml logs -f

lint: ## Run linter
	golangci-lint run ./...

fmt: ## Format code
	go fmt ./...
