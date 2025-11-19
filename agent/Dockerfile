# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY agent/go.mod agent/go.sum ./agent/
COPY pkg/ ./pkg/

WORKDIR /app/agent
RUN go mod download

COPY agent/ .

RUN CGO_ENABLED=0 go build -o agent ./cmd

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/agent/agent .

# Create directory for cache
RUN mkdir -p /app/cache

# Set environment variables with defaults
ENV CONTROLLER_URL=http://controller:8080
ENV CONTROLLER_USER=agent
ENV CONTROLLER_PASSWORD=agent123
ENV WORKER_URL=http://worker:8082
ENV CACHE_FILE=/app/cache/agent_config.cache

# Run the application
CMD ["./agent"]
