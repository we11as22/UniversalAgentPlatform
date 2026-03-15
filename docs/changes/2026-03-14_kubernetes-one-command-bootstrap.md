# Kubernetes One-Command Bootstrap

**Date:** 2026-03-14  
**Type:** architecture

## What Changed

The platform now has a real Kubernetes bootstrap path instead of placeholder manifests. A new `kind` profile, umbrella Helm chart, cluster bootstrap scripts, Gateway routes, mesh policy, image build/load flow, and Kubernetes smoke checks were added so the repository can be brought up with `make up-k8s` after clone.

The Helm chart now deploys the platform workloads and self-hosted dependencies in-cluster, provisions file-backed config maps for Postgres bootstrap, Keycloak realm import, LiveKit, Temporal, observability, and Grafana dashboards, and exposes stable local endpoints through Envoy Gateway. LiveKit is exposed directly through mapped kind ports for browser voice transport.

## Why

The original repository had a strong Docker Compose path but an almost empty Kubernetes layer. That violated the platform goal and made the declared Kubernetes-first architecture aspirational rather than real. The new bootstrap closes that gap and makes the cluster path operational from the repo itself.

## What This Replaces

This replaces the previous minimal `infra/k8s` skeleton and near-empty platform chart that only created a namespace.

## Watch Out For

The one-command bootstrap is intentionally heavy because it builds and loads all local images. On fresh hosts the first run will take significantly longer than later runs. GPU/Triton validation remains a separate `values-gpu.yaml` path because the current host class is CPU-only.

## Related

- [README.md](/root/asudakov/projects/UniversalAgentPlatform/README.md)
- [ARCHITECTURE.md](/root/asudakov/projects/UniversalAgentPlatform/ARCHITECTURE.md)
- [docs/deployment.md](/root/asudakov/projects/UniversalAgentPlatform/docs/deployment.md)
