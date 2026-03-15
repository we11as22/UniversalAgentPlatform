# Cloud-safe Kubernetes bootstrap, external agent API, and production hardening pass

**Date:** 2026-03-15  
**Type:** architecture / feature / constraint

## What Changed

The Kubernetes bootstrap path now derives a browser-safe public domain automatically for cloud-server scenarios, wires that domain into Helm routing, and updates Keycloak redirect URIs after deployment. The platform can now be tested from a browser on another machine without hand-editing DNS or rewriting URLs in the UI.

The public API surface was expanded with a first-class external agent invocation endpoint so platform agents can be consumed outside the chat UI. Admin and chat applications were also switched to route-backed navigation, and the documentation set was expanded into a more complete production handbook.

The Helm chart now includes HPA and PodDisruptionBudget templates with production-oriented autoscaling defaults for stateless and latency-sensitive workloads.

## Why

The platform is intended to be an actual shared agent platform, not a UI-bound chat demo. Cloud-hosted server usage, browser-based validation, and multi-application consumption are primary operating modes, so the bootstrap, contracts, and docs had to reflect that explicitly.

## What This Replaces

This replaces the previous hard-coded `uap.localtest.me` assumption in the Kubernetes path, the thin public API contract, and the partial documentation set.

## Watch Out For

- `make up-k8s` still targets `kind`, so exposed host ports remain fixed to `8088` and `17880/17881`
- GPU Triton validation remains a separate path because the current host profile is CPU-only

## Related

- [README.md](/root/asudakov/projects/UniversalAgentPlatform/README.md)
- [ARCHITECTURE.md](/root/asudakov/projects/UniversalAgentPlatform/ARCHITECTURE.md)
- [deployment.md](/root/asudakov/projects/UniversalAgentPlatform/docs/deployment.md)
- [external-api-usage.md](/root/asudakov/projects/UniversalAgentPlatform/docs/external-api-usage.md)
- [scaling.md](/root/asudakov/projects/UniversalAgentPlatform/docs/scaling.md)
