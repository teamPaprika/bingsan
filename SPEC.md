# Bingsan - Iceberg REST Catalog

## Purpose

A production-grade Apache Iceberg REST Catalog implementation in Go, designed for high concurrency and cloud-native deployments.

## Goals

1. **Fully Functioning REST Catalog** - Complete implementation of the [Apache Iceberg REST Catalog OpenAPI Spec](https://github.com/apache/iceberg/blob/main/open-api/rest-catalog-open-api.yaml)

2. **PostgreSQL Backend** - Metadata storage in PostgreSQL for reliability and ACID compliance

3. **Cloud Storage Compatible** - Support for S3 and GCS as data lake storage backends

4. **Go Concurrency** - Leverage goroutines and channels for high-performance concurrent operations (this is what makes Bingsan special)

5. **Flexible Deployment** - Run as single node or scale horizontally across multiple nodes in Kubernetes

6. **Production Compatible** - Drop-in replacement for existing Java REST catalogs (Polaris, Gravitino, etc.)

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Clients                                │
│         (Spark, Trino, Flink, PyIceberg, etc.)             │
└─────────────────────────┬───────────────────────────────────┘
                          │ REST API
┌─────────────────────────▼───────────────────────────────────┐
│                    Bingsan Cluster                          │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐                     │
│  │ Node 1  │  │ Node 2  │  │ Node N  │  (Kubernetes Pods)  │
│  │  :8181  │  │  :8181  │  │  :8181  │                     │
│  └────┬────┘  └────┬────┘  └────┬────┘                     │
│       │            │            │                           │
│       └────────────┼────────────┘                           │
│                    │ Distributed Locking                    │
└────────────────────┼────────────────────────────────────────┘
                     │
        ┌────────────┴────────────┐
        │                         │
┌───────▼───────┐       ┌────────▼────────┐
│  PostgreSQL   │       │   S3 / GCS      │
│  (Metadata)   │       │   (Data Lake)   │
└───────────────┘       └─────────────────┘
```

## Requirements

### Functional Requirements

#### Phase 1: Core Catalog (Namespace & Table CRUD)
- [ ] `GET /v1/config` - Catalog configuration
- [x] `GET /v1/namespaces` - List namespaces
- [x] `POST /v1/namespaces` - Create namespace
- [x] `GET /v1/namespaces/{namespace}` - Load namespace
- [x] `DELETE /v1/namespaces/{namespace}` - Drop namespace
- [x] `POST /v1/namespaces/{namespace}/properties` - Update namespace properties
- [x] `HEAD /v1/namespaces/{namespace}` - Check namespace exists
- [x] `GET /v1/namespaces/{namespace}/tables` - List tables
- [x] `POST /v1/namespaces/{namespace}/tables` - Create table
- [x] `GET /v1/namespaces/{namespace}/tables/{table}` - Load table
- [x] `DELETE /v1/namespaces/{namespace}/tables/{table}` - Drop table
- [x] `HEAD /v1/namespaces/{namespace}/tables/{table}` - Check table exists

#### Phase 2: Table Operations
- [x] `POST /v1/namespaces/{namespace}/tables/{table}` - Commit table update
- [x] `POST /v1/namespaces/{namespace}/register` - Register table
- [x] `POST /v1/namespaces/{namespace}/tables/{table}/rename` - Rename table (Note: spec uses `POST /v1/tables/rename`)
- [x] `POST /v1/namespaces/{namespace}/tables/{table}/metrics` - Report metrics

#### Phase 3: Views
- [x] `GET /v1/namespaces/{namespace}/views` - List views
- [x] `POST /v1/namespaces/{namespace}/views` - Create view
- [x] `GET /v1/namespaces/{namespace}/views/{view}` - Load view
- [x] `DELETE /v1/namespaces/{namespace}/views/{view}` - Drop view
- [x] `HEAD /v1/namespaces/{namespace}/views/{view}` - Check view exists
- [x] `POST /v1/namespaces/{namespace}/views/{view}` - Replace view
- [x] `POST /v1/views/rename` - Rename view

#### Phase 4: Server-Side Scan Planning
- [x] `POST /v1/namespaces/{namespace}/tables/{table}/plan` - Submit scan plan
- [x] `GET /v1/namespaces/{namespace}/tables/{table}/plan/{plan-id}` - Get scan plan
- [x] `DELETE /v1/namespaces/{namespace}/tables/{table}/plan/{plan-id}` - Cancel scan plan
- [x] `POST /v1/namespaces/{namespace}/tables/{table}/tasks` - Fetch plan tasks

#### Phase 5: Multi-Table Transactions
- [x] `POST /v1/transactions/commit` - Atomic multi-table commit

#### Phase 6: Authentication & Authorization
- [ ] `POST /v1/oauth/tokens` - OAuth2 token exchange
- [ ] `GET /v1/namespaces/{namespace}/tables/{table}/credentials` - Vended credentials (S3/GCS)
- [ ] API Key authentication
- [ ] Namespace-level access control

### Non-Functional Requirements

#### Performance
- [ ] Handle 10,000+ concurrent connections per node
- [ ] Sub-10ms latency for metadata operations (p99)
- [ ] Efficient connection pooling to PostgreSQL

#### Concurrency (Go-specific advantages)
- [ ] Goroutine-per-request model
- [ ] Non-blocking I/O for all storage operations
- [ ] Concurrent scan planning with worker pools
- [ ] Lock-free read paths where possible

#### Scalability
- [ ] Horizontal scaling via Kubernetes
- [ ] Distributed locking for multi-node consistency
- [ ] Stateless nodes (all state in PostgreSQL)
- [ ] Leader election for background tasks (lock cleanup, etc.)

#### Reliability
- [ ] Graceful shutdown with request draining
- [ ] Health checks (`/health/live`, `/health/ready`)
- [ ] Automatic PostgreSQL reconnection
- [ ] Configurable request timeouts

#### Observability
- [x] Structured logging (slog)
- [x] Prometheus metrics endpoint (`/metrics`)
- [x] WebSocket event streaming (`/v1/events/stream`)
- [ ] OpenTelemetry tracing support
- [ ] Request ID propagation

#### Security
- [ ] TLS termination support
- [ ] OAuth2/OIDC integration
- [ ] API key rotation
- [ ] Audit logging

### Storage Backend Support

#### PostgreSQL (Metadata)
- [x] Connection pooling (pgxpool)
- [x] Automatic migrations
- [x] Distributed locking
- [x] Optimistic concurrency control

#### S3 Compatible
- [ ] AWS S3
- [ ] MinIO
- [ ] Cloudflare R2
- [ ] Vended credentials (STS AssumeRole)

#### GCS
- [ ] Google Cloud Storage
- [ ] Vended credentials (Service Account impersonation)

## Compatibility

Must be compatible with:
- Apache Spark (via Iceberg Spark runtime)
- Trino/Presto
- Apache Flink
- PyIceberg
- Dremio
- StarRocks

Should match behavior of:
- [Apache Polaris](https://github.com/apache/polaris) (Snowflake)
- [Gravitino](https://github.com/datastrato/gravitino) (Datastrato)
- [Nessie](https://github.com/projectnessie/nessie)
- [Unity Catalog](https://github.com/unitycatalog/unitycatalog) (Databricks)

## Tech Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.25+ |
| HTTP Framework | Fiber (fasthttp) |
| Database | PostgreSQL 15+ |
| DB Driver | pgx/v5 |
| Migrations | golang-migrate |
| Config | Viper |
| Logging | slog |
| Object Storage | aws-sdk-go-v2, cloud.google.com/go/storage |

## Project Structure

```
bingsan/
├── cmd/
│   └── bingsan/          # Main entry point
├── internal/
│   ├── api/              # HTTP handlers & routing
│   │   ├── handlers/     # Request handlers
│   │   └── middleware/   # Auth, logging, etc.
│   ├── catalog/          # Iceberg catalog logic
│   ├── config/           # Configuration
│   ├── db/               # PostgreSQL operations
│   │   ├── migrations/   # SQL migrations
│   │   └── queries/      # SQL queries
│   ├── models/           # Data structures
│   └── storage/          # S3/GCS backends
├── deployments/
│   └── docker/           # Docker Compose files
├── tests/                # Integration tests
├── config.example.yaml
└── SPEC.md              # This file
```
