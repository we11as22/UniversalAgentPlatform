# ADR 0006: Helm and ArgoCD Deployment Model

- **Status:** Accepted
- **Date:** 2026-03-14

## Decision

Helm charts are the canonical deployment artifact. Raw Kubernetes manifests and ArgoCD applications derive from the same values and templating strategy.

## Consequences

- One source of truth for deploy-time configuration.
- GitOps overlays stay aligned with local and CI rendering.

