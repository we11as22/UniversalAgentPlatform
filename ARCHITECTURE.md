# Architecture

## System Overview

UniversalAgentPlatform is a polyglot, Kubernetes-native, multi-tenant agent platform. The core planes are:

1. Control plane
   - `admin-web`
   - `admin-api`
   - `session-service`
   - `quota-service`
   - `audit-service`
2. Conversation plane
   - `chat-web`
   - `chat-gateway`
   - `conversation-service`
   - `agent-router`
3. Provider and inference plane
   - `provider-gateway`
   - Triton model gateway and model repository
4. Voice plane
   - `voice-gateway`
   - `transcript-service`
   - LiveKit
5. Knowledge and workflow plane
   - `agent-runtime`
   - `rag-service`
   - `indexer`
   - `tool-runner`
   - `workflow-workers`
   - Temporal
6. Observability and analytics plane
   - OpenTelemetry Collector
   - Prometheus
   - Loki
   - Tempo
   - Grafana
   - ClickHouse

Ingress is handled through Envoy Gateway and Gateway API. East-west traffic uses Istio mTLS for mesh-enabled workloads. Stateful platform services run inside the cluster and remain self-hostable.

## Key Decisions

### Triton-only self-hosted inference
**Status:** active  
**Decision:** self-hosted LLM, ASR, and TTS are consumed only via Triton through `provider-gateway`.  
**Why:** preserves provider abstraction, keeps model internals out of business services, and allows model backends to change without changing the platform contract.  
**Trade-off:** GPU validation is a separate path and Triton operations add complexity.

### WebSocket-first text streaming, SSE replay fallback, WebRTC for voice
**Status:** active  
**Decision:** standard chat prefers WebSocket for primary text streaming, keeps SSE as replay and compatibility fallback, and uses LiveKit/WebRTC for realtime voice.  
**Why:** WebSocket handles bidirectional chat control and more resilient browser UX, SSE remains useful for proxy compatibility and deterministic replay by `run_id`, and WebRTC remains the correct transport for duplex media.

### Tenant-first persistence
**Status:** active  
**Decision:** tenant-bearing domain tables use tenant-first access patterns and are designed to be Citus-ready.  
**Why:** tenant isolation and horizontal scale require tenant-aware indexing and distribution.

### Short synchronous chains
**Status:** active  
**Decision:** user-facing request chains stay short; workflows and heavy work leave the sync path quickly.  
**Why:** low latency and resilience degrade fast when request graphs become deep.

### One-command Kubernetes bootstrap
**Status:** active  
**Decision:** `make up-k8s` is the canonical cluster bring-up path.  
**Why:** the platform is meant to be operated on Kubernetes, not treated as a local-only demo that later gets “ported” to K8s.

## Invariants

- **Conversation-agent binding is immutable.**
  A conversation is pinned to one `agent_id`. To switch agent, create or clone a new conversation.
- **Self-hosted inference never bypasses Triton.**
  No direct vLLM/FastAPI/model-specific serving path is allowed for self-hosted models.
- **Provider secret material stays outside database rows.**
  The platform stores `CredentialRef` metadata, not provider secrets.
- **Admin changes are auditable.**
  Admin mutations must emit audit records.
- **Voice transcripts become first-class conversation data.**
  Voice is not a side channel; transcript events are projected into the chat timeline.

## Service Catalog

| Service | Purpose | Sync surface | Async role | Primary scaling concern |
|------|------|------|------|------|
| `chat-gateway` | Browser/API entry for chat and invoke | REST + SSE | emits run lifecycle | concurrent connections and stream fanout |
| `admin-api` | Admin CRUD and operator actions | REST | emits audit/perf events | write bursts from ops automation |
| `conversation-service` | chat/message/run persistence | REST/gRPC-ready | emits conversation events | DB write pressure |
| `agent-router` | resolve agent version and execution chain | REST | orchestration handoff | request routing throughput |
| `provider-gateway` | provider abstraction and fallback | REST/gRPC-ready | health/usage events | provider latency and quota protection |
| `voice-gateway` | voice sessions and transcript flow | REST + LiveKit session bootstrap | voice metrics/events | concurrent realtime sessions |
| `transcript-service` | transcript persistence | REST | transcript projection events | ordered low-latency writes |
| `agent-runtime` | prompt assembly, tool/RAG orchestration | HTTP | workflow handoff | CPU-bound orchestration concurrency |
| `rag-service` | retrieval and citations | HTTP | retrieval metrics | vector latency and QPS |
| `indexer` | ingestion, chunking, vector writes | HTTP | ingestion jobs | background throughput |
| `workflow-workers` | Temporal workers | HTTP | durable workflows | queue depth and retry storms |

## Communication Matrix

