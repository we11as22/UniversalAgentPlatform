# ADR 0003: Streaming and Voice Transports

- **Status:** Accepted
- **Date:** 2026-03-14

## Decision

- Text chat responses stream through SSE.
- Realtime voice uses WebRTC through LiveKit.

## Rationale

SSE fits one-way token streams and enterprise ingress paths. LiveKit provides a production-grade self-hosted media plane for full-duplex audio, interruptions, and session orchestration.

