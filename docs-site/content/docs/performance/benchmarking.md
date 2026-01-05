---
title: "Benchmarking"
weight: 3
---

# Benchmarking

Bingsan includes comprehensive benchmarking support using both Go's built-in benchmarks and the Apache Polaris Tools Gatling framework for load testing.

## Quick Start

### Go Benchmarks

Run micro-benchmarks:

```bash
# All benchmarks
go test -bench=. -benchmem ./tests/benchmark/...

# Specific benchmark
go test -bench=BenchmarkTable -benchmem ./tests/benchmark/...

# Pool benchmarks
go test -bench=BenchmarkPool -benchmem ./tests/benchmark/...
```

### Load Testing with Polaris Tools

```bash
cd benchmarks

# One-time setup
make setup

# Start Bingsan with OAuth2
make start-bingsan

# Create test dataset
make create-dataset

# Run read benchmark
make read-benchmark

# View results
make report
```

## Go Benchmarks

### Available Benchmarks

| Benchmark | Description |
|-----------|-------------|
| `BenchmarkBaseline` | Baseline without pooling |
| `BenchmarkPool` | With object pooling |
| `BenchmarkTable` | Table operations |
| `BenchmarkNamespace` | Namespace operations |
| `BenchmarkConcurrent` | Concurrent load |
| `BenchmarkMemory` | Memory allocation |

### Running Benchmarks

**Basic run:**
```bash
go test -bench=. -benchmem ./tests/benchmark/...
```

**Compare before/after:**
```bash
# Baseline
go test -bench=. -benchmem ./tests/benchmark/... | tee baseline.txt

# After changes
go test -bench=. -benchmem ./tests/benchmark/... | tee optimized.txt

# Compare
benchstat baseline.txt optimized.txt
```

**With CPU profile:**
```bash
go test -bench=BenchmarkTable -cpuprofile=cpu.prof ./tests/benchmark/...
go tool pprof cpu.prof
```

**With memory profile:**
```bash
go test -bench=BenchmarkTable -memprofile=mem.prof ./tests/benchmark/...
go tool pprof mem.prof
```

### Expected Results

| Benchmark | Ops/sec | ns/op | B/op | allocs/op |
|-----------|---------|-------|------|-----------|
| TableMetadata | 100,000+ | <10,000 | <5,000 | <50 |
| LargeSchema | 10,000+ | <100,000 | <50,000 | <200 |
| PoolGet/Put | 20,000,000+ | <50 | 0 | 0 |
| Concurrent-100 | 1,000,000+ | <3,000 | <1,000 | <10 |

---

## Polaris Tools Load Testing

