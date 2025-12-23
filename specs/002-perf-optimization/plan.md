# Implementation Plan: Performance Optimization (sync.Pool)

**Branch**: `002-perf-optimization` | **Date**: 2025-12-24 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/002-perf-optimization/spec.md`

## Summary

Implement object pooling using Go's `sync.Pool` to reduce memory allocation pressure and GC pause times in high-frequency API handlers. The optimization targets JSON serialization buffers in table/view metadata operations and byte slice allocation in token generation. Expected outcome: 30% reduction in allocations with no increase in latency.

## Technical Context

**Language/Version**: Go 1.21+
**Primary Dependencies**: Fiber (fasthttp), goccy/go-json, sync (standard library)
**Storage**: PostgreSQL (unchanged)
**Testing**: go test, benchmarks with `-benchmem`, pprof profiling
**Target Platform**: Linux containers (Docker/Kubernetes)
**Project Type**: Single Go module
**Performance Goals**: 30% reduction in allocations, <10ms p99 GC pause, maintain 5000 req/s
**Constraints**: Must not increase p50 latency, pool overhead <10MB idle
**Scale/Scope**: High-frequency endpoints: table load, table commit, view operations

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Evidence |
|-----------|--------|----------|
| I. API Compliance First | ✅ Pass | No API changes; internal optimization only |
| II. Test-Driven Development | ✅ Pass | Benchmark tests written first to measure baseline |
| III. Go Idioms & Concurrency | ✅ Pass | Using `sync.Pool` - standard library idiom |
| IV. Production-Grade Code | ✅ Pass | Pool metrics exposed, graceful degradation on exhaustion |
| V. Golang Best Practices | ✅ Pass | Follows Effective Go sync.Pool patterns |
| VI. Observability | ✅ Pass | Pool hit/miss metrics added to Prometheus |
| VII. Simplicity & YAGNI | ✅ Pass | Addressing measured performance need (benchmarks) |

**Performance Standards Alignment**:
- Metadata operation latency (p99) < 10ms: ✅ Maintained or improved
- Request throughput > 5,000 req/s: ✅ Maintained or improved
- Memory usage < 500MB base: ✅ Pool overhead <10MB

## Project Structure

### Documentation (this feature)

```text
specs/002-perf-optimization/
├── plan.md              # This file
├── research.md          # Phase 0: sync.Pool best practices
├── data-model.md        # Phase 1: Pool configurations
├── quickstart.md        # Phase 1: How to verify optimization
├── contracts/           # Phase 1: Benchmark test cases
└── tasks.md             # Phase 2: Implementation tasks
```

### Source Code (repository root)

```text
internal/
├── pool/                # NEW: Object pool management
│   ├── buffer.go        # Buffer pool for JSON serialization
│   ├── bytes.go         # Byte slice pool for token generation
│   └── metrics.go       # Prometheus metrics for pool utilization
├── api/
│   └── handlers/
│       ├── table.go     # MODIFIED: Use buffer pool
│       ├── view.go      # MODIFIED: Use buffer pool
│       └── oauth.go     # MODIFIED: Use byte pool
└── metrics/
    └── prometheus.go    # MODIFIED: Add pool metrics

tests/
├── benchmark/
│   ├── pool_bench_test.go    # NEW: Pool-specific benchmarks
│   └── memory_bench_test.go  # MODIFIED: Add allocation tracking
└── unit/
    └── pool_test.go          # NEW: Pool correctness tests
```

**Structure Decision**: Single Go module structure. New `internal/pool/` package for pool implementations to keep separation of concerns. Handlers modified to use pools via dependency injection.

## Complexity Tracking

> No violations - all changes align with constitution principles.

| Aspect | Decision | Justification |
|--------|----------|---------------|
| New package | `internal/pool/` | Encapsulates pool logic, testable in isolation |
| Metrics | Prometheus counters | Aligns with Observability principle (VI) |
| Buffer strategy | Pre-allocated byte slices | Standard Go idiom for high-throughput |
