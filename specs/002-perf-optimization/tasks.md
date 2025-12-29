# Tasks: Performance Optimization (sync.Pool)

**Input**: Design documents from `/specs/002-perf-optimization/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/benchmark-tests.yaml, quickstart.md

**Tests**: Benchmark tests are REQUIRED for this feature - they measure success criteria (30% allocation reduction).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2)
- Include exact file paths in descriptions

## Path Conventions

- **Source**: `internal/` at repository root
- **Tests**: `tests/` at repository root
- Paths follow existing Go project structure

---

## Phase 1: Setup (Pool Package Infrastructure)

**Purpose**: Create the pool package structure and establish baseline measurements

- [x] T001 Create pool package directory structure at `internal/pool/`
- [x] T002 [P] Create baseline benchmark file at `tests/benchmark/baseline_bench_test.go`
- [x] T003 [P] Run and record baseline benchmarks for table metadata operations

---

## Phase 2: Foundational (Pool Core Implementation)

**Purpose**: Core pool infrastructure that MUST be complete before optimizations can be applied

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T004 Implement BufferPool with sync.Pool in `internal/pool/buffer.go`
- [x] T005 [P] Implement BytePool for token generation in `internal/pool/bytes.go`
- [x] T006 [P] Implement pool metrics (hits/misses/returns/discards) in `internal/pool/metrics.go`
- [x] T007 Register pool metrics with Prometheus in `internal/pool/metrics.go` (auto-registered via promauto)
- [x] T008 [P] Create pool unit tests in `tests/unit/pool_test.go`

**Checkpoint**: Pool infrastructure ready - user story implementation can now begin

---

## Phase 3: User Story 1 - Platform Engineer Validates Reduced Memory Pressure (Priority: P1) 🎯 MVP

**Goal**: Reduce memory allocation rate by 30%+ and keep GC pause times under 10ms p99

**Independent Test**: Run load test with 10,000 concurrent requests and measure GC pause times and memory allocation rates before and after optimization.

### Benchmarks for User Story 1

> **NOTE: Run baseline benchmarks FIRST to establish comparison baseline**

- [x] T009 [P] [US1] Create pool benchmark tests in `tests/benchmark/pool_bench_test.go`
- [x] T010 [P] [US1] Create memory stability benchmark in `tests/benchmark/pool_bench_test.go`
- [x] T011 [P] [US1] Create concurrent access benchmark in `tests/benchmark/pool_bench_test.go`

### Implementation for User Story 1

- [x] T012 [US1] Integrate BufferPool into table handlers in `internal/api/handlers/table.go`
- [x] T013 [P] [US1] Integrate BufferPool into view handlers in `internal/api/handlers/view.go`
- [x] T014 [US1] Integrate BytePool into OAuth handler in `internal/api/handlers/oauth.go`
- [x] T015 [US1] Add defer patterns for proper buffer return in all modified handlers
- [x] T016 [US1] Verify pool metrics are exposed via `/metrics` endpoint (auto-registered via promauto)
- [x] T017 [US1] Run benchstat comparison: baseline vs optimized allocations (see benchstat_comparison.md)

**Checkpoint**: Memory allocation reduced by 30%+, GC pauses under 10ms p99

---

## Phase 4: User Story 2 - Data Engineer Experiences Faster Table Metadata Operations (Priority: P2)

**Goal**: Table metadata operations complete under latency targets (1ms serialization, 50ms avg response, 200ms for large schemas)

**Independent Test**: Benchmark table load and commit operations, measuring p50, p95, and p99 latencies.

### Benchmarks for User Story 2

- [x] T018 [P] [US2] Create table load latency benchmark in `tests/benchmark/table_bench_test.go`
- [x] T019 [P] [US2] Create concurrent request benchmark (100 concurrent) in `tests/benchmark/concurrent_bench_test.go`
- [x] T020 [P] [US2] Create large schema benchmark (100+ columns) in `tests/benchmark/table_bench_test.go`

### Implementation for User Story 2

- [x] T021 [US2] Optimize JSON serialization path with pooled buffers in `internal/api/handlers/table.go` (completed in Phase 3)
- [x] T022 [P] [US2] Add buffer size optimization for large schemas in `internal/pool/buffer.go` (4KB default, 64KB max)
- [x] T023 [US2] Implement buffer discard policy for oversized buffers (>64KB) (in BufferPool.Put)
- [x] T024 [US2] Run latency benchmarks and verify targets met

**Checkpoint**: Serialization <1ms, avg response <50ms, large schema <200ms

---

## Phase 5: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, monitoring, and production readiness

- [x] T025 [P] Create pool integration test in `tests/integration/pool_integration_test.go`
- [x] T026 [P] Add pool configuration via environment variables (BINGSAN_POOL_BUFFER_INITIAL, BINGSAN_POOL_BUFFER_MAX)
- [x] T027 [P] Document pool usage patterns in `docs/performance.md`
- [x] T028 Create Grafana dashboard queries for pool metrics (in docs/performance.md)
- [x] T029 [P] Add alerting rule examples for low pool hit rate (in docs/performance.md)
- [x] T030 Run quickstart.md validation to verify all success criteria (see benchstat_comparison.md)
- [x] T031 Final benchstat report comparing all metrics to baseline (see benchstat_comparison.md)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-4)**: All depend on Foundational phase completion
  - US1 can proceed immediately after Foundational
  - US2 can proceed in parallel with US1 or after US1
- **Polish (Phase 5)**: Depends on US1 minimum, ideally both user stories complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on US2
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - Independent of US1 but benefits from US1 pool infrastructure

### Within Each User Story

- Benchmarks MUST be created first to establish measurement baseline
- Pool integration follows benchmarks
- Verification against success criteria last
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- All Foundational tasks marked [P] can run in parallel (within Phase 2)
- T004 (BufferPool), T005 (BytePool), T006 (metrics) can run in parallel
- T008 (unit tests) can run in parallel with T006/T007
- Once Foundational phase completes, US1 and US2 benchmarks can run in parallel
- US1 and US2 implementation can proceed in parallel (different handler files)
- All Polish tasks marked [P] can run in parallel

---

## Parallel Example: Phase 2 (Foundational)

```bash
# Launch pool implementations in parallel:
Task: "Implement BufferPool with sync.Pool in internal/pool/buffer.go"
Task: "Implement BytePool for token generation in internal/pool/bytes.go"
Task: "Implement pool metrics in internal/pool/metrics.go"
```

## Parallel Example: User Story 1

```bash
# Launch all US1 benchmarks in parallel:
Task: "Create pool benchmark tests in tests/benchmark/pool_bench_test.go"
Task: "Create memory stability benchmark in tests/benchmark/memory_bench_test.go"

# Launch handler integrations in parallel (different files):
Task: "Integrate BufferPool into table handlers in internal/api/handlers/table.go"
Task: "Integrate BufferPool into view handlers in internal/api/handlers/view.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (baseline benchmarks)
2. Complete Phase 2: Foundational (pool infrastructure)
3. Complete Phase 3: User Story 1 (memory pressure reduction)
4. **STOP and VALIDATE**: Run benchstat comparison, verify 30% allocation reduction
5. Deploy/demo if ready - this alone provides significant value

### Incremental Delivery

1. Complete Setup + Foundational → Pool infrastructure ready
2. Add User Story 1 → Measure improvement → Deploy (MVP!)
3. Add User Story 2 → Measure latency targets → Deploy
4. Each story adds value without breaking previous stories

### Success Criteria Verification

| Metric | Target | Task to Verify |
|--------|--------|----------------|
| Allocation reduction | ≥30% | T017 (benchstat comparison) |
| GC pause p99 | <10ms | T010 (memory stability benchmark) |
| No latency regression | p50 unchanged | T024 (latency benchmarks) |
| Pool hit rate | ≥80% | T016 (metrics endpoint) |
| Memory stability | No growth | T010 (24h load test) |

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Benchmarks establish baseline BEFORE implementation
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Verify success criteria from spec.md at each checkpoint
