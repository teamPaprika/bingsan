---
title: "Bingsan - Apache Iceberg REST Catalog"
type: docs
---

# Bingsan

**Bingsan** is a high-performance Apache Iceberg REST Catalog implemented in Go. It uses PostgreSQL as the metadata store and leverages Go's concurrency model to provide high throughput and low latency.

## Key Features

- **Complete Iceberg REST API**: Compliant with Apache Iceberg REST Catalog OpenAPI spec
- **PostgreSQL Backend**: Reliable metadata storage with ACID transactions
- **Go Concurrency**: Efficient request handling using goroutines
- **Real-time Event Streaming**: Receive catalog change events via WebSocket
- **Prometheus Metrics**: Metrics endpoint for operational monitoring
- **Kubernetes-Friendly**: Supports single-node or multi-node cluster deployments

## Quick Links

- **[Getting Started]({{< relref "/docs/getting-started/quick-start" >}})** - Get up and running quickly with Bingsan
- **[API Reference]({{< relref "/docs/api" >}})** - Complete API documentation for all endpoints
- **[Configuration Guide]({{< relref "/docs/configuration" >}})** - Learn how to configure Bingsan for your environment

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Clients                                │
│         (Spark, Trino, Flink, PyIceberg, etc.)             │
└─────────────────────────┬───────────────────────────────────┘
                          │ REST API
┌─────────────────────────▼───────────────────────────────────┐
│                    Bingsan Cluster                          │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐                     │
│  │ Node 1  │  │ Node 2  │  │ Node N  │  (Kubernetes Pods)  │
│  │  :8181  │  │  :8181  │  │  :8181  │                     │
│  └────┬────┘  └────┬────┘  └────┬────┘                     │
│       └────────────┼────────────┘                           │
└────────────────────┼────────────────────────────────────────┘
                     │
        ┌────────────┴────────────┐
        │                         │
┌───────▼───────┐       ┌────────▼────────┐
│  PostgreSQL   │       │   S3 / GCS      │
│  (Metadata)   │       │   (Data Lake)   │
└───────────────┘       └─────────────────┘
```

## Compatibility

Bingsan is compatible with:

- Apache Spark (Iceberg Spark runtime)
- Trino / Presto
- Apache Flink
- PyIceberg
- Dremio
- StarRocks

It is designed as a drop-in replacement for existing Java REST catalogs like Apache Polaris, Gravitino, Nessie, and Unity Catalog.

## License

Apache License 2.0
