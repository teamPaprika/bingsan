---
title: "Installation"
weight: 2
---

# Installation

This guide covers all installation methods for Bingsan.

## Method 1: Docker Compose (Recommended)

The easiest way to run Bingsan for development and testing.

### Prerequisites

- Docker Engine 20.10+
- Docker Compose v2.0+

### Steps

```bash
# Clone the repository
git clone https://github.com/kimuyb/bingsan.git
cd bingsan

# Copy configuration
cp config.example.yaml config.yaml

# Start services
docker compose -f deployments/docker/docker-compose.yml up -d
```

## Method 2: Build from Source

Build and run Bingsan directly on your machine.

### Prerequisites

- Go 1.25 or later
- PostgreSQL 15+ (running and accessible)

### Steps

```bash
# Clone the repository
git clone https://github.com/kimuyb/bingsan.git
cd bingsan

# Download dependencies
go mod download

# Build the binary
go build -o bin/iceberg-catalog ./cmd/iceberg-catalog

# Configure
cp config.example.yaml config.yaml
# Edit config.yaml with your PostgreSQL connection details

# Run
./bin/iceberg-catalog
```

### Build Options

Build with version information:

```bash
go build -ldflags "-X main.version=1.0.0 -X main.commit=$(git rev-parse HEAD)" \
  -o bin/iceberg-catalog ./cmd/iceberg-catalog
```

Build for different platforms:

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o bin/iceberg-catalog-linux-amd64 ./cmd/iceberg-catalog

# Linux ARM64
GOOS=linux GOARCH=arm64 go build -o bin/iceberg-catalog-linux-arm64 ./cmd/iceberg-catalog

# macOS AMD64
GOOS=darwin GOARCH=amd64 go build -o bin/iceberg-catalog-darwin-amd64 ./cmd/iceberg-catalog

# macOS ARM64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o bin/iceberg-catalog-darwin-arm64 ./cmd/iceberg-catalog
```

## Method 3: Docker Image

Pull and run the pre-built Docker image.

```bash
# Pull the image
docker pull ghcr.io/kimuyb/bingsan:latest

# Run with environment variables
docker run -d \
  --name bingsan \
  -p 8181:8181 \
  -e ICEBERG_DATABASE_HOST=postgres \
  -e ICEBERG_DATABASE_PORT=5432 \
  -e ICEBERG_DATABASE_USER=iceberg \
  -e ICEBERG_DATABASE_PASSWORD=iceberg \
  -e ICEBERG_DATABASE_DATABASE=iceberg_catalog \
  ghcr.io/kimuyb/bingsan:latest
```

## Method 4: Kubernetes

Deploy to Kubernetes using Helm or raw manifests.

See the [Kubernetes Deployment Guide]({{< relref "/docs/deployment/kubernetes" >}}) for detailed instructions.

## Database Setup

Bingsan requires PostgreSQL 15 or later.

### Create Database

```sql
CREATE DATABASE iceberg_catalog;
CREATE USER iceberg WITH PASSWORD 'iceberg';
GRANT ALL PRIVILEGES ON DATABASE iceberg_catalog TO iceberg;
```

### Automatic Migrations

Bingsan automatically runs database migrations on startup. No manual schema setup is required.

## Verifying Installation

After starting Bingsan, verify it's running correctly:

```bash
# Health check
curl http://localhost:8181/health

# Readiness check (includes DB)
curl http://localhost:8181/ready

# Get configuration
curl http://localhost:8181/v1/config
```

## Environment Variables

All configuration options can be set via environment variables using the `ICEBERG_` prefix:

| Environment Variable | Config Path | Description |
|---------------------|-------------|-------------|
| `ICEBERG_SERVER_HOST` | `server.host` | Bind address |
| `ICEBERG_SERVER_PORT` | `server.port` | Listen port |
| `ICEBERG_DATABASE_HOST` | `database.host` | PostgreSQL host |
| `ICEBERG_DATABASE_PORT` | `database.port` | PostgreSQL port |
| `ICEBERG_DATABASE_USER` | `database.user` | Database user |
| `ICEBERG_DATABASE_PASSWORD` | `database.password` | Database password |
| `ICEBERG_DATABASE_DATABASE` | `database.database` | Database name |
| `ICEBERG_AUTH_ENABLED` | `auth.enabled` | Enable authentication |

See the [Configuration Guide]({{< relref "/docs/configuration" >}}) for all options.
