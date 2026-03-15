# Scaling

## Goal

The platform is designed to serve many users across many agents with low latency by separating traffic classes, keeping sync paths short, and pushing heavy work into the right substrate.

## Scaling Model

### Ingress and balancing

- Envoy Gateway is the north-south entry point
- Kubernetes Services load-balance pods
- Istio manages east-west service traffic for mesh-enabled workloads

### Stateless horizontal scale

These workloads are intended to scale horizontally:

- `chat-gateway`
- `admin-api`
- `session-service`
- `conversation-service`
- `agent-router`
- `provider-gateway`
- `voice-gateway`
- `quota-service`
- `audit-service`
- `transcript-service`
- `agent-runtime`
- `tool-runner`
- `rag-service`
- `indexer`
- `workflow-workers`
- `chat-web`
- `admin-web`

Helm now includes:

- HPA template
- PodDisruptionBudget template
- production autoscaling defaults in `values-prod.yaml`

### Stateful scale separation

- PostgreSQL handles transactions
- Redis handles hot state
- Kafka handles event backlog
- Temporal handles workflows
- Qdrant handles vector load
- MinIO handles objects and artifacts
- ClickHouse handles heavy analytics

This prevents any one store from carrying every access pattern.

## Queueing and Backpressure

### What exists

- Kafka for async event decoupling
- Temporal for durable long-running work
- quota/rate-limit service for admission control
- provider-gateway as the choke point for provider fallback and degradation

### Why this matters

Without these controls, bursty ingestion, perf runs, large document processing, or slow providers would bleed directly into chat latency. The architecture isolates those concerns.

## Low-Latency Rules

- do not deepen user-facing synchronous request chains without strong reason
- keep chat and voice traffic separated at the transport layer
- prefer WebSocket for primary text streaming and keep SSE replay available by `run_id`
- keep provider-specific complexity inside `provider-gateway`
- use Redis for hot counters and short-lived state
- use async workflows for non-immediate work

## Validation Checklist

To validate that the platform is holding scale correctly:

1. run a perf profile
2. inspect `Platform Overview`
3. inspect `Chat Pipeline`
4. inspect `Provider Health`
5. inspect `Data Plane`
6. confirm HPA reacts
7. confirm p95/p99 stay within target
8. confirm queue lag and DB latency remain bounded
