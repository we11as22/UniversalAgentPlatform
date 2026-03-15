# API

## API Surfaces

The platform exposes four main interaction modes:

- REST for public and admin APIs
- WebSocket for primary text-streaming chat and external application streaming
- SSE for replayable text-streaming fallback
- WebRTC/LiveKit for realtime voice
- Kafka/gRPC-ready contracts for internal and async propagation

## Public REST Endpoints

### Chat and agent use

- `GET /api/v1/agents`
- `POST /api/v1/agents/{agent_id}/respond`
- `GET /api/v1/agents/{agent_id}/respond/ws`
- `GET /api/v1/agents/{agent_id}/respond/stream?message=...`
- `POST /api/v1/agents/{agent_id}/respond/stream`
- `POST /api/v1/agents/{agent_id}/respond-from-voice`
- `GET /api/v1/conversations`
- `POST /api/v1/conversations`
- `GET /api/v1/conversations/search?q=...`
- `GET /api/v1/conversations/{conversation_id}/messages`
- `POST /api/v1/conversations/{conversation_id}/runs`
- `GET /api/v1/conversations/{conversation_id}/runs/ws`
- `GET /api/v1/runs/{run_id}/events`
- `POST /api/v1/files/upload`

### Admin and control plane

- `GET /api/v1/providers`
- `POST /api/v1/providers`
- `GET /api/v1/provider-models`
- `POST /api/v1/provider-models`
- `GET /api/v1/agents`
- `GET /api/v1/agents/{agent_id}`
- `POST /api/v1/agents`
- `POST /api/v1/agents/install/rag-example`
- `POST /api/v1/knowledge/index`
- `GET /api/v1/dashboard`
- `GET /api/v1/perf/profiles`
- `GET /api/v1/perf/runs`
- `GET /api/v1/perf/runs/{perf_run_id}/results`
- `POST /api/v1/perf/runs`

## External Application Integration

The public agent invocation endpoint is:

```http
POST /api/v1/agents/{agent_id}/respond
```

Additional public variants:

```http
GET /api/v1/agents/{agent_id}/respond/stream?message=...
POST /api/v1/agents/{agent_id}/respond/stream
POST /api/v1/agents/{agent_id}/respond-from-voice
```

Example request:

```json
{
  "tenant_id": "11111111-1111-1111-1111-111111111111",
  "user_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1",
  "message": "Summarise the tenant handbook",
  "metadata": {
    "source": "backoffice-portal",
    "trace_label": "customer-onboarding"
  }
}
```

Example response:

```json
{
  "agent_id": "f3c66b74-2f54-4fdc-b2f7-123456789abc",
  "agent_version_id": "d8af6b65-c08f-40a7-94f1-123456789abc",
  "tenant_id": "11111111-1111-1111-1111-111111111111",
  "user_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1",
  "provider_name": "acme-demo-provider",
  "provider_kind": "demo",
  "rag_enabled": true,
  "text": "The tenant handbook states ...",
  "retrieval": {
    "source_count": 2
  }
}
```

This endpoint is intended for:

- internal portals
- backoffice applications
- automation entry points
- support tooling
- embedded knowledge assistants

Voice-input example for any agent type:

```json
{
  "text_hint": "spoken request from browser",
  "audio_base64": "<optional-base64-audio>",
  "audio_format": "wav",
  "speak_response": true
}
```

## Streaming Contract

Text chat streams primarily over:

```http
GET /api/v1/agents/{agent_id}/respond/ws
GET /api/v1/conversations/{conversation_id}/runs/ws
```

Event types:

- `run.started`
- `message.delta`
- `stream.heartbeat`
- `run.completed`
- `run.failed`

For conversation-bound WebSocket runs, `run.started` now carries `run_id`. That allows the client to fall back to:

```http
GET /api/v1/runs/{run_id}/events
```

when the network degrades after the run has already been admitted and persisted.

External application streaming is also available over:

```http
GET /api/v1/agents/{agent_id}/respond/stream?message=...
```

Or with a JSON body:

```http
POST /api/v1/agents/{agent_id}/respond/stream
```

SSE remains the deterministic compatibility and replay transport. WebSocket is the preferred primary transport for browser chat and external bidirectional clients.

## Voice Contract

Voice is handled through `voice-gateway` plus LiveKit:

- `POST /api/v1/voice/sessions`
- `POST /api/v1/voice/transcribe`

`voice-gateway` accepts either an existing `conversation_id` or will bootstrap one automatically for pre-chat voice flows. The browser obtains session metadata from `voice-gateway`, then uses LiveKit/WebRTC for transport.

## OpenAPI and SDK

- OpenAPI source: [platform.openapi.yaml](/root/asudakov/projects/UniversalAgentPlatform/packages/schemas/openapi/platform.openapi.yaml)
- TypeScript SDK: [index.ts](/root/asudakov/projects/UniversalAgentPlatform/packages/ts-sdk/src/index.ts)

## Multi-Application Usage Pattern

To reuse one agent across many applications:

1. Create the agent in Admin UI.
2. Bind its provider/model and policies there.
3. Enable RAG if required.
4. Call the public invoke endpoint from each consuming app.
5. Keep agent prompt/model changes centralized in the platform instead of duplicating them in every client.
