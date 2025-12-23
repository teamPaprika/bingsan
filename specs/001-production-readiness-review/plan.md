# Implementation Plan: Production Readiness Review

**Branch**: `001-production-readiness-review` | **Date**: 2025-12-23 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-production-readiness-review/spec.md`

## Summary

Validate Bingsan's production readiness by filling gaps identified in the codebase review: add comprehensive contract tests against Iceberg REST OpenAPI spec, integration tests with real clients (Spark, PyIceberg, Trino), performance benchmarks, and Kubernetes deployment manifests. The existing implementation is feature-complete for core catalog operations (Phases 1-5); this plan focuses on testing and deployment infrastructure.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: Fiber (HTTP), pgx/v5 (PostgreSQL), go-json (JSON), prometheus (metrics), golang-migrate (migrations)
**Storage**: PostgreSQL 15+ (metadata), S3-compatible (data lake)
**Testing**: `go test` with table-driven tests, shell scripts for integration (minimal coverage)
**Target Platform**: Linux containers (Docker), Kubernetes
**Project Type**: Single backend service (REST API)
**Performance Goals**: p99 < 10ms for metadata ops, > 10,000 concurrent connections, > 5,000 req/s
**Constraints**: < 500MB memory, < 5s startup, 30s graceful shutdown
**Scale/Scope**: Production data lake catalog serving Spark/Trino/Flink clusters

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Evidence |
|-----------|--------|----------|
| I. API Compliance First | PARTIAL | Core endpoints implemented but contract tests missing |
| II. Test-Driven Development | VIOLATION | Existing code lacks tests; this plan will add them retroactively |
| III. Go Idioms & Concurrency | PASS | Uses goroutines, context, sync primitives correctly |
| IV. Production-Grade Code | PASS | Graceful shutdown, health checks, error handling present |
| V. Golang Best Practices | PASS | golangci-lint configured, structured code |
| VI. Observability | PARTIAL | Prometheus metrics present; OpenTelemetry missing |
| VII. Simplicity & YAGNI | PASS | No over-engineering detected |
| Performance Standards | NOT VERIFIED | No benchmarks to confirm targets |
| Security Requirements | PARTIAL | Auth implemented; audit logging missing |
| Code Quality Gates | PARTIAL | Linting configured; coverage not measured |
| Deployment Policies | PARTIAL | Dockerfile present; Kubernetes manifests missing |

### Gate Violations Justification

| Violation | Justification | Resolution |
|-----------|---------------|------------|
| II. TDD | Existing implementation predates constitution; adding tests retroactively | Phase 0: Contract tests, Phase 1: Integration tests |
| I. API Compliance | Cannot verify without tests | Phase 0: Generate contract tests from OpenAPI spec |
| Performance Standards | Cannot verify without benchmarks | Phase 1: Add benchmark tests |
| Deployment Policies | Missing k8s manifests | Phase 1: Generate Kubernetes deployment |

**GATE RESULT**: PROCEED with justifications documented. All violations have resolution paths in this plan.

## Project Structure

### Documentation (this feature)

```text
specs/001-production-readiness-review/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output (test entity models)
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (test contracts)
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
# Existing structure (to be enhanced)
cmd/
└── iceberg-catalog/     # Main entry point

internal/
├── api/                 # HTTP handlers & routing
│   ├── handlers/        # Request handlers
│   └── middleware/      # Auth, logging
├── catalog/             # Iceberg catalog logic (empty - to review)
├── config/              # Configuration
├── db/                  # PostgreSQL operations
│   ├── migrations/      # SQL migrations
│   └── queries/         # SQL queries (empty - to review)
├── events/              # Event broker
├── metrics/             # Prometheus metrics
├── models/              # Data structures (empty - to review)
└── storage/             # S3/GCS backends (empty - to review)

tests/                   # Integration tests (minimal)
└── test_api.sh          # Basic shell test

# New additions for production readiness
tests/
├── contract/            # OpenAPI spec compliance tests
├── integration/         # Real client tests (Spark, PyIceberg)
├── benchmark/           # Performance benchmarks
└── e2e/                 # End-to-end scenarios

deployments/
├── docker/              # Existing
│   ├── Dockerfile
│   ├── docker-compose.yml
│   └── test.yaml
└── kubernetes/          # NEW: k8s manifests
    ├── deployment.yaml
    ├── service.yaml
    ├── configmap.yaml
    └── kustomization.yaml
```

**Structure Decision**: Extend existing single-project structure with dedicated test directories (contract, integration, benchmark, e2e) and new kubernetes deployment directory.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Multiple test directories | Different test types have different dependencies and runtime requirements | Single test directory would mix unit/integration/contract tests with conflicting setup |
| Kubernetes manifests | Required for production deployment per Deployment Policies | Docker Compose is insufficient for production multi-node deployments |
