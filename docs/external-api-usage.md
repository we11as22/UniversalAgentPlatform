# External API Usage

## Why This Exists

Agents on this platform are intended to be reusable platform assets, not UI-local chat bots. The same agent should be callable from:

- chat-web
- internal admin tools
- customer operations portals
- automation systems
- other applications over HTTP API

## Public Invoke Endpoint

```http
POST /api/v1/agents/{agent_id}/respond
```

Additional variants:

- `GET /api/v1/agents/{agent_id}/respond/ws`
- `GET /api/v1/agents/{agent_id}/respond/stream?message=...`
- `POST /api/v1/agents/{agent_id}/respond/stream`
- `POST /api/v1/agents/{agent_id}/respond-from-voice`
- `GET /api/v1/perf/runs/{perf_run_id}/results`

### Minimal request

```bash
curl -X POST http://api.<base-domain>:8088/api/v1/agents/<agent_id>/respond \
  -H 'Content-Type: application/json' \
  -H 'X-Tenant-ID: 11111111-1111-1111-1111-111111111111' \
  -d '{
    "message": "Summarise the tenant handbook"
  }'
```

### Recommended request

```bash
curl -X POST http://api.<base-domain>:8088/api/v1/agents/<agent_id>/respond \
  -H 'Content-Type: application/json' \
  -H 'X-Tenant-ID: 11111111-1111-1111-1111-111111111111' \
  -d '{
    "user_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1",
    "message": "What does the policy say about incident escalation?",
    "metadata": {
      "source": "ops-portal",
      "page": "incident-console"
    }
  }'
```

### Streaming request

```bash
curl -N "http://api.<base-domain>:8088/api/v1/agents/<agent_id>/respond/stream?message=$(python3 - <<'PY'
import urllib.parse
print(urllib.parse.quote('Summarise the tenant handbook'))
PY
)"
```

### Streaming request with JSON body

```bash
curl -N -X POST http://api.<base-domain>:8088/api/v1/agents/<agent_id>/respond/stream \
  -H 'Content-Type: application/json' \
  -d '{
    "message": "Summarise the tenant handbook"
}'
```

### WebSocket streaming request

Use WebSocket as the preferred transport when the consuming application can keep a long-lived socket:

```js
const socket = new WebSocket("ws://api.<base-domain>:8088/api/v1/agents/<agent_id>/respond/ws");

socket.onopen = () => {
  socket.send(JSON.stringify({
    tenant_id: "11111111-1111-1111-1111-111111111111",
    user_id: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1",
    message: "Summarise the tenant handbook",
    metadata: {
      source: "ops-portal"
    }
  }));
};

socket.onmessage = (event) => {
  const payload = JSON.parse(event.data);
  if (payload.type === "message.delta") {
    console.log(payload.payload.delta);
  }
  if (payload.type === "run.completed") {
    socket.close();
  }
};
```

### Voice-input request

```bash
curl -X POST http://api.<base-domain>:8088/api/v1/agents/<agent_id>/respond-from-voice \
  -H 'Content-Type: application/json' \
  -H 'X-Tenant-ID: 11111111-1111-1111-1111-111111111111' \
  -d '{
    "text_hint": "spoken request from browser",
    "audio_base64": "",
    "audio_format": "wav",
    "speak_response": true
  }'
```

## TypeScript SDK

Use the generated SDK helper:

```ts
import { api } from "@uap/ts-sdk";

const result = await api.invokeAgent("http://api.example.internal", agentId, {
  tenant_id: "11111111-1111-1111-1111-111111111111",
  user_id: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1",
  message: "Summarise the handbook"
});

const streamResponse = await api.streamAgent("http://api.example.internal", agentId, {
  message: "Summarise the handbook"
});

const streamUrl = api.streamAgentUrl("http://api.example.internal", agentId, "Summarise the handbook");
const websocketUrl = api.streamAgentWebSocketUrl("http://api.example.internal", agentId);
const voiceResult = await api.invokeAgentFromVoice("http://api.example.internal", agentId, {
  tenant_id: "11111111-1111-1111-1111-111111111111",
  text_hint: "spoken request from browser",
  speak_response: true
});
```

## Platform Pattern

The expected pattern is:

1. define the agent once in Admin UI
2. bind provider/model/policies once
3. optionally bind RAG knowledge once
4. consume the same agent from many applications

This keeps prompts, bindings, observability, and operational control centralized.

## Agent Type Matrix

### Text agents

- non-streaming text input
- WebSocket streaming text input
- streaming text input
- non-streaming voice input
- pre-chat voice bootstrap with automatic conversation creation
- chat UI usage

### Voice agents

- non-streaming text input
- WebSocket streaming text input
- streaming text input
- non-streaming voice input
- pre-chat or in-chat voice session bootstrap
- chat UI usage
- voice session usage

### Realtime voice agents

- same API matrix as voice agents for non-realtime invocation
- realtime interaction through LiveKit/WebRTC

## Fallback Model

The recommended client behavior is:

1. use WebSocket first for text streaming
2. if a conversation-bound stream returns `run.started` with `run_id`, fall back to `/api/v1/runs/{run_id}/events`
3. use non-streaming `respond` when the caller does not need incremental tokens
4. use `respond-from-voice` for text, voice, and realtime voice agents when the caller only has a one-shot spoken request
