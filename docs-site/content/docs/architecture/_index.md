---
title: "Architecture"
weight: 4
bookCollapseSection: true
---

# Architecture

Understanding Bingsan's architecture helps with deployment planning, performance tuning, and troubleshooting.

## Overview

Bingsan is a stateless Go application that implements the Apache Iceberg REST Catalog specification. All persistent state is stored in PostgreSQL.

```
┌─────────────────────────────────────────────────────────────┐
│                      Clients                                │
│         (Spark, Trino, Flink, PyIceberg, etc.)             │
└─────────────────────────┬───────────────────────────────────┘
                          │ REST API (HTTP)
┌─────────────────────────▼───────────────────────────────────┐
│                    Bingsan Cluster                          │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐                     │
│  │ Node 1  │  │ Node 2  │  │ Node N  │  (Stateless)        │
│  │  :8181  │  │  :8181  │  │  :8181  │                     │
│  └────┬────┘  └────┬────┘  └────┬────┘                     │
│       │            │            │                           │
│       └────────────┼────────────┘                           │
│                    │ Distributed Locking                    │
└────────────────────┼────────────────────────────────────────┘
                     │
        ┌────────────┴────────────┐
        │                         │
┌───────▼───────┐       ┌────────▼────────┐
│  PostgreSQL   │       │   S3 / GCS      │
│  (Metadata)   │       │   (Data Lake)   │
└───────────────┘       └─────────────────┘
```

## Key Components

### HTTP Server

- Built on [Fiber](https://gofiber.io/) (fasthttp)
- High-performance, low-memory HTTP handling
- Supports HTTP/1.1 with keep-alive

### Database Layer

- PostgreSQL for all metadata storage
- Connection pooling via pgx/v5
- Automatic schema migrations
- Advisory locks for distributed locking

### Storage Integration

- Generates storage paths for tables
- Vends credentials for client data access
- Supports S3, GCS, and local filesystem

### Event Streaming

- WebSocket-based real-time events
- Publish/subscribe model
- Namespace-level filtering

## Design Principles

### Stateless Nodes

Each Bingsan instance is stateless:

- All state in PostgreSQL
- No inter-node communication
- Any node can handle any request
- Easy horizontal scaling

### Optimistic Concurrency

Table commits use optimistic concurrency control:

1. Client reads current metadata
2. Client submits changes with requirements
3. Server validates requirements against current state
4. If valid, changes are applied atomically

### Distributed Locking

PostgreSQL advisory locks prevent concurrent modifications:

- Namespace-level locks for CRUD
- Table-level locks for commits
- Automatic lock release on timeout

## Sections

- [Request Flow]({{< relref "/docs/architecture/request-flow" >}}) - How requests are processed
- [Data Model]({{< relref "/docs/architecture/data-model" >}}) - Database schema and metadata storage
- [Scalability]({{< relref "/docs/architecture/scalability" >}}) - Scaling strategies and limits
