# Admin Agent Configuration And RAG Bundles

**Date:** 2026-03-14
**Type:** feature

## What Changed

The admin control plane now supports real provider creation, provider-model registration, agent creation with current-version bindings, and direct knowledge indexing into Qdrant. Chat execution was corrected so `chat-gateway` resolves the selected agent's live `current_version_id` from `admin-api`, and `provider-gateway` uses that exact version when selecting the bound LLM model.

RAG execution is no longer globally attempted for every agent. `agent-runtime` now performs retrieval only when the agent configuration enables the `tenant_knowledge_search` path. A built-in install endpoint creates a working `Qdrant Knowledge Agent`, and a separate Docker-packaged bundle under `examples/rag-agent-bundle/` can install another RAG-enabled agent into a running stack.

Local runtime defaults were hardened by moving externally published dependency ports to high values, which avoids host conflicts with existing PostgreSQL, Redis, Kafka, ClickHouse, Keycloak, MinIO, Grafana, and other developer services.

## Why

The original platform bootstrap proved the service topology but still had a correctness gap: newly created agents were not guaranteed to execute with their current version and model bindings, and the local stack still collided with common host ports. The user also needed a concrete, repeatable way to install and test a RAG agent from the admin plane and via a separate Docker artifact.

## What This Replaces

This replaces the previous hard-coded agent version path in `chat-gateway`, the always-on RAG attempt in `agent-runtime`, and the list-only admin registry screens.

## Watch Out For

Agent install and knowledge ingest now rely on `admin-api` reaching `indexer` inside the compose or Kubernetes network. If `INDEXER_URL` is overridden incorrectly, RAG onboarding will fail even though the core chat APIs stay healthy.

## Related

- [README.md](../../README.md)
- [ARCHITECTURE.md](../../ARCHITECTURE.md)
- [docs/local-dev.md](../local-dev.md)
