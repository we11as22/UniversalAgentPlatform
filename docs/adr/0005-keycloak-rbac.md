# ADR 0005: Keycloak and RBAC

- **Status:** Accepted
- **Date:** 2026-03-14

## Decision

Use Keycloak for SSO/OIDC, keep tenant membership and platform authorization data in PostgreSQL, and derive runtime claims through a shared auth package.

## Consequences

- Identity can federate with enterprise IdPs.
- Authorization remains platform-owned and auditable.

