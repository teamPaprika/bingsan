---
title: "Docker"
weight: 1
---

# Docker Deployment

Deploy Bingsan using Docker or Docker Compose.

## Docker Compose (Development)

The fastest way to get started for development and testing.

### Prerequisites

- Docker Engine 20.10+
- Docker Compose v2.0+

### Quick Start

```bash
# Clone the repository
git clone https://github.com/kimuyb/bingsan.git
cd bingsan

# Copy configuration
cp config.example.yaml config.yaml

# Start services
docker compose -f deployments/docker/docker-compose.yml up -d
```

### docker-compose.yml

```yaml
version: '3.8'

services:
  bingsan:
    image: ghcr.io/kimuyb/bingsan:latest
    # Or build from source:
    # build:
    #   context: .
    #   dockerfile: Dockerfile
    ports:
      - "8181:8181"
    environment:
      - ICEBERG_SERVER_HOST=0.0.0.0
      - ICEBERG_SERVER_PORT=8181
      - ICEBERG_DATABASE_HOST=postgres
      - ICEBERG_DATABASE_PORT=5432
      - ICEBERG_DATABASE_USER=iceberg
      - ICEBERG_DATABASE_PASSWORD=iceberg
      - ICEBERG_DATABASE_DATABASE=iceberg_catalog
      - ICEBERG_STORAGE_TYPE=local
      - ICEBERG_STORAGE_WAREHOUSE=file:///data/warehouse
    volumes:
      - warehouse-data:/data/warehouse
    depends_on:
      postgres:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8181/health"]
      interval: 10s
      timeout: 5s
      retries: 5

  postgres:
    image: postgres:15-alpine
    environment:
      - POSTGRES_USER=iceberg
      - POSTGRES_PASSWORD=iceberg
      - POSTGRES_DB=iceberg_catalog
    ports:
      - "5432:5432"
    volumes:
      - postgres-data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U iceberg -d iceberg_catalog"]
      interval: 5s
      timeout: 5s
      retries: 5

volumes:
  warehouse-data:
  postgres-data:
```

### Managing the Stack

```bash
# View logs
docker compose -f deployments/docker/docker-compose.yml logs -f

# Stop services
docker compose -f deployments/docker/docker-compose.yml down

# Stop and remove volumes
docker compose -f deployments/docker/docker-compose.yml down -v

# Rebuild after code changes
docker compose -f deployments/docker/docker-compose.yml build --no-cache
docker compose -f deployments/docker/docker-compose.yml up -d
```

---

## Docker Compose with S3 (MinIO)

For testing with S3-compatible storage:

```yaml
version: '3.8'

services:
  bingsan:
    image: ghcr.io/kimuyb/bingsan:latest
    ports:
      - "8181:8181"
    environment:
      - ICEBERG_DATABASE_HOST=postgres
      - ICEBERG_DATABASE_PORT=5432
      - ICEBERG_DATABASE_USER=iceberg
      - ICEBERG_DATABASE_PASSWORD=iceberg
      - ICEBERG_DATABASE_DATABASE=iceberg_catalog
      - ICEBERG_STORAGE_TYPE=s3
      - ICEBERG_STORAGE_WAREHOUSE=s3://warehouse/data
      - ICEBERG_STORAGE_S3_ENDPOINT=http://minio:9000
      - ICEBERG_STORAGE_S3_ACCESS_KEY_ID=minioadmin
      - ICEBERG_STORAGE_S3_SECRET_ACCESS_KEY=minioadmin
      - ICEBERG_STORAGE_S3_BUCKET=warehouse
      - ICEBERG_STORAGE_S3_USE_PATH_STYLE=true
      - ICEBERG_STORAGE_S3_REGION=us-east-1
    depends_on:
      postgres:
        condition: service_healthy
      minio:
        condition: service_healthy

  postgres:
    image: postgres:15-alpine
    environment:
      - POSTGRES_USER=iceberg
      - POSTGRES_PASSWORD=iceberg
      - POSTGRES_DB=iceberg_catalog
    volumes:
      - postgres-data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U iceberg -d iceberg_catalog"]
      interval: 5s
      timeout: 5s
      retries: 5

  minio:
    image: minio/minio:latest
    ports:
      - "9000:9000"
      - "9001:9001"
    environment:
      - MINIO_ROOT_USER=minioadmin
      - MINIO_ROOT_PASSWORD=minioadmin
    command: server /data --console-address ":9001"
    volumes:
      - minio-data:/data
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
      interval: 5s
      timeout: 5s
      retries: 5

  minio-init:
    image: minio/mc:latest
    depends_on:
      minio:
        condition: service_healthy
    entrypoint: >
      /bin/sh -c "
      mc alias set myminio http://minio:9000 minioadmin minioadmin;
      mc mb myminio/warehouse --ignore-existing;
      exit 0;
      "

volumes:
  postgres-data:
  minio-data:
```

