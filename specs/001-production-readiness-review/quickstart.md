# Quickstart: Production Readiness Review

**Phase 1 Output** | **Date**: 2025-12-23 | **Branch**: `001-production-readiness-review`

## Overview

This guide walks through validating Bingsan's production readiness by running the test suites and deployment validation.

## Prerequisites

- Go 1.25+
- Docker & Docker Compose
- Python 3.9+ (for PyIceberg tests)
- kubectl (for Kubernetes validation)

## 1. Run Contract Tests

Contract tests validate that Bingsan's API matches the Apache Iceberg REST Catalog OpenAPI specification.

```bash
# Start the test environment
docker compose -f deployments/docker/docker-compose.yml up -d

# Wait for services to be healthy
docker compose -f deployments/docker/docker-compose.yml ps

# Run contract tests
go test -v ./tests/contract/... -timeout 10m

# Check results
echo "Contract tests passed: API is spec-compliant"
```

**Expected Output:**
- All tests pass
- No schema validation errors
- All required endpoints respond correctly

## 2. Run Integration Tests

### PyIceberg Integration

```bash
# Install PyIceberg
pip install pyiceberg pytest

# Run PyIceberg tests
cd tests/integration
pytest test_pyiceberg.py -v

# Verify output
echo "PyIceberg integration passed"
```

### Spark Integration

```bash
# Start Spark test environment
docker compose -f deployments/docker/docker-compose-spark.yml up -d

# Wait for Spark to be ready
sleep 30

# Run Spark tests
docker compose -f deployments/docker/docker-compose-spark.yml exec spark-test \
    pytest /tests/test_spark.py -v

# Check results
echo "Spark integration passed"

# Cleanup
docker compose -f deployments/docker/docker-compose-spark.yml down -v
```

**Expected Output:**
- Namespace creation works
- Table CRUD operations succeed
- Schema evolution works
- Snapshot history accessible

## 3. Run Performance Benchmarks

```bash
# Run Go benchmarks
go test -bench=. ./tests/benchmark/... -benchmem -benchtime=10s

# Expected output:
# BenchmarkHealthCheck-8     500000    2341 ns/op    1024 B/op    12 allocs/op
# BenchmarkListNamespaces-8  100000   15234 ns/op    2048 B/op    24 allocs/op
# BenchmarkCreateTable-8      50000   32456 ns/op    4096 B/op    48 allocs/op

# Run load test with vegeta
echo "GET http://localhost:8181/health" | vegeta attack -rate=1000 -duration=30s | vegeta report

# Expected: 99%ile latency < 10ms, 0% errors
```

**Performance Targets:**

| Metric | Target | How to Verify |
|--------|--------|---------------|
| Metadata p99 latency | < 10ms | Check vegeta report |
| Requests per second | > 5,000 | Check RPS in report |
| Error rate | 0% | Check success rate |

## 4. Validate Kubernetes Deployment

```bash
# Apply Kubernetes manifests
kubectl apply -k deployments/kubernetes/base

# Wait for deployment
kubectl rollout status deployment/bingsan -n default

# Check pods are healthy
kubectl get pods -l app=bingsan

# Test health endpoint
kubectl port-forward svc/bingsan 8181:8181 &
curl http://localhost:8181/health

# Check Prometheus metrics
curl http://localhost:8181/metrics | head -20

# Cleanup
kubectl delete -k deployments/kubernetes/base
```

**Expected Output:**
- Pod status: Running
- Health check returns 200
- Metrics endpoint returns Prometheus format

## 5. Complete Validation Checklist

Run through this checklist to confirm production readiness:

```bash
# 1. API Compliance
go test -v ./tests/contract/...
# [ ] All contract tests pass

# 2. Client Compatibility
pytest tests/integration/test_pyiceberg.py -v
# [ ] PyIceberg can create namespaces
# [ ] PyIceberg can create tables
# [ ] PyIceberg can load table metadata

# 3. Performance
go test -bench=. ./tests/benchmark/...
# [ ] Latency targets met
# [ ] No memory leaks
# [ ] Throughput targets met

# 4. Observability
curl http://localhost:8181/metrics
# [ ] Prometheus metrics exposed
# [ ] Request latency histograms present
# [ ] Database connection metrics present

# 5. Health Checks
curl http://localhost:8181/health
curl http://localhost:8181/ready
# [ ] Liveness returns 200
# [ ] Readiness returns 200

# 6. Deployment
kubectl apply -k deployments/kubernetes/base
# [ ] Pods start successfully
# [ ] Health probes pass
# [ ] Resource limits applied
```

## 6. Troubleshooting

### Contract Tests Failing

```bash
# Check API is running
curl http://localhost:8181/health

# Check database connection
docker compose logs postgres

# Check OpenAPI spec version
cat tests/contract/iceberg-rest-catalog-open-api.yaml | grep 'version:'
```

### Performance Below Target

```bash
# Check for goroutine leaks
go test -bench=. -cpuprofile=cpu.out
go tool pprof cpu.out

# Check database connection pool
curl http://localhost:8181/metrics | grep db_connections

# Check for slow queries
docker compose logs catalog 2>&1 | grep -i slow
```

### Kubernetes Deployment Issues

```bash
# Check pod events
kubectl describe pod -l app=bingsan

# Check logs
kubectl logs -l app=bingsan --tail=100

# Check resource usage
kubectl top pod -l app=bingsan
```

## Next Steps

After completing validation:

1. **Document Results**: Record test results in `specs/001-production-readiness-review/checklists/results.md`
2. **Address Gaps**: Create issues for any failing tests or missing features
3. **Performance Baseline**: Save benchmark results for regression detection
4. **Deployment**: Use validated Kubernetes manifests for production deployment

## Files Created

| File | Purpose |
|------|---------|
| `tests/contract/*.go` | Contract tests against OpenAPI spec |
| `tests/integration/test_pyiceberg.py` | PyIceberg integration tests |
| `tests/integration/test_spark.py` | Spark integration tests |
| `tests/benchmark/*_bench_test.go` | Performance benchmarks |
| `deployments/kubernetes/base/*` | Kubernetes deployment manifests |
