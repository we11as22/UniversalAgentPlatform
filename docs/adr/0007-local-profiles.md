# ADR 0007: Local Profiles

- **Status:** Accepted
- **Date:** 2026-03-14

## Decision

Support two local runtime profiles:

- `local-api`: CPU-only, full platform functionality through external/BYO providers.
- `gpu`: Triton-enabled profile for self-hosted inference validation.

## Consequences

- Developers can work without local GPU access.
- GPU validation remains a mandatory acceptance path before production release.