---

## Standalone Docker

Run Bingsan as a standalone container (requires external PostgreSQL).

### Pull and Run

```bash
# Pull the image
docker pull ghcr.io/kimuyb/bingsan:latest

# Run with environment variables
docker run -d \
  --name bingsan \
  -p 8181:8181 \
  -e ICEBERG_DATABASE_HOST=your-postgres-host \
  -e ICEBERG_DATABASE_PORT=5432 \
  -e ICEBERG_DATABASE_USER=iceberg \
  -e ICEBERG_DATABASE_PASSWORD=your-password \
  -e ICEBERG_DATABASE_DATABASE=iceberg_catalog \
  -e ICEBERG_STORAGE_TYPE=s3 \
  -e ICEBERG_STORAGE_WAREHOUSE=s3://your-bucket/warehouse \
  -e ICEBERG_STORAGE_S3_REGION=us-east-1 \
  ghcr.io/kimuyb/bingsan:latest
```

### With Configuration File

```bash
docker run -d \
  --name bingsan \
  -p 8181:8181 \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  ghcr.io/kimuyb/bingsan:latest
```

### Build from Source

```bash
# Build the image
docker build -t bingsan:local .

# Run
docker run -d \
  --name bingsan \
  -p 8181:8181 \
  -e ICEBERG_DATABASE_HOST=host.docker.internal \
  bingsan:local
```

---

## Dockerfile

```dockerfile
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /iceberg-catalog ./cmd/iceberg-catalog

# Runtime image
FROM alpine:3.19

RUN apk --no-cache add ca-certificates wget

WORKDIR /app

COPY --from=builder /iceberg-catalog /app/iceberg-catalog

EXPOSE 8181

CMD ["/app/iceberg-catalog"]
```

---

## Production Docker Deployment

### Recommendations

1. **Use specific image tags**: Don't use `latest` in production
2. **Resource limits**: Set memory and CPU limits
3. **Health checks**: Configure container health checks
4. **Logging**: Use json-file or external logging driver
5. **Secrets**: Use Docker secrets or external secret management

### Example Production Compose

```yaml
version: '3.8'

services:
  bingsan:
    image: ghcr.io/kimuyb/bingsan:v1.0.0
    deploy:
      replicas: 2
      resources:
        limits:
          memory: 512M
          cpus: '1.0'
        reservations:
          memory: 128M
          cpus: '0.25'
      restart_policy:
        condition: on-failure
        delay: 5s
        max_attempts: 3
    ports:
      - "8181:8181"
    environment:
      - ICEBERG_DATABASE_HOST=postgres
      - ICEBERG_DATABASE_PASSWORD_FILE=/run/secrets/db_password
    secrets:
      - db_password
    logging:
      driver: json-file
      options:
        max-size: "100m"
        max-file: "5"
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8181/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s

secrets:
  db_password:
    external: true
```

---

## Networking

### Bridge Network (Default)

Services communicate via container names:

```yaml
services:
  bingsan:
    environment:
      - ICEBERG_DATABASE_HOST=postgres  # Container name
```

### Host Network

For maximum performance (Linux only):

```yaml
services:
  bingsan:
    network_mode: host
    environment:
      - ICEBERG_DATABASE_HOST=localhost
      - ICEBERG_SERVER_PORT=8181
```

### External Network

Connect to existing network:

```yaml
networks:
  default:
    external: true
    name: my-network
```

---

## Troubleshooting

### Container Won't Start

```bash
# Check logs
docker logs bingsan

# Check if port is in use
lsof -i :8181
```

### Database Connection Failed

```bash
# Test PostgreSQL connectivity
docker exec bingsan nc -zv postgres 5432

# Check environment variables
docker exec bingsan env | grep ICEBERG_DATABASE
```

### Health Check Failing

```bash
# Test health endpoint
docker exec bingsan wget -q -O- http://localhost:8181/health

# Test readiness endpoint
docker exec bingsan wget -q -O- http://localhost:8181/ready
```
