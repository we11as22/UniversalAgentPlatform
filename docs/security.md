# Security

## Baseline Controls

The platform ships with these baseline controls:

- tenant-first data model
- Keycloak-based auth integration
- RBAC-ready control plane
- secret indirection through credential refs
- audit logging path for admin mutations
- mesh-aware internal traffic model
- network policy manifests
- PodDisruptionBudget and autoscaling support for production delivery

## Secret Handling

Provider credentials are not stored in database plaintext. The control plane stores references such as:

- env var locator
- file locator
- Kubernetes secret locator
- vault-style locator

The operational expectation is:

1. DB stores metadata
2. runtime resolves secrets from the configured secret source
3. provider-gateway is the policy boundary for provider access

## Auth and Identity

- Keycloak is the IdP and token issuer
- chat and admin UIs are separate clients
- redirect URIs are updated for the active Kubernetes public domain during bootstrap

## Tenant Isolation

Tenant isolation is enforced by:

- tenant-bound records
- tenant-first access patterns
- tenant-scoped API lookups
- tenant awareness in chat, admin, and RAG flows

## Logging and Audit

Admin changes must remain auditable. Observability surfaces are not a substitute for immutable audit events; they are a complement.

## Production Hardening Recommendations

For long-lived production environments, add or tighten:

- real TLS at ingress
- external secret manager integration
- stricter Keycloak policies and identity flows
- explicit network egress controls
- node isolation for Triton/GPU workloads
- backup and retention policies for PostgreSQL, MinIO, Qdrant, ClickHouse
