---
title: "Documentation"
weight: 1
bookFlatSection: true
---

# Bingsan Documentation

Welcome to the Bingsan documentation. This guide covers everything you need to know about deploying and using Bingsan as your Apache Iceberg REST Catalog.

## What is Bingsan?

Bingsan is a production-grade Apache Iceberg REST Catalog implementation in Go. It provides:

- Full compliance with the Apache Iceberg REST Catalog OpenAPI specification
- High-performance metadata operations using Go's concurrency model
- PostgreSQL-backed metadata storage with ACID guarantees
- Real-time event streaming via WebSocket
- Cloud-native deployment support (Kubernetes, Docker)

## Documentation Sections

### [Getting Started]({{< relref "/docs/getting-started" >}})
Installation, prerequisites, and quick start guide.

### [API Reference]({{< relref "/docs/api" >}})
Complete documentation of all REST API endpoints.

### [Configuration]({{< relref "/docs/configuration" >}})
Server, database, storage, and authentication configuration options.

### [Architecture]({{< relref "/docs/architecture" >}})
System design, components, and scalability considerations.

### [Performance]({{< relref "/docs/performance" >}})
Object pooling, distributed locking, benchmarking, and tuning guides.

### [Deployment]({{< relref "/docs/deployment" >}})
Production deployment guides for Docker and Kubernetes.
