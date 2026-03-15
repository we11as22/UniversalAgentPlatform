# WebSocket-first streaming and full agent workflow documentation

**Date:** 2026-03-15  
**Type:** architecture / feature / constraint

## What Changed

The text-streaming path now treats WebSocket as the primary transport for chat and external application streaming while keeping SSE as a replay and compatibility fallback. Conversation-bound WebSocket runs emit `run.started` with `run_id`, which allows the client to continue consuming the same run over `/api/v1/runs/{run_id}/events` if the network degrades after admission.

The performance subsystem now exercises WebSocket text streaming through a dedicated `chat_ws` k6 suite in addition to the existing `chat`, `admin`, and `voice` suites. The documentation set was expanded to describe agent workflows for text, voice, realtime voice, RAG, external-provider, and Triton-backed self-hosted agents.

## Why

The platform is intended for browser and application use under non-ideal network conditions. WebSocket better fits bidirectional chat control, but the platform still needs a deterministic replay path and a proxy-friendly fallback. The old REST-plus-WebSocket chat flow also risked duplicating runs; consolidating around one admitted run with replay closes that gap.

## What This Replaces

This replaces the previous text-streaming posture where SSE was treated as the primary browser transport and the initial WebSocket path risked double-executing a conversation run.

## Watch Out For

Clients should prefer WebSocket for primary streaming, but they should keep the SSE replay path implemented. If a client discards `run_id` from `run.started`, it loses the ability to resume a conversation-bound stream cleanly after a transport interruption.

## Related

- `ARCHITECTURE.md`
- `docs/api.md`
- `docs/external-api-usage.md`
- `docs/perf-testing.md`
- `docs/agent-workflows.md`
