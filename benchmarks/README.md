# Bingsan Performance Benchmarks

This directory contains performance benchmarks for the Bingsan Iceberg REST Catalog using the [Apache Polaris Tools](https://github.com/apache/polaris-tools) Gatling framework.

## Prerequisites

- **Java 17+** - Required for Gatling
- **Docker & Docker Compose** - For running bingsan
- **Make** - For running benchmark commands

### Installing Java (if needed)

```bash
# macOS
brew install openjdk@17

# Ubuntu/Debian
sudo apt install openjdk-17-jdk

# Verify installation
java -version  # Should show 17 or higher
```

## Quick Start

```bash
# 1. Set up the benchmark framework (one-time)
make setup

# 2. Start bingsan with OAuth2 enabled
make start-bingsan

# 3. Create test dataset
make create-dataset

# 4. Run read benchmark
make read-benchmark

# 5. View results in browser
make report

# 6. Stop bingsan when done
make stop-bingsan
```

## Available Commands

| Command | Description |
|---------|-------------|
| `make help` | Show all available commands |
| `make setup` | Clone polaris-tools and build simulations |
| `make start-bingsan` | Start bingsan with OAuth2 for benchmarking |
| `make stop-bingsan` | Stop bingsan benchmark stack |
| `make create-dataset` | Create test dataset (run first) |
| `make read-benchmark` | Run read-only operations benchmark |
| `make read-update-benchmark` | Run mixed read/write benchmark |
| `make create-commits-benchmark` | Run commit throughput benchmark |
| `make weighted-benchmark` | Run weighted workload benchmark |
| `make full-benchmark` | Run all benchmarks sequentially |
| `make report` | Open latest benchmark report |
| `make quick-test` | Test connectivity to bingsan |
| `make clean` | Remove polaris-tools |
| `make clean-data` | Remove docker volumes |

## Benchmark Simulations

### CreateTreeDataset
Populates the catalog with test data:
- Creates namespaces in a tree structure
- Creates tables and views in each namespace
- Must be run before other benchmarks

### ReadTreeDataset
Read-only operations:
- Lists namespaces
- Gets table/view metadata
- Checks existence

### ReadUpdateTreeDataset
Mixed workload:
- Configurable read/write ratio (default: 80% read, 20% write)
- Updates table/view properties
- Creates new tables/views

### CreateCommits
Commit throughput:
- Creates table commits at configurable rate
- Measures commit latency and throughput

### WeightedWorkloadOnTreeDataset
Realistic workload simulation:
- Multiple reader/writer groups
- Configurable mean and variance for operation timing
- Tests conflict handling

## Polaris Compatibility Mode

The polaris-tools benchmarks expect a Polaris-compatible API. Bingsan supports this via an opt-in compatibility layer that adds:

- **Path rewriting**: `/api/catalog/v1/{catalog}/...` → `/v1/...`
- **Polaris OAuth endpoint**: `/api/catalog/v1/oauth/tokens`
- **Mock Management API**: `/api/management/v1/catalogs`

This is **automatically enabled** in the benchmark docker-compose via:
```yaml
ICEBERG_COMPAT_POLARIS_ENABLED: "true"
```

For production use, this should remain **disabled** (the default).

## Configuration

Edit `config/bingsan.conf` to customize:

```hocon
# Connection
http.base-url = "http://localhost:8181"

# Authentication
auth.client-id = "benchmark-client"
auth.client-secret = "benchmark-secret"

# Dataset size
dataset {
  namespace-width = 2
  namespace-depth = 3
  tables-per-namespace = 5
  views-per-namespace = 3
}

# Workload settings
workload.read-update-tree-dataset {
  read-write-ratio = 0.8
  throughput = "50/sec"
  duration-in-minutes = 3
}
```

## Results

Benchmark results are stored in:
```
polaris-tools/benchmarks/build/reports/gatling/<simulation-name>/
```

Each report includes:
- **Response time distribution** - Percentiles (p50, p95, p99)
- **Requests per second** - Throughput over time
- **Active users** - Concurrent load
- **Errors** - Failed requests and error types

## Architecture

```
benchmarks/
├── Makefile                     # Benchmark commands
├── README.md                    # This file
├── config/
│   └── bingsan.conf            # Gatling configuration overrides
├── docker-compose.benchmark.yaml # Bingsan stack with OAuth2
├── scripts/
│   └── setup.sh                # Setup script
└── polaris-tools/              # Cloned apache/polaris-tools repo
    └── benchmarks/
        └── build/reports/gatling/  # Results here
```

## Troubleshooting

### Java version error
```bash
# Check Java version
java -version

# Set JAVA_HOME if needed
export JAVA_HOME=$(/usr/libexec/java_home -v 17)
```

### Connection refused
```bash
# Check if bingsan is running
docker ps | grep bingsan-bench

# Check logs
make logs

# Test connectivity
make quick-test
```

### OAuth2 authentication errors
```bash
# Verify OAuth2 is enabled
curl http://localhost:8181/v1/config

# Test token endpoint
curl -X POST http://localhost:8181/v1/oauth/tokens \
  -d "grant_type=client_credentials&client_id=benchmark-client&client_secret=benchmark-secret"
```

### Out of memory during simulation
Edit `polaris-tools/benchmarks/build.gradle.kts` and increase JVM heap:
```kotlin
gatling {
    jvmArgs = listOf("-Xmx2g", "-Xms1g")
}
```

## Comparing Results

To compare benchmark runs:

1. Run baseline benchmark and note the report directory
2. Make changes to bingsan
3. Run benchmark again
4. Compare the HTML reports side by side

For automated comparison, export results to CSV and use tools like `benchstat`.
