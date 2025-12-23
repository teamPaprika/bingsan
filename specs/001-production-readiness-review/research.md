# Research: Production Readiness Review

**Phase 0 Output** | **Date**: 2025-12-23 | **Branch**: `001-production-readiness-review`

## Summary

Research findings for implementing comprehensive testing, benchmarking, and deployment infrastructure to validate Bingsan's production readiness.

---

## 1. Contract Testing Against OpenAPI Spec

### Decision: Use `kin-openapi` library

**Rationale**: kin-openapi is the most mature and widely-used Go library for OpenAPI 3.x validation. It provides request/response validation against OpenAPI specs.

**Alternatives Considered**:
- `oapi-codegen`: Good for code generation but less flexible for validation-only use cases
- Manual JSON schema validation: Too labor-intensive for full spec coverage
- Postman/Newman: Not Go-native, harder to integrate with CI

### Implementation Pattern

```go
package contract_test

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/getkin/kin-openapi/openapi3"
    "github.com/getkin/kin-openapi/openapi3filter"
    "github.com/getkin/kin-openapi/routers/gorillamux"
)

func TestAPICompliance(t *testing.T) {
    // Load OpenAPI spec
    loader := openapi3.NewLoader()
    doc, err := loader.LoadFromFile("../../iceberg-rest-catalog-open-api.yaml")
    if err != nil {
        t.Fatalf("Failed to load OpenAPI spec: %v", err)
    }

    // Create router for request matching
    router, err := gorillamux.NewRouter(doc)
    if err != nil {
        t.Fatalf("Failed to create router: %v", err)
    }

    // Table-driven test cases
    tests := []struct {
        name       string
        method     string
        path       string
        body       string
        wantStatus int
    }{
        {"ListNamespaces", "GET", "/v1/namespaces", "", 200},
        {"CreateNamespace", "POST", "/v1/namespaces", `{"namespace":["test"]}`, 200},
        {"GetNamespace", "GET", "/v1/namespaces/test", "", 200},
        // ... more test cases
    }

    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
            req.Header.Set("Content-Type", "application/json")

            // Find route and validate request
            route, pathParams, err := router.FindRoute(req)
            if err != nil {
                t.Fatalf("Route not found: %v", err)
            }

            requestValidationInput := &openapi3filter.RequestValidationInput{
                Request:    req,
                PathParams: pathParams,
                Route:      route,
            }

            if err := openapi3filter.ValidateRequest(context.Background(), requestValidationInput); err != nil {
                t.Errorf("Request validation failed: %v", err)
            }

            // Execute request and validate response
            rr := httptest.NewRecorder()
            handler.ServeHTTP(rr, req)

            responseValidationInput := &openapi3filter.ResponseValidationInput{
                RequestValidationInput: requestValidationInput,
                Status:                 rr.Code,
                Header:                 rr.Header(),
                Body:                   io.NopCloser(rr.Body),
            }

            if err := openapi3filter.ValidateResponse(context.Background(), responseValidationInput); err != nil {
                t.Errorf("Response validation failed: %v", err)
            }
        })
    }
}
```

### OpenAPI Spec Endpoints to Test

From Apache Iceberg REST Catalog spec:

