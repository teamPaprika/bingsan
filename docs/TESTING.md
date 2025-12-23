# Bingsan Testing Guide

## Overview

Bingsan uses Docker Compose-based test runners for integration testing. Tests are organized into phases, each testing specific functionality.

## Prerequisites

- Docker with remote context configured (`chuncom`)
- Running Bingsan stack (`docker-compose.yml`)

## Running the Stack

```bash
# Start the Bingsan stack
docker --context chuncom compose -f deployments/docker/docker-compose.yml up -d

# Check status
docker --context chuncom compose -f deployments/docker/docker-compose.yml ps

# View logs
docker --context chuncom logs iceberg-catalog --tail 50
```

## Test Phases

### Phase 1: Core CRUD Operations

Tests namespace CRUD operations.

```bash
docker --context chuncom compose -f deployments/docker/test-phase1.yaml up --abort-on-container-exit
```

**Endpoints tested:**
- `POST /v1/namespaces` - Create namespace
- `GET /v1/namespaces` - List namespaces
- `GET /v1/namespaces/{namespace}` - Get namespace
- `PUT /v1/namespaces/{namespace}/properties` - Update properties
- `DELETE /v1/namespaces/{namespace}` - Delete namespace

### Phase 2: Table Operations

Tests table CRUD and commit operations.

```bash
docker --context chuncom compose -f deployments/docker/test-phase2.yaml up --abort-on-container-exit
```

**Endpoints tested:**
- `POST /v1/namespaces/{namespace}/tables` - Create table
- `GET /v1/namespaces/{namespace}/tables` - List tables
- `GET /v1/namespaces/{namespace}/tables/{table}` - Get table
- `POST /v1/namespaces/{namespace}/tables/{table}` - Commit table
- `DELETE /v1/namespaces/{namespace}/tables/{table}` - Drop table
- `POST /v1/namespaces/{namespace}/tables/{table}/rename` - Rename table

### Phase 3: View Operations

Tests view CRUD operations.

```bash
docker --context chuncom compose -f deployments/docker/test-phase3.yaml up --abort-on-container-exit
```

**Endpoints tested:**
- `POST /v1/namespaces/{namespace}/views` - Create view
- `GET /v1/namespaces/{namespace}/views` - List views
- `GET /v1/namespaces/{namespace}/views/{view}` - Get view
- `POST /v1/namespaces/{namespace}/views/{view}` - Replace view
- `DELETE /v1/namespaces/{namespace}/views/{view}` - Drop view
- `POST /v1/namespaces/{namespace}/views/{view}/rename` - Rename view

### Phase 4: Scan Planning

Tests server-side scan planning (Iceberg REST spec compliant).

```bash
docker --context chuncom compose -f deployments/docker/test-phase4.yaml up --abort-on-container-exit
```

**Endpoints tested:**
- `POST /v1/namespaces/{namespace}/tables/{table}/plan` - Submit scan plan
- `GET /v1/namespaces/{namespace}/tables/{table}/plan/{plan-id}` - Get plan status
- `DELETE /v1/namespaces/{namespace}/tables/{table}/plan/{plan-id}` - Cancel plan
- `POST /v1/namespaces/{namespace}/tables/{table}/tasks` - Fetch scan tasks

### Phase 5: Prometheus Metrics & WebSocket Events

Tests observability features.

```bash
docker --context chuncom compose -f deployments/docker/test-phase5.yaml up --abort-on-container-exit
```

**Endpoints tested:**
- `GET /metrics` - Prometheus metrics endpoint
- `GET /v1/events/stream` - WebSocket event stream

**Metrics verified:**
- `iceberg_namespaces_total`
- `iceberg_tables_total{namespace}`
- `iceberg_db_connections_*`
- HTTP request metrics

## Running All Tests

```bash
# Run all test phases sequentially
for phase in 1 2 3 4 5; do
  echo "=== Running Phase $phase ==="
  docker --context chuncom compose -f deployments/docker/test-phase$phase.yaml up --abort-on-container-exit
  echo ""
done
```

## Manual Testing

### Health Check
```bash
curl http://localhost:8181/health
curl http://localhost:8181/ready
```

### Prometheus Metrics
```bash
curl http://localhost:8181/metrics | grep iceberg_
```

### WebSocket Events (requires wscat)
```bash
# Install wscat if not available
npm install -g wscat

# Connect to event stream (with auth token)
wscat -c "ws://localhost:8181/v1/events/stream?token=YOUR_API_KEY"

# Connect with namespace filter
wscat -c "ws://localhost:8181/v1/events/stream?token=YOUR_API_KEY&namespace=my_ns"
```

### API Key Management
```bash
# Create API key
curl -X POST http://localhost:8181/v1/api-keys \
  -H "Content-Type: application/json" \
  -d '{"name": "test-key", "description": "Test API key"}'

# Use API key for authenticated requests
curl -H "Authorization: Bearer YOUR_API_KEY" http://localhost:8181/v1/namespaces
```

## Troubleshooting

### Container Logs
```bash
docker --context chuncom logs iceberg-catalog --tail 100 -f
docker --context chuncom logs iceberg-postgres --tail 50
```

### Reset Database
```bash
docker --context chuncom compose -f deployments/docker/docker-compose.yml down -v
docker --context chuncom compose -f deployments/docker/docker-compose.yml up -d
```

### Rebuild After Code Changes
```bash
docker --context chuncom compose -f deployments/docker/docker-compose.yml build --no-cache
docker --context chuncom compose -f deployments/docker/docker-compose.yml down && \
docker --context chuncom compose -f deployments/docker/docker-compose.yml up -d
```
