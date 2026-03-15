# Runnable Local Stack Bootstrap

**Date:** 2026-03-14
**Type:** feature

## What Changed

The repository now includes runnable Go services, Python services, Next.js applications, Dockerfiles, compose environments, PostgreSQL schema migrations, seed data, performance scripts, CI definitions, and packaging for the first end-to-end local platform path.

## Why

Architecture-only scaffolding was not sufficient for the requested platform. The repository needed executable services, images, and a local stack that can be validated with real builds and compose startup attempts.

## What This Replaces

The prior state where only architecture and workspace foundation existed.

## Watch Out For

The local stack still depends on host port availability. Qdrant now defaults to a safer high port, but other services may still need environment-specific overrides in crowded development machines.

## Related

- `infra/docker-compose/compose.base.yml`
- `apps/chat-web/`
- `apps/admin-web/`
- `services/`

