# UniversalAgentPlatform

UniversalAgentPlatform is a Kubernetes-first enterprise agent platform for on-prem or cloud-hosted deployments. It provides a ChatGPT-style chat UI, an operator-grade admin cockpit, voice-ready agent flows, Triton-only self-hosted inference, RAG, observability, and built-in performance testing in one monorepo.

## What It Solves

This project is for teams that need to run many enterprise agents for many users without hard-wiring business logic to one model vendor or one chat surface. The platform centralizes agent definitions, provider/model bindings, RAG knowledge, quotas, monitoring, and performance validation so agents can be consumed from chat, admin workflows, or external applications through API.

It is designed for:

- on-prem or controlled-cloud deployments
- multi-tenant internal platforms
- teams that need self-hosted inference behind Triton
- teams that need both browser chat and external API consumers
- teams that care about scale, auditability, and observability

## Quick Start

### Canonical Kubernetes path

```bash
git clone <repo>
cd UniversalAgentPlatform
make up-k8s
```

This is the primary bootstrap path. It:

1. installs missing CLI tooling
2. creates a `kind` cluster
3. installs Istio and Envoy Gateway
4. builds and loads platform images
5. deploys the Helm chart
6. derives a browser-safe public domain automatically
7. updates Keycloak redirect URIs
8. runs smoke checks

If `UAP_BASE_DOMAIN` is not set, the bootstrap script tries to derive a public `sslip.io` hostname from the server’s public IP. That makes browser testing from another machine work without manual DNS setup.

### Local Docker path

```bash
cp .env.example .env
make bootstrap
make up-local-api
make smoke
```

### GPU validation path

```bash
make up-gpu
```

Use this path when validating Triton-backed self-hosted LLM/ASR/TTS on GPU infrastructure.

## How It Works

The platform is split into control plane, conversation plane, provider plane, voice plane, workflow plane, and performance plane:

- `admin-web` and `admin-api` manage agents, providers, models, knowledge, perf runs, and operator workflows
- `chat-web` and `chat-gateway` handle user chat, WebSocket-first text streaming with SSE replay fallback, voice bootstrap, uploads, and external agent invocation
- `agent-router`, `agent-runtime`, `provider-gateway`, `rag-service`, `tool-runner`, and `workflow-workers` execute agent logic
- `voice-gateway` and `transcript-service` coordinate LiveKit/WebRTC voice and transcript persistence
- PostgreSQL, Redis, Kafka, Temporal, Qdrant, MinIO, ClickHouse, Prometheus, Loki, Tempo, and Grafana provide the data and operational substrate

Self-hosted inference is never called directly by business services. All self-hosted LLM, ASR, and TTS traffic goes through Triton and is abstracted behind `provider-gateway`.

Agent consumption is not limited to chat. The platform now supports:

- non-streaming agent API
- WebSocket streaming agent API
- streaming agent API over SSE
- POST-body streaming agent API for server-side consumers
- non-streaming voice-input API for text, voice, and realtime voice agents
- optional host-edge fronting through Nginx or Caddy

## Main Workflows

### Add and test an agent

1. Open Admin UI.
2. Register a provider.
3. Register one or more provider models.
4. Create an agent and bind the LLM model.
5. Optionally enable RAG and ingest knowledge.
6. Open Chat UI and create a new chat with that agent.
7. Or call the same agent from another application through:
   - `POST /api/v1/agents/{agent_id}/respond`
   - `GET /api/v1/agents/{agent_id}/respond/ws`
   - `GET /api/v1/agents/{agent_id}/respond/stream`
   - `POST /api/v1/agents/{agent_id}/respond/stream`
   - `POST /api/v1/agents/{agent_id}/respond-from-voice`
   - `GET /api/v1/perf/runs/{perf_run_id}/results`

### Configure any agent type from Admin UI

1. Open `Agents`.
2. Set modality: `text`, `voice`, or `realtime_voice`.
3. Bind one LLM model for every agent.
4. Bind ASR and TTS models for `voice` or `realtime_voice` agents.
5. Set tools, config, and policies directly in the same form.
6. Enable RAG when the agent must retrieve from Qdrant-backed knowledge.
7. Use `Knowledge` to ingest tenant or agent knowledge.
8. Use `Perf` and `Observability` to validate rollout and latency before exposing the agent to users.

