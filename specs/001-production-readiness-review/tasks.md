# Tasks: Production Readiness Review

**Input**: Design documents from `/specs/001-production-readiness-review/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: This feature is specifically about adding comprehensive tests, so tests are REQUIRED for all user stories.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Go project**: `internal/`, `cmd/`, `tests/` at repository root
- **Deployments**: `deployments/docker/`, `deployments/kubernetes/`
- **Test types**: `tests/contract/`, `tests/integration/`, `tests/benchmark/`, `tests/e2e/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Test infrastructure initialization and framework setup

- [x] T001 Create test directory structure: `tests/contract/`, `tests/integration/`, `tests/benchmark/`, `tests/e2e/`
- [x] T002 Add test dependencies to `go.mod`: kin-openapi, testify
- [x] T003 [P] Download Iceberg REST OpenAPI spec to `tests/contract/iceberg-rest-catalog-open-api.yaml`
- [x] T004 [P] Create Python test environment with `tests/integration/requirements.txt` (pyiceberg, pytest)
- [x] T005 [P] Create Kubernetes base directory structure: `deployments/kubernetes/base/`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core test infrastructure that MUST be complete before user story tests can run

**CRITICAL**: No user story work can begin until this phase is complete

- [x] T006 Create test helper for HTTP server setup in `tests/helpers_test.go`
- [x] T007 [P] Create test database fixtures in `tests/fixtures/`
- [x] T008 [P] Create Docker Compose test environment in `deployments/docker/docker-compose-test.yml`
- [x] T009 Create OpenAPI spec loader utility in `tests/contract/loader.go`
- [x] T010 [P] Create pytest fixtures for REST catalog connection in `tests/integration/conftest.py`

**Checkpoint**: Test infrastructure ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Platform Engineer Validates Production Readiness (Priority: P1)

**Goal**: Validate Bingsan's API compliance against Iceberg REST OpenAPI spec and confirm reliability

**Independent Test**: Run `go test -v ./tests/contract/...` - all contract tests must pass

### Contract Tests for User Story 1

> **NOTE: Write these tests FIRST, ensure they FAIL before any implementation fixes**

- [x] T011 [P] [US1] Contract test for `/v1/config` endpoint in `tests/contract/config_test.go`
- [x] T012 [P] [US1] Contract test for namespace CRUD operations in `tests/contract/namespace_test.go`
- [x] T013 [P] [US1] Contract test for table CRUD operations in `tests/contract/table_test.go`
- [x] T014 [P] [US1] Contract test for view CRUD operations in `tests/contract/view_test.go`
- [x] T015 [P] [US1] Contract test for scan planning operations in `tests/contract/scan_test.go`
- [x] T016 [P] [US1] Contract test for multi-table transactions in `tests/contract/transaction_test.go`
- [x] T017 [P] [US1] Contract test for error responses (400, 404, 409) in `tests/contract/errors_test.go`

### Health & Reliability Tests for User Story 1

- [x] T018 [P] [US1] Health endpoint test (/health, /ready) in `tests/contract/health_test.go`
- [x] T019 [P] [US1] Graceful shutdown test in `tests/e2e/shutdown_test.go`
- [x] T020 [US1] Database connection failure handling test in `tests/e2e/resilience_test.go`

### Kubernetes Deployment for User Story 1

- [x] T021 [P] [US1] Create Kubernetes Deployment manifest in `deployments/kubernetes/base/deployment.yaml`
- [x] T022 [P] [US1] Create Kubernetes Service manifest in `deployments/kubernetes/base/service.yaml`
- [x] T023 [P] [US1] Create Kubernetes ConfigMap manifest in `deployments/kubernetes/base/configmap.yaml`
- [x] T024 [P] [US1] Create Kubernetes ServiceAccount manifest in `deployments/kubernetes/base/serviceaccount.yaml`
- [x] T025 [US1] Create Kustomization file in `deployments/kubernetes/base/kustomization.yaml`

### Implementation Fixes for User Story 1 (if contract tests reveal issues)

- [x] T026 [US1] Fix any schema validation issues found by contract tests
- [x] T027 [US1] Add missing response fields per OpenAPI spec (if any)
- [x] T028 [US1] Add missing error response formats per Iceberg spec (if any)

**Checkpoint**: API is 100% spec-compliant, Kubernetes deployment ready

---

## Phase 4: User Story 2 - Data Engineer Validates Client Compatibility (Priority: P2)

**Goal**: Verify Spark, PyIceberg, and Trino can connect and perform operations

**Independent Test**: Run `pytest tests/integration/test_pyiceberg.py -v` and Spark integration tests

### PyIceberg Integration Tests for User Story 2

- [x] T029 [P] [US2] PyIceberg namespace operations test in `tests/integration/test_pyiceberg.py::test_namespace_crud`
- [x] T030 [P] [US2] PyIceberg table creation test in `tests/integration/test_pyiceberg.py::test_create_table`
- [x] T031 [P] [US2] PyIceberg table loading test in `tests/integration/test_pyiceberg.py::test_load_table`
- [x] T032 [US2] PyIceberg schema evolution test in `tests/integration/test_pyiceberg.py::test_schema_evolution`
- [x] T033 [US2] PyIceberg snapshot history test in `tests/integration/test_pyiceberg.py::test_snapshot_history`

### Spark Integration Tests for User Story 2

