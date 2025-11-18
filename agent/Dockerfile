# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
COPY ../pkg/go.mod ../pkg/go.sum ../pkg/

# Download dependencies
RUN go mod download

# Copy source code
COPY . .
COPY ../pkg ../pkg

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o agent ./cmd

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /build/agent .

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
