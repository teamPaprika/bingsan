---
title: "Deployment"
weight: 5
bookCollapseSection: true
---

# Deployment

This section covers deploying Bingsan in various environments.

## Deployment Options

### Development

- [Docker Compose]({{< relref "/docs/deployment/docker" >}}) - Quick local setup

### Production

- [Kubernetes]({{< relref "/docs/deployment/kubernetes" >}}) - Scalable cloud deployment
- [Docker]({{< relref "/docs/deployment/docker" >}}) - Single-node production

## Quick Comparison

| Feature | Docker Compose | Docker | Kubernetes |
|---------|---------------|--------|------------|
| Complexity | Low | Low | High |
| Scalability | Single node | Single node | Multi-node |
| HA | No | No | Yes |
| Monitoring | Manual | Manual | Integrated |
| Best for | Development | Small prod | Large prod |

## Prerequisites

### All Deployments

- PostgreSQL 15+ database
- Object storage (S3/GCS) for production

### Docker

- Docker Engine 20.10+
- Docker Compose v2.0+ (for Compose deployments)

### Kubernetes

- Kubernetes 1.24+
- kubectl configured
- Helm 3.x (optional)

## Common Configuration

Regardless of deployment method, you'll need to configure:

1. **Database**: PostgreSQL connection details
2. **Storage**: S3/GCS credentials and bucket
3. **Auth** (optional): OAuth/API key settings

See [Configuration]({{< relref "/docs/configuration" >}}) for all options.