| Category | Endpoints | Method |
|----------|-----------|--------|
| Config | `/v1/config` | GET |
| OAuth | `/v1/oauth/tokens` | POST |
| Namespaces | `/v1/{prefix}/namespaces` | GET, POST |
| Namespaces | `/v1/{prefix}/namespaces/{namespace}` | GET, HEAD, DELETE |
| Namespaces | `/v1/{prefix}/namespaces/{namespace}/properties` | POST |
| Tables | `/v1/{prefix}/namespaces/{namespace}/tables` | GET, POST |
| Tables | `/v1/{prefix}/namespaces/{namespace}/tables/{table}` | GET, POST, DELETE, HEAD |
| Tables | `/v1/{prefix}/tables/rename` | POST |
| Tables | `/v1/{prefix}/namespaces/{namespace}/register` | POST |
| Views | `/v1/{prefix}/namespaces/{namespace}/views` | GET, POST |
| Views | `/v1/{prefix}/namespaces/{namespace}/views/{view}` | GET, POST, DELETE, HEAD |
| Views | `/v1/{prefix}/views/rename` | POST |
| Scan | `/v1/{prefix}/namespaces/{namespace}/tables/{table}/plan` | POST |
| Scan | `/v1/{prefix}/namespaces/{namespace}/tables/{table}/plan/{plan-id}` | GET, DELETE |
| Scan | `/v1/{prefix}/namespaces/{namespace}/tables/{table}/tasks` | POST |
| Transactions | `/v1/{prefix}/transactions/commit` | POST |
| Credentials | `/v1/{prefix}/namespaces/{namespace}/tables/{table}/credentials` | GET |
| Metrics | `/v1/{prefix}/namespaces/{namespace}/tables/{table}/metrics` | POST |

---

## 2. Integration Testing with PyIceberg

### Decision: Use PyIceberg `RestCatalog` with pytest

**Rationale**: PyIceberg is the official Python client for Iceberg and is commonly used in data engineering workflows. Testing compatibility ensures production viability.

**Alternatives Considered**:
- Spark-only testing: Too heavyweight for CI; PyIceberg is faster
- Java Iceberg client: Requires JVM setup, slower startup
- Mock clients: Don't validate real compatibility

### PyIceberg Configuration

```python
# tests/integration/conftest.py
import pytest
from pyiceberg.catalog.rest import RestCatalog

@pytest.fixture(scope="session")
def rest_catalog():
    """Create a REST catalog connection to Bingsan."""
    catalog = RestCatalog(
        name="bingsan",
        **{
            "uri": "http://localhost:8181",
            "warehouse": "s3://warehouse/",
            "s3.endpoint": "http://localhost:9000",
            "s3.access-key-id": "minioadmin",
            "s3.secret-access-key": "minioadmin",
        }
    )
    return catalog

@pytest.fixture(scope="function")
def test_namespace(rest_catalog):
    """Create a temporary namespace for each test."""
    ns = ("test", f"ns_{uuid.uuid4().hex[:8]}")
    rest_catalog.create_namespace(ns)
    yield ns
    rest_catalog.drop_namespace(ns)
```

### Test Scenarios

```python
# tests/integration/test_pyiceberg.py
import pytest
from pyiceberg.schema import Schema
from pyiceberg.types import LongType, StringType, NestedField

def test_create_namespace(rest_catalog, test_namespace):
    """Verify namespace creation and listing."""
    namespaces = rest_catalog.list_namespaces()
    assert test_namespace in namespaces

def test_create_table(rest_catalog, test_namespace):
    """Verify table creation with schema."""
    schema = Schema(
        NestedField(1, "id", LongType(), required=True),
        NestedField(2, "name", StringType(), required=False),
    )
    table = rest_catalog.create_table(
        identifier=(*test_namespace, "test_table"),
        schema=schema,
    )
    assert table is not None
    assert table.name() == "test_table"

def test_load_table(rest_catalog, test_namespace):
    """Verify table loading and metadata retrieval."""
    # Create table first
    schema = Schema(NestedField(1, "id", LongType()))
    rest_catalog.create_table((*test_namespace, "load_test"), schema=schema)

    # Load and verify
    table = rest_catalog.load_table((*test_namespace, "load_test"))
    assert table.schema() == schema

def test_table_commit(rest_catalog, test_namespace):
    """Verify table metadata commits."""
    schema = Schema(NestedField(1, "id", LongType()))
    table = rest_catalog.create_table((*test_namespace, "commit_test"), schema=schema)

    # Evolve schema
    with table.update_schema() as update:
        update.add_column("name", StringType())

    # Reload and verify
    table = rest_catalog.load_table((*test_namespace, "commit_test"))
    assert "name" in [f.name for f in table.schema().fields]
```