- [x] T034 [P] [US2] Create Spark test Docker Compose in `deployments/docker/docker-compose-spark.yml`
- [x] T035 [P] [US2] Spark namespace operations test in `tests/integration/test_spark.py::test_spark_namespace`
- [x] T036 [US2] Spark table creation test in `tests/integration/test_spark.py::test_spark_create_table`
- [x] T037 [US2] Spark table read/write test in `tests/integration/test_spark.py::test_spark_read_write`
- [x] T038 [US2] Spark time travel test in `tests/integration/test_spark.py::test_spark_time_travel`

### Client Compatibility Fixes for User Story 2 (if integration tests reveal issues)

- [x] T039 [US2] Fix any PyIceberg compatibility issues (if any)
- [x] T040 [US2] Fix any Spark compatibility issues (if any)

**Checkpoint**: PyIceberg and Spark clients work correctly with Bingsan

---

## Phase 5: User Story 3 - Operations Team Monitors System Health (Priority: P3)

**Goal**: Validate Prometheus metrics, logging, and event streaming for production monitoring

**Independent Test**: Scrape /metrics and verify all expected metrics are present

### Metrics & Observability Tests for User Story 3

- [x] T041 [P] [US3] Prometheus metrics endpoint test in `tests/contract/metrics_test.go`
- [x] T042 [P] [US3] Request latency histogram metrics test in `tests/e2e/metrics_test.go::TestLatencyMetrics`
- [x] T043 [P] [US3] Database connection pool metrics test in `tests/e2e/metrics_test.go::TestDBMetrics`
- [x] T044 [US3] Entity count metrics test (namespaces, tables, views) in `tests/e2e/metrics_test.go::TestEntityMetrics`

### Event Streaming Tests for User Story 3

- [x] T045 [US3] WebSocket event stream connection test in `tests/e2e/events_test.go::TestEventConnection`
- [x] T046 [US3] Catalog change event test in `tests/e2e/events_test.go::TestCatalogEvents`

### Performance Benchmark Tests for User Story 3

- [x] T047 [P] [US3] Health check benchmark in `tests/benchmark/health_bench_test.go`
- [x] T048 [P] [US3] List namespaces benchmark in `tests/benchmark/namespace_bench_test.go`
- [x] T049 [P] [US3] Create table benchmark in `tests/benchmark/table_bench_test.go`
- [x] T050 [US3] Concurrent connections benchmark in `tests/benchmark/concurrent_bench_test.go`
- [x] T051 [US3] Memory usage benchmark with profiling in `tests/benchmark/memory_bench_test.go`

### Observability Fixes for User Story 3 (if tests reveal gaps)

- [x] T052 [US3] Add any missing Prometheus metrics (if any)
- [x] T053 [US3] Fix event streaming issues (if any)

**Checkpoint**: All observability requirements validated

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [x] T054 [P] Create CI workflow for contract tests in `.github/workflows/contract-tests.yml`
- [x] T055 [P] Create CI workflow for integration tests in `.github/workflows/integration-tests.yml`
- [x] T056 [P] Create CI workflow for benchmarks in `.github/workflows/benchmarks.yml`
- [x] T057 [P] Document test results in `specs/001-production-readiness-review/checklists/results.md`
- [x] T058 Run quickstart.md validation
- [x] T059 Code coverage report generation and threshold validation

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-5)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 -> P2 -> P3)
- **Polish (Phase 6)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - May need contract tests from US1 to verify API works
- **User Story 3 (P3)**: Can start after Foundational (Phase 2) - Benchmarks benefit from fixed API from US1

### Within Each User Story

- Tests MUST be written and FAIL before implementation fixes
- Contract tests before integration tests
- Core tests before edge case tests
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- All Foundational tasks marked [P] can run in parallel (within Phase 2)
- Once Foundational phase completes, all user stories can start in parallel (if team capacity allows)
- All contract tests for US1 marked [P] can run in parallel
- All integration tests for US2 marked [P] can run in parallel
- All benchmark tests for US3 marked [P] can run in parallel

---

## Parallel Example: User Story 1

```bash
# Launch all contract tests for User Story 1 together:
Task: "Contract test for /v1/config endpoint in tests/contract/config_test.go"
Task: "Contract test for namespace CRUD operations in tests/contract/namespace_test.go"
Task: "Contract test for table CRUD operations in tests/contract/table_test.go"
Task: "Contract test for view CRUD operations in tests/contract/view_test.go"
Task: "Contract test for scan planning operations in tests/contract/scan_test.go"
Task: "Contract test for multi-table transactions in tests/contract/transaction_test.go"
Task: "Contract test for error responses in tests/contract/errors_test.go"

# Launch all Kubernetes manifests together:
Task: "Create Kubernetes Deployment manifest in deployments/kubernetes/base/deployment.yaml"
Task: "Create Kubernetes Service manifest in deployments/kubernetes/base/service.yaml"
Task: "Create Kubernetes ConfigMap manifest in deployments/kubernetes/base/configmap.yaml"
Task: "Create Kubernetes ServiceAccount manifest in deployments/kubernetes/base/serviceaccount.yaml"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1 (Contract tests + K8s deployment)
4. **STOP and VALIDATE**: All contract tests pass, can deploy to Kubernetes
5. System is production-ready from API compliance perspective

### Incremental Delivery

1. Complete Setup + Foundational -> Test infrastructure ready
2. Add User Story 1 -> API compliance validated -> Deploy/Demo (MVP!)
3. Add User Story 2 -> Client compatibility validated -> Deploy/Demo
4. Add User Story 3 -> Observability validated -> Deploy/Demo
5. Each story adds confidence without breaking previous validation

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1 (Contract tests + K8s)
   - Developer B: User Story 2 (Integration tests)
   - Developer C: User Story 3 (Benchmarks + Metrics)
3. Stories complete and validate independently

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Verify tests fail before implementing fixes
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence
