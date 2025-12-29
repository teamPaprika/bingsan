# Feature Specification: Performance Optimization Investigation

**Feature Branch**: `002-perf-optimization`
**Created**: 2025-12-24
**Status**: Draft
**Input**: User description: "Investigate sync.Pool and Protobuf applicability for performance optimization"

## Executive Summary

This specification defines the scope for investigating and implementing performance optimizations in Bingsan, specifically evaluating whether object pooling and binary serialization can improve system throughput and reduce resource consumption.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Platform Engineer Validates Reduced Memory Pressure (Priority: P1)

As a platform engineer deploying Bingsan in production, I need the catalog service to use memory efficiently under high load so that garbage collection pauses don't impact API response times.

**Why this priority**: Memory efficiency directly impacts system stability and response consistency under load. GC pauses can cause request timeouts.

**Independent Test**: Run load test with 10,000 concurrent requests and measure GC pause times and memory allocation rates before and after optimization.

**Acceptance Scenarios**:

1. **Given** the catalog is under sustained high load (1000+ requests/second), **When** processing table metadata operations, **Then** memory allocation rate is reduced by at least 30% compared to baseline
2. **Given** the catalog is under sustained high load, **When** GC runs, **Then** pause times remain under 10ms for 99th percentile
3. **Given** the catalog has been running for 24 hours under load, **When** memory usage is measured, **Then** it remains stable without continuous growth

---

### User Story 2 - Data Engineer Experiences Faster Table Metadata Operations (Priority: P2)

As a data engineer using PyIceberg or Spark to interact with the catalog, I need table metadata operations to complete quickly so my data pipelines don't have unnecessary delays.

**Why this priority**: Table load and commit operations are the most frequent API calls and directly impact data pipeline performance.

**Independent Test**: Benchmark table load and commit operations before and after optimization, measuring p50, p95, and p99 latencies.

**Acceptance Scenarios**:

1. **Given** a client requests table metadata, **When** the response is serialized, **Then** serialization completes in under 1ms for typical table schemas
2. **Given** multiple clients request metadata simultaneously, **When** processing 100 concurrent requests, **Then** average response time remains under 50ms
3. **Given** a table with large schema (100+ columns), **When** loading table metadata, **Then** response time is under 200ms

---

### Edge Cases

- What happens when the object pool is exhausted under extreme load?
- How does the system handle mixed workloads (read-heavy vs write-heavy)?
- What is the memory overhead of maintaining object pools during idle periods?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST reuse allocated memory buffers for JSON serialization operations
- **FR-002**: System MUST return pooled objects to the pool after request completion
- **FR-003**: System MUST handle pool exhaustion gracefully by allocating new objects when necessary
- **FR-004**: System MUST not leak pooled objects (all borrowed objects must be returned)
- **FR-005**: System MUST maintain thread-safety for all pooled resource access
- **FR-006**: System MUST provide metrics for pool utilization (hits, misses, size)

### Non-Functional Requirements

- **NFR-001**: Memory allocation rate for API handlers MUST be reduced by at least 30%
- **NFR-002**: 99th percentile GC pause times MUST remain under 10ms under load
- **NFR-003**: Implementation MUST NOT increase average response latency
- **NFR-004**: Pool overhead during idle periods MUST NOT exceed 10MB

### Out of Scope

- **Protobuf for external API**: The Iceberg REST spec mandates JSON for client communication. Protobuf cannot replace JSON for external APIs without breaking client compatibility.
- **gRPC endpoints**: No internal service-to-service communication exists that would benefit from gRPC/Protobuf.
- **Custom serializers**: The project already uses goccy/go-json which is optimized; custom serializers are unnecessary.

## Applicability Analysis

### sync.Pool - APPLICABLE

**Where it applies**:
1. **JSON encoding buffers**: Table metadata serialization in handlers (table.go, view.go)
2. **Token generation**: Random byte slice allocation in oauth.go
3. **Response builders**: Fiber context response building

**Expected benefit**: 30-40% reduction in allocations for high-frequency operations based on industry benchmarks.

### Protobuf - NOT APPLICABLE

**Why it doesn't apply**:
1. **External API constraint**: Iceberg REST Catalog specification requires JSON for all client communication
2. **No internal services**: Bingsan is a single service with no internal RPC calls
3. **Already optimized**: Using goccy/go-json which provides near-protobuf performance for JSON

**When Protobuf would be applicable**:
- If Bingsan added internal microservices that communicate with each other
- If a non-Iceberg binary protocol was introduced for specialized clients

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Memory allocation rate for table metadata operations reduced by 30% or more (measured via benchmarks)
- **SC-002**: 99th percentile GC pause time under 10ms during sustained 1000 req/s load
- **SC-003**: No increase in p50 response latency (must remain same or improve)
- **SC-004**: Pool hit rate above 80% during sustained load
- **SC-005**: Zero memory leaks after 24-hour load test (stable memory footprint)

## Assumptions

- The goccy/go-json library is already providing optimal JSON performance
- Fiber framework's internal pooling is already optimized
- Primary optimization opportunity is in application-level buffer reuse
- Benchmarks will be run on consistent hardware for valid comparisons

## Dependencies

- Existing benchmark test infrastructure in `tests/benchmark/`
- Load testing capability (via benchmark tests or external tools)
- Memory profiling tools (pprof)