---

## 3. Spark Integration Testing

### Decision: Use Docker Compose with Spark 3.5+ and pytest

**Rationale**: Spark is the primary consumer of Iceberg catalogs. Docker Compose provides reproducible CI environments.

**Alternatives Considered**:
- Local Spark installation: Non-reproducible across environments
- Databricks testing: Requires paid account, not suitable for open source
- Spark standalone mode only: Missing cluster behavior validation

### Docker Compose Configuration

```yaml
# deployments/docker/docker-compose-spark.yml
version: '3.8'
services:
  catalog:
    build: ../..
    ports: ["8181:8181"]
    environment:
      - ICEBERG_DATABASE_HOST=postgres
      - ICEBERG_STORAGE_S3_ENDPOINT=http://minio:9000
    depends_on:
      postgres: {condition: service_healthy}
      minio: {condition: service_healthy}

  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: iceberg
      POSTGRES_PASSWORD: iceberg
      POSTGRES_DB: iceberg_catalog
    healthcheck:
      test: ["CMD", "pg_isready", "-U", "iceberg"]

  minio:
    image: minio/minio:latest
    command: server /data --console-address ":9001"
    healthcheck:
      test: ["CMD", "mc", "ready", "local"]

  spark:
    image: bitnami/spark:3.5
    environment:
      - SPARK_MODE=master
    depends_on:
      catalog: {condition: service_healthy}
```

### Spark Test Configuration

```python
# tests/integration/test_spark.py
from pyspark.sql import SparkSession
import pytest

@pytest.fixture(scope="session")
def spark():
    session = SparkSession.builder \
        .appName("bingsan-integration-test") \
        .config("spark.sql.catalog.iceberg", "org.apache.iceberg.spark.SparkCatalog") \
        .config("spark.sql.catalog.iceberg.type", "rest") \
        .config("spark.sql.catalog.iceberg.uri", "http://catalog:8181") \
        .config("spark.sql.catalog.iceberg.warehouse", "s3://warehouse/") \
        .config("spark.sql.catalog.iceberg.s3.endpoint", "http://minio:9000") \
        .config("spark.sql.catalog.iceberg.s3.access-key-id", "minioadmin") \
        .config("spark.sql.catalog.iceberg.s3.secret-access-key", "minioadmin") \
        .config("spark.sql.extensions", "org.apache.iceberg.spark.extensions.IcebergSparkSessionExtensions") \
        .getOrCreate()
    yield session
    session.stop()

def test_spark_create_namespace(spark):
    spark.sql("CREATE NAMESPACE IF NOT EXISTS iceberg.spark_test")
    result = spark.sql("SHOW NAMESPACES IN iceberg").collect()
    assert any(row[0] == "spark_test" for row in result)

def test_spark_create_table(spark):
    spark.sql("""
        CREATE TABLE iceberg.spark_test.users (
            id LONG, name STRING
        ) USING iceberg
    """)
    tables = spark.sql("SHOW TABLES IN iceberg.spark_test").collect()
    assert any(row[1] == "users" for row in tables)
```

---

## 4. Performance Benchmarking

### Decision: Use Go's built-in `testing.B` with `b.RunParallel()`

**Rationale**: Native Go benchmarking integrates with CI and provides consistent, reproducible measurements. For load testing, use `vegeta` or `k6`.

**Alternatives Considered**:
- vegeta only: Good for HTTP but doesn't test handler internals
- wrk: C-based, harder to integrate with Go tests
- k6: Excellent but requires JavaScript scripting

### Benchmark Patterns

