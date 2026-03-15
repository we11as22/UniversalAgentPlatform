# Agent Workflows

## Goal

This document shows the intended end-to-end operator and consumer workflows for the main agent types supported by the platform.

## 1. Text Agent

Use when the agent primarily answers in text but still needs one-shot voice input.

### Admin workflow

1. Open `Admin -> Providers` and register the provider.
2. Open `Admin -> Models` and register the LLM model.
3. Open `Admin -> Agents`.
4. Set:
   - modality = `text`
   - LLM binding = selected provider model
   - ASR/TTS bindings = optional
5. Save config, tools, and policies.

### Consumer workflow

- Chat UI:
  - create a new chat
  - send text
  - or start voice session and use push-to-talk for transcript-driven invocation
- External app:
  - sync: `POST /api/v1/agents/{agent_id}/respond`
  - streaming: `GET /api/v1/agents/{agent_id}/respond/ws`
  - one-shot spoken request: `POST /api/v1/agents/{agent_id}/respond-from-voice`

## 2. Voice Agent

Use when the agent needs ASR/TTS but not always a full duplex realtime room.

### Admin workflow

1. Bind LLM.
2. Bind ASR.
3. Bind TTS.
4. Set modality = `voice`.
5. Save policies and tools.

### Consumer workflow

- Chat UI:
  - normal text chat still works
  - voice session bootstrap works
  - one-shot voice-to-response flow works
- External app:
  - `respond`
  - `respond/ws`
  - `respond/stream`
  - `respond-from-voice`

## 3. Realtime Voice Agent

Use when the agent should participate in a LiveKit/WebRTC room.

### Admin workflow

1. Bind LLM, ASR, and TTS.
2. Set modality = `realtime_voice`.
3. Verify the agent appears in `Admin -> Voice`.
4. Use `Admin -> Observability` to open voice and provider dashboards.

### Consumer workflow

- Chat UI:
  - bootstrap voice session
  - switch into realtime room flow
  - transcripts persist into the conversation timeline
- External app:
  - use non-realtime APIs for one-shot requests
  - use `voice-gateway` + LiveKit for true realtime duplex behavior

## 4. RAG Agent

Use when the agent must search tenant or agent knowledge in Qdrant.

### Admin workflow

1. Create the base agent.
2. Enable `rag_enabled`.
3. Use `Admin -> Knowledge` to ingest content.
4. Confirm the agent appears in the RAG subset and install the example bundle if needed.

### Consumer workflow

- Chat UI:
  - create a chat with that agent
  - ask grounded questions
- External app:
  - use the same invoke endpoints
  - inspect the `retrieval` block in the response

## 5. External Provider Agent

Use when the model is not self-hosted and should be called through a BYO or external provider.

### Admin workflow

1. Register provider with credential ref metadata, not secret plaintext.
2. Register provider model.
3. Bind model to the agent.

## 6. Self-Hosted Triton Agent

Use when LLM, ASR, or TTS is self-hosted.

### Invariant

Self-hosted models are only exposed through Triton and only consumed by the platform through `provider-gateway`.

### Admin workflow

1. Register the Triton-backed provider.
2. Register provider models for `llm`, `asr`, or `tts`.
3. Bind them to the agent by modality.

## 7. Rollout Validation

For any agent type:

1. Open `Admin -> Perf`.
2. Run `validation-short`.
3. Inspect:
   - `chat.*`
   - `chat_ws.*`
   - `admin.*`
   - `voice.*`
4. Open Grafana from `Admin -> Observability`.
5. Confirm provider health, latency, and error posture before broader rollout.
