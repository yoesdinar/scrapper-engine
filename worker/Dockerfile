# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY worker/go.mod worker/go.sum ./worker/
COPY pkg/ ./pkg/

WORKDIR /app/worker
RUN go mod download

COPY worker/ .

RUN CGO_ENABLED=0 go build -o worker ./cmd

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /app

COPY --from=builder /app/worker/worker .

# Expose port
EXPOSE 8082

# Set environment variables with defaults
ENV WORKER_PORT=8082

# Run the application
CMD ["./worker"]
