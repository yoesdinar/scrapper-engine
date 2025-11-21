# Build stage
FROM --platform=$BUILDPLATFORM golang:1.21-bookworm AS builder

ARG TARGETOS
ARG TARGETARCH

RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    libc6-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY controller/go.mod controller/go.sum ./controller/
COPY pkg/ ./pkg/

WORKDIR /app/controller
RUN go mod download

COPY controller/ .

RUN CGO_ENABLED=1 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -a -installsuffix cgo -o controller ./cmd

# Runtime stage
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/controller/controller .

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
