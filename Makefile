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

docker-build: ## Build Docker images
	docker-compose -f docker/docker-compose.controller.yml build
	docker-compose -f docker/docker-compose.agent-worker.yml build

docker-up: ## Start all services with Docker
	docker-compose -f docker/docker-compose.controller.yml up -d
	docker-compose -f docker/docker-compose.agent-worker.yml up -d

docker-down: ## Stop all services
	docker-compose -f docker/docker-compose.controller.yml down
	docker-compose -f docker/docker-compose.agent-worker.yml down

docker-logs: ## Show Docker logs
	docker-compose -f docker/docker-compose.controller.yml logs -f

lint: ## Run linter
	golangci-lint run ./...

fmt: ## Format code
	go fmt ./...
