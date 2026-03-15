# Cockpit UI Redesign

**Date:** 2026-03-14
**Type:** feature

## What Changed

The frontend was reworked from a basic dark CRUD shell into a proper operator cockpit. Shared UI primitives now provide a stronger shell, panels, badges, tabs and action cards. Both `admin-web` and `chat-web` were redesigned around the same visual system with better hierarchy, more deliberate typography, stronger surfaces, and clearer operational entry points.

The admin experience is now explicitly organized by domain: overview, agents, providers, models, knowledge, voice, perf, observability and security. Each tab maps to an operator use-case instead of a generic page dump. The observability zone includes direct launchers into provisioned Grafana dashboards and supporting systems like Keycloak, Prometheus, MinIO and Temporal.

The chat workspace was also upgraded into a clearer cockpit: agent posture, conversation rail, streaming transcript, voice session state and observability jumps are all visible without collapsing the main conversation flow.

## Why

The platform already had the functional path for chat, admin and RAG onboarding, but the interface still looked like an engineering scaffold. For a platform intended to be run by enterprise operators, the UI itself needs to support decision-making speed, reduce navigation friction and make monitoring entry points obvious.

## What This Replaces

This replaces the earlier minimal shell, flat cards and list-first admin views that required more context switching and did not expose operational domains or monitoring paths clearly.

## Watch Out For

The observability tab now links to specific Grafana dashboard UIDs. If dashboard provisioning paths or UIDs change, the admin launchers must be updated together with the dashboard files.

## Related

- [README.md](../../README.md)
- [docs/local-dev.md](../local-dev.md)
- [ARCHITECTURE.md](../../ARCHITECTURE.md)