```go
// tests/benchmark/api_bench_test.go
package benchmark

import (
    "net/http"
    "testing"

    "github.com/gofiber/fiber/v2"
    "github.com/kimuyb/bingsan/internal/api"
)

func BenchmarkHealthCheck(b *testing.B) {
    app := setupTestApp()
    b.ResetTimer()

    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            req, _ := http.NewRequest("GET", "/health", nil)
            resp, _ := app.Test(req)
            resp.Body.Close()
        }
    })
}

func BenchmarkListNamespaces(b *testing.B) {
    app := setupTestApp()
    b.ReportAllocs()
    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        req, _ := http.NewRequest("GET", "/v1/namespaces", nil)
        resp, _ := app.Test(req)
        resp.Body.Close()
    }
}

func BenchmarkConcurrentTableCreation(b *testing.B) {
    app := setupTestApp()

    concurrencies := []int{1, 10, 50, 100}
    for _, c := range concurrencies {
        b.Run(fmt.Sprintf("concurrency-%d", c), func(b *testing.B) {
            b.SetParallelism(c)
            b.RunParallel(func(pb *testing.PB) {
                for pb.Next() {
                    // Create unique table per iteration
                    body := fmt.Sprintf(`{"name":"table_%d"}`, rand.Int())
                    req, _ := http.NewRequest("POST", "/v1/namespaces/test/tables",
                        strings.NewReader(body))
                    req.Header.Set("Content-Type", "application/json")
                    resp, _ := app.Test(req)
                    resp.Body.Close()
                }
            })
        })
    }
}
```

### Performance Targets (from Constitution)

| Metric | Target | Test Command |
|--------|--------|--------------|
| Metadata p99 | < 10ms | `go test -bench=. -benchtime=10s` |
| Concurrent connections | > 10,000 | `vegeta attack -rate=10000 -duration=30s` |
| Request throughput | > 5,000 req/s | `go test -bench=. -cpu=4` |
| Memory usage | < 500MB | `go test -bench=. -memprofile=mem.out` |
| Startup time | < 5s | Measured in integration tests |

---

## 5. Kubernetes Deployment

### Decision: Use Kustomize with base/overlays pattern

**Rationale**: Kustomize is built into kubectl and provides clean separation between base configs and environment-specific overlays.

**Alternatives Considered**:
- Helm: More complex for simple deployments; overkill for single service
- Raw YAML: No environment customization
- Terraform: Better for infrastructure, not application deployment

### Deployment Structure

```
deployments/kubernetes/
├── base/
│   ├── kustomization.yaml
│   ├── deployment.yaml
│   ├── service.yaml
│   ├── configmap.yaml
│   └── serviceaccount.yaml
└── overlays/
    ├── dev/
    │   ├── kustomization.yaml
    │   └── replicas-patch.yaml
    └── prod/
        ├── kustomization.yaml
        ├── replicas-patch.yaml
        └── resources-patch.yaml
```

### Base Deployment YAML

```yaml
# deployments/kubernetes/base/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: bingsan
  labels:
    app: bingsan
spec:
  replicas: 1
  selector:
    matchLabels:
      app: bingsan
  template:
    metadata:
      labels:
        app: bingsan
    spec:
      serviceAccountName: bingsan
      containers:
      - name: bingsan
        image: bingsan:latest
        ports:
        - containerPort: 8181
          name: http
        env:
        - name: ICEBERG_DATABASE_HOST
          valueFrom:
            configMapKeyRef:
              name: bingsan-config
              key: database-host
        livenessProbe:
          httpGet:
            path: /health
            port: http
          initialDelaySeconds: 5
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: http
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "1000m"
```

---

## Summary of Decisions

| Area | Decision | Key Library/Tool |
|------|----------|------------------|
| Contract Testing | kin-openapi validation | `github.com/getkin/kin-openapi` |
| PyIceberg Testing | pytest with RestCatalog | `pyiceberg`, `pytest` |
| Spark Testing | Docker Compose + PySpark | `bitnami/spark:3.5` |
| Benchmarking | Go testing.B + vegeta | `go test -bench`, `vegeta` |
| Kubernetes | Kustomize base/overlays | `kubectl apply -k` |

All NEEDS CLARIFICATION items from Technical Context have been resolved through this research.
