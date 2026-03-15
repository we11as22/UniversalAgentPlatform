# Agent matrix, edge proxy, and perf visibility hardening

**Date:** 2026-03-15
**Type:** feature

## What Changed

The platform now treats all agent variants more consistently across admin, API, and chat surfaces. Agent configuration in the admin cockpit now supports LLM, ASR, and TTS model bindings, tool lists, and policy JSON so voice and realtime-voice agents can be configured from the same registry workflow as text agents. The chat workspace now turns push-to-talk voice input into a full conversation run instead of stopping at transcript projection.

The external API surface was tightened as well. Voice-input invocation now supports optional audio payload fields, the SDK exposes a streaming URL helper, and perf runs now expose persisted metric rows through a dedicated endpoint that is surfaced back into the admin `Perf` tab. Optional host-edge Nginx and Caddy layers were also promoted into explicit `make` targets so cloud-hosted deployments can front the platform ingress without inventing ad hoc shell commands.

## Why

The prior state still had gaps between the declared agent matrix and the actual operator/runtime experience. Text agents could accept voice transcripts conceptually, but the browser path did not complete an agent run. Voice agents could be declared, but the admin form did not let operators bind ASR and TTS models in the same lifecycle flow. Perf runs could be queued, but their persisted results were not visible from the cockpit itself.

## What This Replaces

This replaces a narrower admin agent form, transcript-only browser push-to-talk flow, and perf history that only showed queue state without showing result metrics.

## Watch Out For

Voice input still uses the platform ASR contract, so production-grade audio ingestion depends on the configured provider path behind `provider-gateway`. When enabling voice or realtime-voice agents in production, make sure ASR and TTS bindings are actually registered for the tenant instead of relying on demo fallback behaviour.

## Related

- ARCHITECTURE.md: Public APIs
- docs/api.md
- docs/external-api-usage.md
