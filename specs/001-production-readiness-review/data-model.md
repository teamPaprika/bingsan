# Data Model: Production Readiness Review

**Phase 1 Output** | **Date**: 2025-12-23 | **Branch**: `001-production-readiness-review`

## Overview

This document defines the data models for testing infrastructure. The core Iceberg catalog entities (Namespace, Table, View) are already defined in the existing implementation. This feature adds test-specific entities and configurations.

---

## Test Configuration Entities

### TestSuite

Represents a collection of related test cases.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| name | string | yes | Unique suite identifier (e.g., "contract", "integration") |
| type | enum | yes | "contract", "integration", "benchmark", "e2e" |
| timeout | duration | no | Maximum execution time (default: 10m) |
| parallelism | int | no | Max concurrent tests (default: GOMAXPROCS) |
| tags | []string | no | Labels for filtering (e.g., "slow", "flaky") |

### TestCase

Represents an individual test scenario.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | string | yes | Unique test identifier |
| suite | string | yes | Parent suite name |
| name | string | yes | Human-readable test name |
| description | string | no | What the test validates |
| method | string | yes | HTTP method (GET, POST, etc.) |
| path | string | yes | API endpoint path |
| requestBody | json | no | Request payload template |
| expectedStatus | int | yes | Expected HTTP status code |
| expectedSchema | string | no | JSON schema for response validation |
| setup | []string | no | Pre-test operations |
| teardown | []string | no | Post-test cleanup |

### BenchmarkConfig

Configuration for performance benchmarks.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| name | string | yes | Benchmark identifier |
| endpoint | string | yes | Target API endpoint |
| method | string | yes | HTTP method |
| concurrency | int | yes | Number of concurrent workers |
| duration | duration | yes | Test duration |
| targetRPS | int | no | Target requests per second |
| thresholds | Thresholds | yes | Pass/fail criteria |

### Thresholds

Performance thresholds for benchmark validation.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| p50_ms | int | yes | 50th percentile latency threshold |
| p95_ms | int | yes | 95th percentile latency threshold |
| p99_ms | int | yes | 99th percentile latency threshold |
| errorRate | float | yes | Maximum acceptable error rate (0.0-1.0) |
| minRPS | int | no | Minimum requests per second |

---

## Test Fixture Entities

### TestNamespace

Test fixture for namespace operations.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| name | []string | yes | Hierarchical namespace identifier |
| properties | map[string]string | no | Namespace properties |
| cleanup | bool | yes | Whether to delete after test (default: true) |

### TestTable

Test fixture for table operations.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| namespace | []string | yes | Parent namespace |
| name | string | yes | Table name |
| schema | Schema | yes | Table schema |
| partitionSpec | PartitionSpec | no | Partition specification |
| properties | map[string]string | no | Table properties |
| cleanup | bool | yes | Whether to delete after test |

### TestView

Test fixture for view operations.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| namespace | []string | yes | Parent namespace |
| name | string | yes | View name |
| schema | Schema | yes | View schema |
| sqlQuery | string | yes | SQL representation |
| properties | map[string]string | no | View properties |
| cleanup | bool | yes | Whether to delete after test |

---

## Contract Test Entities

### OpenAPIContract

Represents an OpenAPI spec contract.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| specPath | string | yes | Path to OpenAPI YAML file |
| baseURL | string | yes | API base URL for testing |
| endpoints | []EndpointContract | yes | List of endpoint contracts |

### EndpointContract

Contract for a single API endpoint.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| operationId | string | yes | OpenAPI operation ID |
| path | string | yes | Endpoint path |
| method | string | yes | HTTP method |
| requestSchema | string | no | JSON schema ref for request |
| responseSchema | string | yes | JSON schema ref for response |
| statusCodes | []int | yes | Valid response status codes |

---

## Kubernetes Deployment Entities

### DeploymentConfig

Kubernetes deployment configuration.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| name | string | yes | Deployment name |
| namespace | string | yes | Kubernetes namespace |
| replicas | int | yes | Number of replicas |
| image | string | yes | Container image |
| resources | ResourceRequirements | yes | CPU/memory limits |
| env | []EnvVar | no | Environment variables |

### ResourceRequirements

Container resource configuration.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| requestsCPU | string | yes | CPU request (e.g., "250m") |
| requestsMemory | string | yes | Memory request (e.g., "256Mi") |
| limitsCPU | string | yes | CPU limit (e.g., "1000m") |
| limitsMemory | string | yes | Memory limit (e.g., "512Mi") |

---

## State Transitions

### Test Execution States

```
PENDING → RUNNING → PASSED
                  → FAILED
                  → SKIPPED
                  → TIMEOUT
```

### Benchmark Execution States

```
PENDING → WARMUP → RUNNING → COMPLETED
                           → FAILED
                           → THRESHOLD_EXCEEDED
```

---

## Validation Rules

### TestCase Validation

1. `id` must be unique within the test suite
2. `method` must be a valid HTTP method
3. `path` must start with `/`
4. `expectedStatus` must be a valid HTTP status code (100-599)
5. If `requestBody` is provided, `method` must be POST, PUT, or PATCH

### BenchmarkConfig Validation

1. `concurrency` must be > 0
2. `duration` must be > 0
3. `thresholds.p50_ms` <= `thresholds.p95_ms` <= `thresholds.p99_ms`
4. `thresholds.errorRate` must be between 0.0 and 1.0

### DeploymentConfig Validation

1. `replicas` must be >= 1
2. `resources.requestsCPU` <= `resources.limitsCPU`
3. `resources.requestsMemory` <= `resources.limitsMemory`
4. `image` must be a valid container image reference

---

## Entity Relationships

```
TestSuite (1) ←→ (N) TestCase
BenchmarkConfig (1) ←→ (1) Thresholds
OpenAPIContract (1) ←→ (N) EndpointContract
DeploymentConfig (1) ←→ (1) ResourceRequirements
TestTable (N) ←→ (1) TestNamespace
TestView (N) ←→ (1) TestNamespace
```
