# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.Version=$(git describe --tags --always --dirty 2>/dev/null || echo dev)" \
    -o /app/iceberg-catalog \
    ./cmd/iceberg-catalog

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 iceberg && \
    adduser -u 1000 -G iceberg -s /bin/sh -D iceberg

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/iceberg-catalog /app/iceberg-catalog

# Copy migrations (embedded in binary, but keep for reference)
COPY --from=builder /app/internal/db/migrations /app/migrations

# Set ownership
RUN chown -R iceberg:iceberg /app

# Switch to non-root user
USER iceberg

# Expose port
EXPOSE 8181

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8181/health || exit 1

# Run
ENTRYPOINT ["/app/iceberg-catalog"]
