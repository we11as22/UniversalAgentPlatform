# Runbooks

## Platform Bring-Up

### Kubernetes

```bash
make up-k8s
```

Check status:

```bash
kubectl -n uap get pods
kubectl -n uap get httproutes
kubectl -n uap get svc
make smoke-k8s
```

### Local Docker

```bash
make up-local-api
make smoke
```

## Agent Onboarding Runbook

1. Open Admin UI.
2. Go to `Providers` and register the provider.
3. Go to `Models` and register the provider model.
4. Go to `Agents` and create the agent.
5. Set modality and bind:
   - LLM for all agents
   - ASR/TTS for voice-capable agents
6. Set tools, config, and policies.
5. If RAG is needed, go to `Knowledge` and ingest content.
6. Open Chat UI and create a new chat with the agent.
7. Validate the same agent from the public API endpoint.

## Example RAG Agent Runbook

```bash
make install-rag-agent-bundle
```

Then:

1. open Admin UI
2. verify the agent exists
3. verify knowledge was indexed
4. test via Chat UI
5. test via `POST /api/v1/agents/{agent_id}/respond`

## Observability Runbook

Use the Admin UI `Observability` tab first. It contains direct launchers for:

- Platform Overview
- Chat Pipeline
- Voice Pipeline
- Provider Health
- Triton Inference
- Data Plane
- Agent Overview
- Tenant Overview
- Load Testing Results
- Cost / Usage / Latency

Supporting systems are also linked directly:

- Grafana
- Prometheus
- Keycloak
- Temporal
- MinIO
- Chat UI

## Scale Verification Runbook

1. confirm HPA objects exist
   - `kubectl -n uap get hpa`
2. confirm PDB objects exist
   - `kubectl -n uap get pdb`
3. open Grafana Platform Overview
4. run perf profiles from Admin UI
5. verify:
   - request rate
   - p95/p99 latency
   - `chat_ws` session duration and completion
   - stream-start latency
   - provider error rate
   - DB saturation
   - queue lag
   - cache hit posture

## Failure Handling Runbook

### Provider degradation

1. open `Provider Health`
2. identify failing provider or rising latency
3. switch/bind a different provider model if necessary
4. verify external applications using `POST /api/v1/agents/{agent_id}/respond` continue to receive responses

### Voice degradation

1. open `Voice Pipeline`
2. check LiveKit reachability
3. verify transcript persistence
4. confirm fallback to text mode remains available

### Chat latency regression

1. open `Chat Pipeline`
2. check `chat-gateway`, `agent-router`, `agent-runtime`, and `provider-gateway` latencies
3. inspect Prometheus and data-plane dashboards
4. run `smoke` perf profile to validate the current baseline
