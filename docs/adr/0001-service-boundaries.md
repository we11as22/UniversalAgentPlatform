# ADR 0001: Service Boundaries

- **Status:** Accepted
- **Date:** 2026-03-14

## Context

The platform spans chat, admin, routing, voice, retrieval, providers, quotas, and analytics. A monolith would make on-prem scaling and capability isolation harder.

## Decision

Use a microservice architecture with Go services for latency-critical APIs and Python services for orchestration, tools, and retrieval. Service ownership is:

- Go: `chat-gateway`, `admin-api`, `session-service`, `conversation-service`, `agent-router`, `provider-gateway`, `voice-gateway`, `quota-service`, `audit-service`, `transcript-service`
- Python: `agent-runtime`, `tool-runner`, `rag-service`, `indexer`, `workflow-workers`

## Consequences

- Independent deployability and scaling by plane.
- Stronger contracts required through OpenAPI, gRPC, and event schemas.
- More operational surface, offset by platform tooling and observability.

