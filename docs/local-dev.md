# Local Development

## Profiles

- `local-api`
  - CPU-only development profile
  - best for functional work on chat, admin, RAG, API, and control plane
- `gpu`
  - GPU validation profile
  - use when verifying Triton-backed self-hosted inference

## Local Docker Workflow

```bash
cp .env.example .env
make bootstrap
make up-local-api
make smoke
```

Stop everything:

```bash
make down
```

## Local URLs

- Chat UI: `http://localhost:3200`
- Admin UI: `http://localhost:3300`
- Grafana: `http://localhost:13000`
- Keycloak: `http://localhost:18081`
- Temporal UI: `http://localhost:18088`
- MinIO Console: `http://localhost:19001`

## Kubernetes-First Local Workflow

The primary cluster-style local path is:

```bash
make up-k8s
make smoke-k8s
```

This path:

- creates a kind cluster
- installs Istio and Envoy Gateway
- deploys the Helm chart
- exposes browser-accessible URLs on a derived domain

If you do nothing, the script will use either:

- a public `sslip.io` hostname derived from the server IP
- or `uap.localtest.me` as fallback

If you want a specific domain:

```bash
UAP_BASE_DOMAIN=agents.example.internal make up-k8s
```

## Add and Test an Agent

1. Open Admin UI.
2. Go to `Providers`.
3. Create or reuse a provider.
4. Go to `Models`.
5. Register an `llm` model.
6. Go to `Agents`.
7. Create an agent and bind that model.
8. If grounded retrieval is needed, go to `Knowledge` and ingest the content.
9. Open Chat UI and create a new chat with that agent.

## Test the Same Agent Outside Chat

Use the public invoke endpoint:

```bash
curl -X POST http://localhost:3220/api/v1/agents/<agent_id>/respond \
  -H 'Content-Type: application/json' \
  -d '{"message":"Summarise the tenant handbook"}'
```

For Kubernetes, use the printed `api.<base-domain>:8088` URL instead.

## Example RAG Agent

Install the bundled example:

```bash
make install-rag-agent-bundle
```

Then:

1. verify the agent in Admin UI
2. create a new chat with `Bundle RAG Agent` or `Qdrant Knowledge Agent`
3. ask a grounded question
4. call the same agent over the public API endpoint

## GPU Validation

```bash
make up-gpu
```

Use this path only on GPU-enabled infrastructure.
