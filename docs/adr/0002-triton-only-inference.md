# ADR 0002: Triton-Only Self-Hosted Inference

- **Status:** Accepted
- **Date:** 2026-03-14

## Context

Self-hosted inference must remain provider-neutral and swappable without leaking model-specific semantics into product services.

## Decision

All self-hosted LLM, ASR, TTS, and embedding capabilities are exposed only through Triton and consumed only by `provider-gateway`.

## Consequences

- Triton model repositories, health, and quotas become first-class operational concerns.
- Local CPU-only development uses external/BYO providers, but self-hosted acceptance must run through Triton in GPU environments.

