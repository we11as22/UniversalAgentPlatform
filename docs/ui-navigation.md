# UI Navigation

## Admin Cockpit

Admin UI is organized by platform operator tasks, not by backend table names.

### Overview

- posture summary
- quick actions
- launcher cards into Grafana and supporting systems

### Agents

- inspect current agent inventory
- create agents
- choose modality per agent: `text`, `voice`, `realtime_voice`
- bind LLM, ASR, and TTS models where appropriate
- set config, tools, and policy JSON per agent
- enable RAG
- install example RAG agent

### Providers

- register self-hosted or external providers
- keep credentials indirect

### Models

- register LLM, ASR, TTS, or embedding models
- bind concrete models without leaking provider internals into agent logic

### Knowledge

- index knowledge into Qdrant
- test RAG onboarding flows

### Voice

- see voice-capable agents
- jump directly into voice monitoring

### Perf

- launch perf profiles
- inspect recent runs
- confirm metrics from `chat`, `chat_ws`, `admin`, and `voice`

### Observability

- one-click links into all major dashboards
- links into Grafana, Prometheus, Keycloak, Temporal, MinIO, Chat UI

### Security

- platform invariants and baseline controls
- fast links to identity and core dashboards

## Chat Cockpit

### Chat

- create new chat
- select agent
- stream response
- clone conversation into another agent

### Voice

- focus on realtime/push-to-talk testing
- validate transcript projection and voice monitoring

### Search

- filter conversation history
- jump into the right thread quickly

### Settings

- inspect environment posture and key endpoints

### API

- see how to invoke agents from other applications

## Navigation Validation Standard

Every primary area should:

- be reachable by direct route
- have a visible purpose
- support keyboard navigation
- keep operational links explicit
- avoid dead-end screens

## Validation Snapshot

The current route validation baseline is:

- Admin: `/`, `/agents`, `/providers`, `/models`, `/knowledge`, `/voice`, `/perf`, `/observability`, `/security`
- Chat: `/`, `/voice`, `/search`, `/settings`, `/api-usage`

All of these are expected to return `200` and render the same shell/navigation system.
