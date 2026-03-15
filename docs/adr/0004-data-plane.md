# ADR 0004: Data Plane

- **Status:** Accepted
- **Date:** 2026-03-14

## Decision

Use PostgreSQL as the primary transactional store, Redis for hot state, Kafka as the event log, Qdrant for vectors, MinIO for object storage, Temporal for workflows, and ClickHouse for heavy analytics.

## Consequences

- PostgreSQL schemas remain service-owned and Citus-ready.
- Event-driven integrations and durable workflows are first-class patterns rather than optional add-ons.

