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
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o controller ./cmd

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates sqlite

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /build/controller .

# Create directory for database
RUN mkdir -p /app/data

# Expose port
EXPOSE 8080

# Set environment variables with defaults
ENV CONTROLLER_PORT=8080
ENV CONTROLLER_DB_PATH=/app/data/controller.db
ENV CONTROLLER_AGENT_USER=agent
ENV CONTROLLER_AGENT_PASS=agent123
ENV CONTROLLER_ADMIN_USER=admin
ENV CONTROLLER_ADMIN_PASS=admin123

# Run the application
CMD ["./controller"]