### Install the example RAG agent

```bash
make install-rag-agent-bundle
```

Then open Admin UI, verify the seeded agent and knowledge, and test it from Chat UI or external API.

### Launch perf validation

1. Open Admin UI.
2. Go to `Perf`.
3. Launch `validation-short`, `smoke`, `load`, `stress`, `spike`, or `soak`.
4. Open the Grafana load-testing dashboard from the same cockpit.

### Optional edge proxies

```bash
make up-edge-caddy
make up-edge-nginx
make down-edge
```

Use these only when you want an outer host-level reverse proxy in front of the platform ingress. Envoy Gateway remains the canonical ingress inside Kubernetes.

For the Docker path, these edge proxies now route by hostname to the correct local surfaces:

- `chat.<base-domain>` -> chat-web
- `admin.<base-domain>` -> admin-web
- `api.<base-domain>` -> chat-gateway
- `admin-api.<base-domain>` -> admin-api
- `grafana.<base-domain>` -> Grafana
- `prometheus.<base-domain>` -> Prometheus
- `keycloak.<base-domain>` -> Keycloak
- `temporal.<base-domain>` -> Temporal UI
- `minio.<base-domain>` -> MinIO Console
- `livekit.<base-domain>` -> LiveKit

## Primary URLs

### Local Docker

| Surface | URL |
|------|------|
| Chat UI | `http://localhost:3200` |
| Admin UI | `http://localhost:3300` |
| Grafana | `http://localhost:13000` |
| Keycloak | `http://localhost:18081` |
| Temporal UI | `http://localhost:18088` |
| MinIO Console | `http://localhost:19001` |

### Kubernetes

`make up-k8s` prints the exact URLs it derived for your server. Typical output is:

| Surface | URL pattern |
|------|------|
| Chat UI | `http://chat.<base-domain>:8088` |
| Admin UI | `http://admin.<base-domain>:8088` |
| Chat API | `http://api.<base-domain>:8088` |
| Admin API | `http://admin-api.<base-domain>:8088` |
| Grafana | `http://grafana.<base-domain>:8088` |
| Prometheus | `http://prometheus.<base-domain>:8088` |
| Keycloak | `http://keycloak.<base-domain>:8088` |
| Temporal UI | `http://temporal.<base-domain>:8088` |
| MinIO Console | `http://minio.<base-domain>:8088` |
| LiveKit | `ws://livekit.<base-domain>:17880` |

## Key Entry Points

| What | Where |
|------|------|
| Architecture and invariants | `ARCHITECTURE.md` |
| Kubernetes deployment | `docs/deployment.md` |
| Local development | `docs/local-dev.md` |
| External API usage | `docs/external-api-usage.md` |
| Agent workflows | `docs/agent-workflows.md` |
| Scaling and production posture | `docs/scaling.md` |
| UI and navigation guide | `docs/ui-navigation.md` |
| Runbooks and operations | `docs/runbooks.md` |
| Troubleshooting | `docs/troubleshooting.md` |

## Configuration

| Variable | Default | Description |
|------|------|------|
| `UAP_BASE_DOMAIN` | auto-derived | Base domain for browser-accessible Kubernetes endpoints |
| `UAP_PUBLIC_SCHEME` | `http` | Public scheme for generated URLs |
| `DATABASE_URL` | local default | Primary PostgreSQL DSN |
| `REDIS_URL` | local default | Redis DSN |
| `KAFKA_BROKERS` | local default | Kafka bootstrap servers |
| `KEYCLOAK_URL` | local default | Keycloak base URL |
| `LIVEKIT_URL` | local default | LiveKit WebSocket URL |
| `QDRANT_URL` | local default | Qdrant endpoint |
| `TRITON_ENDPOINT` | local default | Triton HTTP endpoint |

## Further Reading

- [ARCHITECTURE.md](ARCHITECTURE.md)
- [docs/deployment.md](docs/deployment.md)
- [docs/api.md](docs/api.md)
- [docs/external-api-usage.md](docs/external-api-usage.md)
- [docs/agent-workflows.md](docs/agent-workflows.md)
- [docs/scaling.md](docs/scaling.md)
- [docs/security.md](docs/security.md)
- [docs/perf-testing.md](docs/perf-testing.md)
- [docs/changes/](docs/changes/)