| Interaction | Style | Reason |
|------|------|------|
| Browser -> `chat-gateway` | REST/WebSocket/SSE | chat UX, primary token stream delivery, and replay fallback |
| Browser -> `voice-gateway` | REST + LiveKit | voice session creation and WebRTC bootstrap |
| Browser -> `admin-api` | REST | operator CRUD and workflows |
| `chat-gateway` -> `conversation-service` | sync | persist chat state in user path |
| `chat-gateway` -> `agent-router` | sync | short execution dispatch |
| `agent-router` -> `agent-runtime` | sync | low-latency agent execution |
| `agent-runtime` -> `provider-gateway` | sync | single provider abstraction surface |
| `agent-runtime` -> `rag-service` | sync | retrieval before generation |
| services -> Kafka | async | analytics, health, audit, durable replay |
| services -> Temporal | async | long-running workflows and retries |

## Scalability Posture

The platform is designed around the controls needed to hold many users and many agents concurrently with low latency:

- **Load balancing**
  - Envoy Gateway balances ingress traffic.
  - Kubernetes Services load-balance pods.
  - Istio provides service-to-service traffic management inside the mesh.
- **Horizontal scaling**
  - stateless frontends, gateways, router, provider, runtime, RAG, and worker services are horizontally scalable
  - Helm now includes HPA and PDB templates
  - `values-prod.yaml` defines multi-replica/autoscaled defaults for latency-sensitive stateless workloads
- **Queues and decoupling**
  - Kafka is the event backbone
  - Temporal carries durable long-running workflows
  - heavy ingestion and perf orchestration should stay off the chat sync path
- **Caching and hot state**
  - Redis is used for hot counters, quotas, sessions, and other low-latency state
- **Data scaling**
  - PostgreSQL is the transactional source of truth
  - schema/index shape is Citus-ready
  - Qdrant scales retrieval separately from transactional load
  - MinIO scales attachment and artifact storage independently
  - ClickHouse absorbs heavy analytics instead of forcing large analytical reads onto PostgreSQL
- **Backpressure and protection**
  - `quota-service` enforces rate limits and quotas
  - `provider-gateway` is the place for provider fallback and degradation logic
  - HPA, queueing, and separation between sync and async paths reduce latency collapse under load

## Latency-Critical Paths

### Standard text chat

1. Browser sends message to `chat-gateway`
2. `chat-gateway` persists the user message in `conversation-service`
3. `chat-gateway` resolves the current agent version through `admin-api`
4. `chat-gateway` dispatches to `agent-router`
5. `agent-router` calls `agent-runtime`
6. `agent-runtime` optionally retrieves from `rag-service`
7. `agent-runtime` calls `provider-gateway`
8. `provider-gateway` calls external provider or Triton
9. result returns to `chat-gateway`
10. `chat-gateway` persists assistant output and streams it to the browser over WebSocket by preference or SSE fallback

This path stays deliberately short. Expensive or non-user-critical tasks belong in Kafka/Temporal paths, not inline.

### Voice

1. Browser requests voice session from `voice-gateway`
2. `voice-gateway` issues LiveKit session details
3. media flows through LiveKit
4. ASR/TTS traffic is routed through `provider-gateway`
5. transcript events persist through `transcript-service`
6. transcript is projected into the conversation timeline

Voice is separated from text chat at the transport layer so audio concurrency does not pollute the standard text request path.

## UI/UX Navigation Model

- `admin-web` is organized by operator domains: `Overview`, `Agents`, `Providers`, `Models`, `Knowledge`, `Voice`, `Perf`, `Observability`, `Security`
- `chat-web` is organized by user tasks: `Chat`, `Voice`, `Search`, `Settings`, `API`
- every route is now backed by a real Next.js page rather than only client-side tab state
- observability links are exposed directly in the admin cockpit so operators can jump into Grafana, Prometheus, Keycloak, Temporal, MinIO, and Chat UI

## External API Model

Agents are not chat-only. External applications can invoke registered agents through:

- `POST /api/v1/agents/{agent_id}/respond`
- `GET /api/v1/agents/{agent_id}/respond/ws`
- `GET /api/v1/agents/{agent_id}/respond/stream`
- `POST /api/v1/agents/{agent_id}/respond-from-voice`
- `GET /api/v1/perf/runs/{perf_run_id}/results`

That surface uses the same agent registry, LLM/ASR/TTS model bindings, provider routing, and optional RAG logic as the chat application. Text, voice, and realtime-voice agents therefore share one control plane, one observability posture, one perf posture, and one external API strategy.

## External Dependencies

| Dependency | Why it exists |
|------|------|
| PostgreSQL | transactional source of truth |
| Redis | hot state, quotas, session-like low-latency data |
| Kafka | event backbone and replay |
| Temporal | durable workflows and retries |
| Qdrant | vector search |
| MinIO | object storage |
| Triton | unified self-hosted inference plane |
| LiveKit | self-hosted realtime voice transport |
| Prometheus/Loki/Tempo/Grafana | metrics/logs/traces/dashboards |
| ClickHouse | heavy event analytics and product/perf analytics |

## Known Constraints

- full Triton/TensorRT validation requires GPU infrastructure
- `make up-k8s` targets `kind`, so the exposed host ports are fixed to `8088` and `17880/17881`
- external DNS is not required for the one-command bootstrap because the script derives a browser-safe domain automatically
