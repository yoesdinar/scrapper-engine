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
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o worker ./cmd

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /build/worker .

# Expose port
EXPOSE 8082

# Set environment variables with defaults
ENV WORKER_PORT=8082

# Run the application
CMD ["./worker"]