Bingsan supports load testing with [Apache Polaris Tools](https://github.com/apache/polaris-tools), a Gatling-based benchmark framework.

### Prerequisites

- **Java 17+** - Required for Gatling
- **Docker & Docker Compose** - For running Bingsan
- **Make** - For running benchmark commands

```bash
# Install Java (macOS)
brew install openjdk@17

# Verify
java -version  # Should show 17 or higher
```

### Setup

```bash
cd benchmarks

# Clone polaris-tools and build
make setup

# Verify setup
ls polaris-tools/benchmarks/  # Should exist
```

### Running Benchmarks

**Start the benchmark stack:**
```bash
make start-bingsan
```

This starts Bingsan with:
- OAuth2 authentication enabled
- Polaris compatibility mode
- PostgreSQL database
- Exposed on port 8181

**Create test dataset:**
```bash
make create-dataset
```

Creates namespaces, tables, and views for testing.

**Run benchmarks:**

| Command | Description |
|---------|-------------|
| `make read-benchmark` | Read-only operations |
| `make read-update-benchmark` | Mixed read/write (80/20) |
| `make create-commits-benchmark` | Commit throughput |
| `make weighted-benchmark` | Weighted workload simulation |
| `make full-benchmark` | All benchmarks sequentially |

**View results:**
```bash
make report
```

Opens the HTML report in your browser.

### Configuration

Edit `config/bingsan.conf`:

```hocon
# Connection
http.base-url = "http://localhost:8181"

# Authentication
auth.client-id = "benchmark-client"
auth.client-secret = "benchmark-secret"

# Dataset size
dataset {
  namespace-width = 2    # Namespaces per level
  namespace-depth = 3    # Namespace tree depth
  tables-per-namespace = 5
  views-per-namespace = 3
}

# Workload settings
workload.read-update-tree-dataset {
  read-write-ratio = 0.8     # 80% read, 20% write
  throughput = "50/sec"      # Target throughput
  duration-in-minutes = 3    # Test duration
}
```

### Benchmark Simulations

#### CreateTreeDataset
Creates test data structure:
- Hierarchical namespace tree
- Tables and views in each namespace
- Must run before other benchmarks

#### ReadTreeDataset
Read-only operations:
- List namespaces
- Get table/view metadata
- Check existence

#### ReadUpdateTreeDataset
Mixed workload:
- Configurable read/write ratio
- Property updates
- New resource creation

#### CreateCommits
Commit throughput:
- Table commits at configurable rate
- Measures commit latency
- Tests lock contention

#### WeightedWorkloadOnTreeDataset
Realistic simulation:
- Multiple reader/writer groups
- Configurable timing variance
- Tests conflict handling

### Results

Results are stored in:
```
polaris-tools/benchmarks/build/reports/gatling/<simulation>/
```

Each report includes:
- **Response time distribution** (p50, p95, p99)
- **Requests per second** over time
- **Active users** (concurrent load)
- **Error breakdown**

---

## Polaris Compatibility Mode

Polaris Tools expects a Polaris-compatible API. Bingsan enables this via:

```yaml
compat:
  polaris_enabled: true
```

This adds:
- **Path rewriting**: `/api/catalog/v1/{catalog}/...` → `/v1/...`
- **Polaris OAuth endpoint**: `/api/catalog/v1/oauth/tokens`
- **Mock Management API**: `/api/management/v1/catalogs`

{{< hint warning >}}
**Production Warning**: Keep `polaris_enabled: false` in production. This mode is only for benchmark compatibility.
{{< /hint >}}

---

## Comparing Results

### Manual Comparison

1. Run baseline benchmark, save report path
2. Make changes to Bingsan
3. Run benchmark again
4. Compare HTML reports side by side

### Using benchstat

For Go benchmarks:

```bash
# Install benchstat
go install golang.org/x/perf/cmd/benchstat@latest

# Run comparison
benchstat baseline.txt optimized.txt
```

Output:
```
name           old time/op    new time/op    delta
TableMeta-8     15.2µs ± 2%    10.1µs ± 1%  -33.55%  (p=0.000 n=10+10)
LargeSchema-8    120µs ± 3%      90µs ± 2%  -25.00%  (p=0.000 n=10+10)

name           old alloc/op   new alloc/op   delta
TableMeta-8     4.86kB ± 0%    3.84kB ± 0%  -20.99%  (p=0.000 n=10+10)
LargeSchema-8   48.2kB ± 0%    38.5kB ± 0%  -20.12%  (p=0.000 n=10+10)
```

---

## Troubleshooting

### Java Version Error

```bash
# Check version
java -version

# Set JAVA_HOME (macOS)
export JAVA_HOME=$(/usr/libexec/java_home -v 17)
```

### Connection Refused

```bash
# Check if Bingsan is running
docker ps | grep bingsan-bench

# View logs
make logs

# Test connectivity
make quick-test
```

### OAuth2 Errors

```bash
# Verify OAuth2 is enabled
curl http://localhost:8181/v1/config

# Test token endpoint
curl -X POST http://localhost:8181/v1/oauth/tokens \
  -d "grant_type=client_credentials&client_id=benchmark-client&client_secret=benchmark-secret"
```

### Out of Memory

Edit `polaris-tools/benchmarks/build.gradle.kts`:

```kotlin
gatling {
    jvmArgs = listOf("-Xmx2g", "-Xms1g")
}
```

### Cleanup

```bash
# Remove polaris-tools
make clean

# Remove docker volumes
make clean-data

# Stop Bingsan
make stop-bingsan
```

---

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Benchmark

on:
  push:
    branches: [main]
  pull_request:

jobs:
  benchmark:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'

      - name: Run benchmarks
        run: |
          go test -bench=. -benchmem ./tests/benchmark/... | tee results.txt

      - name: Compare with baseline
        run: |
          benchstat baseline.txt results.txt
```

### Performance Gate

Fail CI if performance regresses:

```bash
#!/bin/bash
# scripts/check-perf.sh

benchstat -delta-test=none baseline.txt results.txt | grep -E '\+[0-9]+\.[0-9]+%' && {
  echo "Performance regression detected!"
  exit 1
}
```
